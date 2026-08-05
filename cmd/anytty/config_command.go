package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/anytty/anytty/shared/filepublish"
	"github.com/anytty/anytty/shared/securefs"
	tuiconfig "github.com/anytty/anytty/tui/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	return tuiconfig.DefaultPath()
}

func newConfigPathsCommand(configPath, socket, logFile *string) *cobra.Command {
	var jsonOutput bool
	type pathsView struct {
		Config  string `json:"config"`
		Socket  string `json:"socket"`
		Log     string `json:"log"`
		History string `json:"history"`
	}
	command := &cobra.Command{
		Use: "paths", Short: "Show resolved configuration and runtime paths", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			view := pathsView{Config: effectiveConfigPath(*configPath), Socket: resolveV3Socket(*socket), Log: resolveV3LogFilePath(*logFile), History: resolveV3HistoryStorageDir()}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(view)
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Config", Value: view.Config},
				cliField{Label: "Socket", Value: view.Socket},
				cliField{Label: "Log", Value: view.Log},
				cliField{Label: "History", Value: view.History},
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
				config, err := tuiconfig.Load(pathIfExplicitOrExisting(*configPath, path), os.Getenv)
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
				if err := yaml.Unmarshal(data, &value); err != nil {
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
			var output bytes.Buffer
			encoder := yaml.NewEncoder(&output)
			encoder.SetIndent(2)
			if err := encoder.Encode(node); err != nil {
				return err
			}
			_ = encoder.Close()
			_, err = io.WriteString(cmd.OutOrStdout(), strings.TrimSpace(output.String())+"\n")
			return err
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

func loadConfigDocument(path string, allowMissing bool) (*yaml.Node, []byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && allowMissing {
		data = []byte("version: 1\n")
	} else if err != nil {
		return nil, nil, err
	}
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(false)
	if err := decoder.Decode(&document); err != nil {
		return nil, nil, &cliError{code: 2, message: err.Error(), cause: err}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, nil, usageCLIError("config contains multiple YAML documents")
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, nil, usageCLIError("config root must be a mapping")
	}
	return &document, data, nil
}

func configKeyParts(key string) []string {
	parts := strings.Split(strings.TrimSpace(key), ".")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func configNodeAt(document *yaml.Node, parts []string) (*yaml.Node, bool) {
	if document == nil || len(document.Content) != 1 || len(parts) == 0 {
		return nil, false
	}
	current := document.Content[0]
	for _, part := range parts {
		if part == "" || current.Kind != yaml.MappingNode {
			return nil, false
		}
		next, _, found := mappingValue(current, part)
		if !found {
			return nil, false
		}
		current = next
	}
	return current, true
}

func setConfigNode(document *yaml.Node, parts []string, value *yaml.Node) error {
	if len(parts) == 0 {
		return fmt.Errorf("config key cannot be empty")
	}
	current := document.Content[0]
	for index, part := range parts {
		if part == "" {
			return fmt.Errorf("config key contains an empty path component")
		}
		existing, valueIndex, found := mappingValue(current, part)
		if index == len(parts)-1 {
			if found {
				current.Content[valueIndex] = value
			} else {
				current.Content = append(current.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part}, value)
			}
			return nil
		}
		if !found {
			existing = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			current.Content = append(current.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part}, existing)
		} else if existing.Kind != yaml.MappingNode {
			return fmt.Errorf("config key %s is not a mapping", strings.Join(parts[:index+1], "."))
		}
		current = existing
	}
	return nil
}

func unsetConfigNode(document *yaml.Node, parts []string) bool {
	if len(parts) == 0 || document == nil || len(document.Content) != 1 {
		return false
	}
	return unsetConfigMappingNode(document.Content[0], parts)
}

func unsetConfigMappingNode(mapping *yaml.Node, parts []string) bool {
	value, valueIndex, found := mappingValue(mapping, parts[0])
	if !found {
		return false
	}
	if len(parts) == 1 {
		mapping.Content = append(mapping.Content[:valueIndex-1], mapping.Content[valueIndex+1:]...)
		return true
	}
	if value.Kind != yaml.MappingNode || !unsetConfigMappingNode(value, parts[1:]) {
		return false
	}
	// 自下而上删除 mutation 产生的空父 map，避免严格 runtime parser 接收到无语义 section。
	if len(value.Content) == 0 {
		mapping.Content = append(mapping.Content[:valueIndex-1], mapping.Content[valueIndex+1:]...)
	}
	return true
}

func mappingValue(mapping *yaml.Node, key string) (*yaml.Node, int, bool) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil, 0, false
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1], index + 1, true
		}
	}
	return nil, 0, false
}

func parseConfigValue(value string) (*yaml.Node, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(value), &document); err != nil {
		return nil, err
	}
	if len(document.Content) != 1 {
		return nil, fmt.Errorf("config value is empty")
	}
	return document.Content[0], nil
}

func validateAndWriteConfig(path string, document *yaml.Node) error {
	// YAML AST 只负责结构化编辑；运行时 strict parser 仍是字段、类型和取值合法性的唯一真值。
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if _, err := tuiconfig.Parse(output.Bytes()); err != nil {
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
	if _, err := temporary.Write(output.Bytes()); err != nil {
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
