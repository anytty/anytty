package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anytty/anytty/tui/state"
)

func refreshPromptCompletions(root state.Root, shell state.ShellStore) state.ShellStore {
	shell = shell.EnsureDefaults()
	shell = syncCreatePromptWorkdirForServer(root, shell)
	prompt := shell.Overlay.Prompt
	field := prompt.ActivePromptField()
	if field == nil {
		return shell.ClearPromptSuggestions()
	}
	switch strings.TrimSpace(field.Key) {
	case "workdir":
		if !promptWorkdirCompletionUsesLocal(root, prompt) {
			return shell.ClearPromptSuggestions()
		}
		title, items, empty := workdirSuggestionPopup(field.Value, field.Cursor)
		return shell.SetActivePromptSuggestions(title, items, empty)
	case "server":
		if prompt.Purpose != "terminal.create" {
			return shell.ClearPromptSuggestions()
		}
		// 中文说明：server 下拉只消费 reducer-owned endpoint 投影；
		// 选择结果仍通过 TerminalPoolCreateRequestMsg.EndpointID 回到 owning daemon。
		title, items, empty := terminalCreateEndpointSuggestionPopup(root, field.Value)
		return shell.SetActivePromptSuggestions(title, items, empty)
	default:
		return shell.ClearPromptSuggestions()
	}
}

func promptWorkdirCompletionUsesLocal(root state.Root, prompt state.PromptState) bool {
	if prompt.Purpose != "terminal.create" {
		return true
	}
	return false
}

func terminalCreateEndpointSuggestionPopup(root state.Root, value string) (string, []string, string) {
	options := terminalCreateEndpointSuggestionItems(root)
	if len(options) == 0 {
		return "servers", nil, "(no available servers)"
	}
	query := strings.TrimSpace(value)
	if query == "" {
		return "servers", options, ""
	}
	if endpointID, ok := terminalCreateEndpointIDFromValue(root, query); ok {
		return "servers", reorderPromptSuggestions(options, terminalCreateEndpointPromptValue(root, endpointID)), ""
	}
	queryLower := strings.ToLower(query)
	filtered := make([]string, 0, len(options))
	for _, option := range options {
		if strings.Contains(strings.ToLower(option), queryLower) {
			filtered = append(filtered, option)
		}
	}
	if len(filtered) == 0 {
		return "servers", nil, "(no matching servers)"
	}
	return "servers", filtered, ""
}

func terminalCreateEndpointSuggestionItems(root state.Root) []string {
	endpoints := state.TerminalCreateEndpointItems(root)
	items := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		items = append(items, terminalCreateEndpointPromptValue(root, endpoint.ID))
	}
	return items
}

func reorderPromptSuggestions(items []string, current string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	current = strings.TrimSpace(current)
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), current) {
			out = append(out, item)
			break
		}
	}
	for _, item := range items {
		duplicate := false
		for _, existing := range out {
			if strings.EqualFold(existing, item) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			out = append(out, item)
		}
	}
	return out
}

func workdirSuggestionPopup(value string, cursor int) (string, []string, string) {
	runes := []rune(value)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	prefix := string(runes[:cursor])
	if strings.TrimSpace(prefix) == "" {
		return "", nil, ""
	}
	baseDisplay, baseResolved, fragment, ok := workdirCompletionBase(prefix)
	if !ok {
		return "", nil, ""
	}
	title := "path: " + baseResolved
	entries, err := os.ReadDir(baseResolved)
	if err != nil {
		return title, nil, "(path not found)"
	}
	fragmentLower := strings.ToLower(fragment)
	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(fragment, ".") {
			continue
		}
		if fragmentLower != "" && !strings.HasPrefix(strings.ToLower(name), fragmentLower) {
			continue
		}
		label := name + string(filepath.Separator)
		if baseDisplay != "" {
			label = baseDisplay + label
		}
		items = append(items, label)
	}
	sort.Strings(items)
	if len(items) == 0 {
		return title, nil, "(no matching directories)"
	}
	return title, items, ""
}

func workdirCompletionBase(prefix string) (string, string, string, bool) {
	home, _ := os.UserHomeDir()
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", "", false
	}
	switch {
	case prefix == "~":
		if strings.TrimSpace(home) == "" {
			return "", "", "", false
		}
		return "~/", home, "", true
	case strings.HasPrefix(prefix, "~/"):
		rest := strings.TrimPrefix(prefix, "~/")
		base, fragment := splitPathPrefix(rest)
		return "~/" + base, filepath.Join(home, filepath.FromSlash(base)), fragment, true
	case strings.HasPrefix(prefix, "/"):
		base, fragment := splitPathPrefix(strings.TrimPrefix(prefix, "/"))
		return "/" + base, filepath.Join(string(filepath.Separator), filepath.FromSlash(base)), fragment, true
	default:
		base, fragment := splitPathPrefix(prefix)
		return base, filepath.Join(cwd, filepath.FromSlash(base)), fragment, true
	}
}

func splitPathPrefix(prefix string) (string, string) {
	lastSlash := strings.LastIndex(prefix, "/")
	if lastSlash < 0 {
		return "", prefix
	}
	return prefix[:lastSlash+1], prefix[lastSlash+1:]
}
