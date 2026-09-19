package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tuiconfig "github.com/anytty/anytty/clients/tui/config"
	"github.com/anytty/anytty/shared/filepublish"
	"github.com/anytty/anytty/shared/securefs"
	"github.com/spf13/cobra"
)

func newConfigCommand(configPath, socket, logFile *string) *cobra.Command {
	command := &cobra.Command{Use: "config", Short: "Inspect and modify the AnyTTY client configuration"}
	command.AddCommand(
		newConfigPathsCommand(configPath, socket, logFile),
		newConfigShowCommand(configPath),
		newConfigGetCommand(configPath),
		newConfigSetCommand(configPath),
		newConfigUnsetCommand(configPath),
		newConfigValidateCommand(configPath),
	)
	return command
}

func effectiveConfigPath(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return filepath.Clean(explicit)
	}
	return tuiconfig.Path()
}

func newConfigPathsCommand(configPath, socket, logFile *string) *cobra.Command {
	var jsonOutput bool
	type pathsView struct {
		Config    string `json:"config"`
		Socket    string `json:"socket"`
		Log       string `json:"log"`
		History   string `json:"history"`
		Clipboard string `json:"clipboard"`
	}
	command := &cobra.Command{
		Use: "paths", Short: "Show resolved configuration and runtime paths", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			view := pathsView{
				Config: effectiveConfigPath(*configPath), Socket: resolveV3Socket(*socket), Log: resolveV3LogFilePath(*logFile),
				History: resolveV3HistoryStorageDir(), Clipboard: resolveV3ClipboardStoragePath(),
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(view)
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Config", Value: view.Config},
				cliField{Label: "Socket", Value: view.Socket},
				cliField{Label: "Log", Value: view.Log},
				cliField{Label: "History", Value: view.History},
				cliField{Label: "Clipboard", Value: view.Clipboard},
			)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newConfigShowCommand(configPath *string) *cobra.Command {
	var effective, jsonOutput bool
	command := &cobra.Command{
		Use: "show", Short: "Show source or effective configuration", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := effectiveConfigPath(*configPath)
			if effective {
				config, err := tuiconfig.Load(pathIfExplicitOrExisting(*configPath, path))
				if err != nil {
					return &cliError{code: 2, message: err.Error(), cause: err}
				}
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(config)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return &cliError{code: 3, message: fmt.Sprintf("config file %s does not exist; use `anytty config set` to create it", path), cause: err}
				}
				return err
			}
			if jsonOutput {
				var value any
				if err := json.Unmarshal(data, &value); err != nil {
					return usageCLIError(err.Error())
				}
				return json.NewEncoder(cmd.OutOrStdout()).Encode(value)
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
	command.Flags().BoolVar(&effective, "effective", false, "show defaults and environment overrides")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print source configuration as JSON")
	return command
}

func pathIfExplicitOrExisting(explicit, resolved string) string {
	if strings.TrimSpace(explicit) != "" {
		return resolved
	}
	if _, err := os.Stat(resolved); err == nil {
		return resolved
	}
	return ""
}

func newConfigGetCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "get KEY", Short: "Read one source configuration value", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			document, _, err := loadConfigDocument(effectiveConfigPath(*configPath), false)
			if err != nil {
				return err
			}
			node, found := configNodeAt(document, configKeyParts(args[0]))
			if !found {
				return &cliError{code: 3, message: fmt.Sprintf("config key %s was not found", args[0])}
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(node)
		},
	}
}

func newConfigSetCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "set KEY VALUE", Short: "Atomically set one configuration value", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := effectiveConfigPath(*configPath)
			document, _, err := loadConfigDocument(path, true)
			if err != nil {
				return err
			}
			value, err := parseConfigValue(args[1])
			if err != nil {
				return usageCLIError(err.Error())
			}
			if err := setConfigNode(document, configKeyParts(args[0]), value); err != nil {
				return usageCLIError(err.Error())
			}
			if err := validateAndWriteConfig(path, document); err != nil {
				return err
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Key", Value: args[0]},
				cliField{Label: "Status", Value: "updated"},
			)
		},
	}
}

func newConfigUnsetCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "unset KEY", Short: "Atomically remove one source configuration value", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := effectiveConfigPath(*configPath)
			document, _, err := loadConfigDocument(path, false)
			if err != nil {
				return err
			}
			if !unsetConfigNode(document, configKeyParts(args[0])) {
				return &cliError{code: 3, message: fmt.Sprintf("config key %s was not found", args[0])}
			}
			if err := validateAndWriteConfig(path, document); err != nil {
				return err
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Key", Value: args[0]},
				cliField{Label: "Status", Value: "unset"},
			)
		},
	}
}

func newConfigValidateCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use: "validate [FILE]", Short: "Validate a configuration with the runtime parser", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := effectiveConfigPath(*configPath)
			if len(args) == 1 {
				path = filepath.Clean(args[0])
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := tuiconfig.Parse(data); err != nil {
				return &cliError{code: 2, message: err.Error(), cause: err}
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Config", Value: path},
				cliField{Label: "Status", Value: "valid"},
			)
		},
	}
}

// loadConfigDocument 读取 JSON 配置为通用文档树；allowMissing 时缺文件按空
// 对象处理（`config set` 首次创建配置）。
func loadConfigDocument(path string, allowMissing bool) (map[string]any, []byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && allowMissing {
		data = []byte("{}")
	} else if err != nil {
		return nil, nil, err
	}
	document := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return nil, nil, &cliError{code: 2, message: err.Error(), cause: err}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, nil, usageCLIError("config contains multiple JSON documents")
	}
	return document, data, nil
}

func configKeyParts(key string) []string {
	parts := strings.Split(strings.TrimSpace(key), ".")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func configNodeAt(document map[string]any, parts []string) (any, bool) {
	if len(parts) == 0 {
		return nil, false
	}
	var current any = document
	for _, part := range parts {
		if part == "" {
			return nil, false
		}
		mapping, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = mapping[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func setConfigNode(document map[string]any, parts []string, value any) error {
	if len(parts) == 0 {
		return fmt.Errorf("config key cannot be empty")
	}
	current := document
	for index, part := range parts {
		if part == "" {
			return fmt.Errorf("config key contains an empty path component")
		}
		if index == len(parts)-1 {
			current[part] = value
			return nil
		}
		existing, found := current[part]
		if !found {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}
		child, ok := existing.(map[string]any)
		if !ok {
			return fmt.Errorf("config key %s is not an object", strings.Join(parts[:index+1], "."))
		}
		current = child
	}
	return nil
}

func unsetConfigNode(document map[string]any, parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	return unsetConfigMappingNode(document, parts)
}

func unsetConfigMappingNode(mapping map[string]any, parts []string) bool {
	value, found := mapping[parts[0]]
	if !found {
		return false
	}
	if len(parts) == 1 {
		delete(mapping, parts[0])
		return true
	}
	child, ok := value.(map[string]any)
	if !ok || !unsetConfigMappingNode(child, parts[1:]) {
		return false
	}
	if len(child) == 0 {
		delete(mapping, parts[0])
	}
	return true
}

// parseConfigValue 优先按 JSON 标量/结构解析，失败时按字符串处理。
func parseConfigValue(value string) (any, error) {
	var decoded any
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return value, nil
	}
	// 只接受完整 JSON 值；"12h" 这类字面量必须按字符串处理而不是截断成数字。
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return value, nil
	}
	return decoded, nil
}

func validateAndWriteConfig(path string, document map[string]any) error {
	output, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	output = append(output, '\n')
	// JSON 树只负责结构化编辑；运行时 strict parser 仍是字段、类型和取值合法性的唯一真值。
	if _, err := tuiconfig.Parse(output); err != nil {
		return &cliError{code: 2, message: err.Error(), cause: err}
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	if err := securefs.SecureDirectory(parent); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".anytty-config-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := securefs.SecureFile(temporaryPath); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(output); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	// 同目录 temporary + rename 保证读者只观察到完整旧版本或完整新版本。
	if err := filepublish.Rename(temporaryPath, path); err != nil {
		return err
	}
	return filepublish.SyncDirectory(parent)
}
