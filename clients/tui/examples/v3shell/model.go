package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/app"
	apipb "github.com/anytty/anytty/proto/access/apipb"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	gproto "google.golang.org/protobuf/proto"
)

// ---------------------------------------------------------------- geometry

// rect is one solved rectangle in viewport cells.
type rect struct {
	x, y, w, h int
}

// ------------------------------------------------------------- split tree

// pane is one leaf of a tab's split tree: a card plus its content state.
type pane struct {
	id               string
	title            string
	sourceID         string
	detachedSourceID string
	lines            []string
	scroll           int
	locked           bool
	pending          string
}

// leaf is one pane card in the recursive split tree.
type leaf struct {
	pane *pane
	rect rect
}

// split is a recursive split node: orient "row" (a | b side by side) or
// "col" (a above b). ratio is a's share of the split extent (only honored
// while 0 < ratio < 1); bias is the additive cell bias the resize scene writes
// on top of the default/ratio first extent, matching the legacy SplitNode
// Ratio/BiasCells contract. Resizing clears ratio so bias is authoritative.
type split struct {
	orient string
	ratio  float64
	bias   int
	a, b   treeNode
	seq    int
	rect   rect
}

// splitFirstExtent ports render.splitFirstExtent: the first child's extent in
// cells. Precedence is ratio (when 0 < ratio < 1) over the even default half,
// then the additive bias, clamped to leave one cell for the second child. A
// demo/golden split (ratio 0.5, bias 0) still yields total/2, so the geometry
// is byte-identical.
func (sp *split) splitFirstExtent(total int) int {
	if total <= 1 {
		return total
	}
	first := total / 2
	if sp.ratio > 0 && sp.ratio < 1 {
		first = int(float64(total) * sp.ratio)
	}
	first += sp.bias
	return clampInt(first, 1, total-1)
}

type treeNode interface{ isTreeNode() }

func (*leaf) isTreeNode()  {}
func (*split) isTreeNode() {}

// tab is one tab: a split tree plus the in-order pane list and focus index.
type tab struct {
	id       string
	title    string
	panes    []*pane
	root     treeNode
	focus    int
	splitSeq int
}

func makeTab(id, title string, panes []*pane, flow string) *tab {
	t := &tab{id: id, title: title}
	var leaves []*leaf
	for _, p := range panes {
		leaves = append(leaves, &leaf{pane: p})
	}
	var root treeNode
	if len(leaves) > 0 {
		root = leaves[0]
	}
	for _, l := range leaves[1:] {
		t.splitSeq++
		root = &split{orient: flow, ratio: 0.5, a: root, b: l, seq: t.splitSeq}
	}
	t.root = root
	t.rebuild()
	return t
}

func (t *tab) nextSplitSeq() int { t.splitSeq++; return t.splitSeq }

func leafNodes(node treeNode) []*leaf {
	switch n := node.(type) {
	case nil:
		return nil
	case *leaf:
		return []*leaf{n}
	case *split:
		return append(leafNodes(n.a), leafNodes(n.b)...)
	}
	return nil
}

func (m *model) handleCreatePromptKey(key, char string) app.Cmd {
	m.ensurePromptCursors()
	if len(m.promptFields) == 0 {
		return nil
	}
	field := clampInt(m.promptField, 0, len(m.promptFields)-1)
	m.promptField = field
	if m.promptSuggestionFocused {
		return m.handleCreateSuggestionKey(key, char)
	}
	value := []rune(m.promptFields[field])
	cursor := clampInt(m.promptCursors[field], 0, len(value))
	if key != "enter" {
		m.promptError = ""
	}
	switch key {
	case "esc":
		m.overlay = ""
		m.promptKind = "command"
		m.promptPublicTagList = false
		m.promptError = ""
	case "up", "shift-tab":
		m.clearPromptSuggestions()
		m.promptField = clampInt(m.promptField-1, 0, len(m.promptFields)-1)
		return m.refreshCreateSuggestions(false)
	case "down", "tab":
		if key == "tab" && len(m.promptSuggestions) > 0 && m.promptSuggestionField == field {
			m.promptSuggestionFocused = true
			return nil
		}
		if field == 2 && key == "tab" {
			if m.focusCreateSuggestions() {
				return nil
			}
		}
		// Workdir completion is endpoint-scoped and arrives asynchronously. The
		// old prompt keeps the field active while Tab starts that request; once
		// the result arrives, the candidate list receives focus instead of the
		// form silently moving to tags.
		if field == 3 && key == "tab" && strings.TrimSpace(promptPathPrefix(string(value), cursor)) != "" && m.promptEndpoint != "" && m.client != nil {
			return m.refreshCreateSuggestions(true)
		}
		m.clearPromptSuggestions()
		m.promptField = clampInt(m.promptField+1, 0, len(m.promptFields)-1)
		return m.refreshCreateSuggestions(false)
	case "left":
		m.promptCursors[field] = clampInt(cursor-1, 0, len(value))
		return m.refreshCreateSuggestions(false)
	case "right":
		m.promptCursors[field] = clampInt(cursor+1, 0, len(value))
		return m.refreshCreateSuggestions(false)
	case "home":
		m.promptCursors[field] = 0
		return m.refreshCreateSuggestions(false)
	case "end":
		m.promptCursors[field] = len(value)
		return m.refreshCreateSuggestions(false)
	case "enter":
		return m.submitCreatePrompt()
	case "backspace":
		m.clearPromptSuggestionFocus()
		if cursor > 0 {
			value = append(value[:cursor-1], value[cursor:]...)
			m.promptFields[field] = string(value)
			m.promptCursors[field] = cursor - 1
		}
		return m.refreshCreateSuggestions(false)
	case "delete":
		m.clearPromptSuggestionFocus()
		if cursor < len(value) {
			value = append(value[:cursor], value[cursor+1:]...)
			m.promptFields[field] = string(value)
		}
		return m.refreshCreateSuggestions(false)
	default:
		m.clearPromptSuggestionFocus()
		if char != "" && len([]rune(char)) > 0 && !strings.HasPrefix(key, "ctrl-") {
			insert := []rune(char)
			next := make([]rune, 0, len(value)+len(insert))
			next = append(next, value[:cursor]...)
			next = append(next, insert...)
			next = append(next, value[cursor:]...)
			value = next
			m.promptFields[field] = string(value)
			m.promptCursors[field] = cursor + len(insert)
		}
		return m.refreshCreateSuggestions(false)
	}
	return nil
}

func (m *model) clearPromptSuggestions() {
	m.promptSuggestions = nil
	m.promptSuggestionTitle = ""
	m.promptSuggestionEmpty = ""
	m.promptSuggestionSel = 0
	m.promptSuggestionOffset = 0
	m.promptSuggestionField = -1
	m.promptSuggestionValue = ""
	m.promptSuggestionCursor = 0
	m.promptSuggestionEndpoint = ""
	m.promptSuggestionFocusRequested = false
	m.promptSuggestionFocused = false
}

func (m *model) clearPromptSuggestionFocus() {
	m.promptSuggestionFocused = false
	m.promptSuggestionSel = 0
	m.promptSuggestionOffset = 0
}

func (m *model) insertCreatePromptText(text string) app.Cmd {
	if text == "" || len(m.promptFields) == 0 {
		return nil
	}
	m.ensurePromptCursors()
	field := clampInt(m.promptField, 0, len(m.promptFields)-1)
	m.promptField = field
	m.clearPromptSuggestionFocus()
	value := []rune(m.promptFields[field])
	cursor := clampInt(m.promptCursors[field], 0, len(value))
	insert := []rune(text)
	updated := make([]rune, 0, len(value)+len(insert))
	updated = append(updated, value[:cursor]...)
	updated = append(updated, insert...)
	updated = append(updated, value[cursor:]...)
	m.promptFields[field] = string(updated)
	m.promptCursors[field] = cursor + len(insert)
	m.promptError = ""
	return m.refreshCreateSuggestions(false)
}

func (m *model) createEndpointPromptValue(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	for _, tab := range m.pickerTabs() {
		if tab.name == endpoint {
			if tab.label != "" && tab.label != tab.name {
				return fmt.Sprintf("%s (%s)", tab.label, tab.name)
			}
			return tab.name
		}
	}
	return endpoint
}

func (m *model) createEndpointSuggestions(value string) []string {
	trimmed := strings.TrimSpace(value)
	// A recognized endpoint opens the complete dropdown with the current
	// endpoint first. Partial text filters the available endpoint labels.
	if endpoint, ok := m.normalizeCreateEndpoint(trimmed); ok {
		all := make([]string, 0, len(m.pickerTabs()))
		for _, tab := range m.pickerTabs() {
			all = append(all, m.createEndpointPromptValue(tab.name))
		}
		current := m.createEndpointPromptValue(endpoint)
		for index, option := range all {
			if strings.EqualFold(option, current) {
				all[0], all[index] = all[index], all[0]
				break
			}
		}
		return all
	}
	query := strings.ToLower(trimmed)
	var suggestions []string
	for _, tab := range m.pickerTabs() {
		option := m.createEndpointPromptValue(tab.name)
		if query == "" || strings.Contains(strings.ToLower(option), query) || strings.EqualFold(tab.name, strings.TrimSpace(value)) || strings.EqualFold(tab.label, strings.TrimSpace(value)) {
			suggestions = append(suggestions, option)
		}
	}
	return suggestions
}

func (m *model) setPromptSuggestions(field int, title string, suggestions []string, empty string, focus bool) {
	m.promptSuggestionField = field
	m.promptSuggestionTitle = title
	m.promptSuggestionEmpty = empty
	m.promptSuggestions = append([]string(nil), suggestions...)
	m.promptSuggestionSel = clampInt(m.promptSuggestionSel, 0, maxInt(0, len(suggestions)-1))
	m.promptSuggestionOffset = promptSuggestionOffsetForSelection(m.promptSuggestionOffset, m.promptSuggestionSel, len(suggestions))
	if focus && len(suggestions) > 0 {
		m.promptSuggestionFocused = true
	}
}

func promptSuggestionOffsetForSelection(offset, selected, count int) int {
	if count <= promptSuggestionVisibleRows {
		return 0
	}
	maxOffset := count - promptSuggestionVisibleRows
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	if selected < offset {
		return selected
	}
	if selected >= offset+promptSuggestionVisibleRows {
		return selected - promptSuggestionVisibleRows + 1
	}
	return offset
}

func promptPathParent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	separator := "/"
	if strings.Contains(value, "\\") && !strings.Contains(value, "/") {
		separator = "\\"
	}
	trimmed := strings.TrimRight(value, "/\\")
	if trimmed == "" {
		return value
	}
	index := strings.LastIndex(trimmed, separator)
	if index < 0 {
		return ""
	}
	return trimmed[:index+1]
}

func promptPathPrefix(value string, cursor int) string {
	runes := []rune(value)
	cursor = clampInt(cursor, 0, len(runes))
	return string(runes[:cursor])
}

func (m *model) refreshCreateSuggestions(focus bool) app.Cmd {
	if m.overlay != overlayPrompt || m.promptKind != "terminal.create" || len(m.promptFields) < 5 {
		m.clearPromptSuggestions()
		return nil
	}
	field := clampInt(m.promptField, 0, len(m.promptFields)-1)
	m.promptField = field
	switch field {
	case 2:
		endpointCmd := m.setCreateEndpoint(m.promptFields[field])
		items := m.createEndpointSuggestions(m.promptFields[field])
		empty := ""
		if len(items) == 0 {
			if len(m.pickerTabs()) == 0 {
				empty = "(no available servers)"
			} else if strings.TrimSpace(m.promptFields[field]) != "" {
				empty = "(no matching servers)"
			}
		}
		m.setPromptSuggestions(field, "servers", items, empty, focus)
		return endpointCmd
	case 3:
		value := m.promptFields[field]
		cursor := clampInt(m.promptCursors[field], 0, len([]rune(value)))
		prefix := promptPathPrefix(value, cursor)
		if strings.TrimSpace(prefix) == "" || m.promptEndpoint == "" || m.client == nil {
			m.clearPromptSuggestions()
			return nil
		}
		m.clearPromptSuggestions()
		m.promptSuggestionField = field
		m.promptSuggestionValue = value
		m.promptSuggestionCursor = cursor
		m.promptSuggestionEndpoint = m.promptEndpoint
		m.promptSuggestionFocusRequested = focus
		command, err := gproto.Marshal(&apipb.CommandEnvelope{
			Command: &apipb.CommandEnvelope_PathListDirectories{PathListDirectories: &apipb.PathListDirectoriesCommand{
				Prefix: prefix,
				Limit:  100,
			}},
		})
		if err != nil {
			return nil
		}
		return m.emit("access.call", &pb.MethodParams{Endpoint: m.promptEndpoint, AccessCommand: command}, opMsg{
			op: "path", endpoint: m.promptEndpoint, promptField: field,
			promptValue: value, promptCursor: cursor, promptFocus: focus,
		})
	default:
		m.clearPromptSuggestions()
		return nil
	}
}

func (m *model) focusCreateSuggestions() bool {
	if m.promptField != 2 || len(m.promptFields) < 3 {
		return false
	}
	suggestions := m.createEndpointSuggestions(m.promptFields[2])
	if len(suggestions) == 0 {
		return false
	}
	m.setPromptSuggestions(2, "servers", suggestions, "", true)
	m.promptSuggestionSel = 0
	if current := strings.TrimSpace(m.promptFields[2]); current != "" {
		for index, suggestion := range suggestions {
			if strings.EqualFold(suggestion, current) {
				m.promptSuggestionSel = index
				break
			}
		}
	}
	m.promptSuggestionOffset = promptSuggestionOffsetForSelection(m.promptSuggestionOffset, m.promptSuggestionSel, len(suggestions))
	m.promptSuggestionFocused = true
	return true
}

func (m *model) handleCreateSuggestionKey(key, char string) app.Cmd {
	if len(m.promptSuggestions) == 0 {
		m.clearPromptSuggestions()
		return nil
	}
	switch key {
	case "esc":
		m.clearPromptSuggestionFocus()
		return nil
	case "up", "shift-tab":
		m.promptSuggestionSel = (m.promptSuggestionSel + len(m.promptSuggestions) - 1) % len(m.promptSuggestions)
		m.promptSuggestionOffset = promptSuggestionOffsetForSelection(m.promptSuggestionOffset, m.promptSuggestionSel, len(m.promptSuggestions))
		return nil
	case "down", "tab":
		m.promptSuggestionSel = (m.promptSuggestionSel + 1) % len(m.promptSuggestions)
		m.promptSuggestionOffset = promptSuggestionOffsetForSelection(m.promptSuggestionOffset, m.promptSuggestionSel, len(m.promptSuggestions))
		return nil
	case "enter":
		value := m.promptSuggestions[clampInt(m.promptSuggestionSel, 0, len(m.promptSuggestions)-1)]
		field := m.promptSuggestionField
		if field < 0 || field >= len(m.promptFields) {
			field = m.promptField
		}
		m.promptFields[field] = value
		m.ensurePromptCursors()
		m.promptCursors[field] = len([]rune(value))
		m.clearPromptSuggestions()
		if field == 2 {
			return app.Batch(m.setCreateEndpoint(value), m.refreshCreateSuggestions(false))
		}
		return m.refreshCreateSuggestions(false)
	case "right":
		field := m.promptSuggestionField
		if field < 0 || field >= len(m.promptFields) {
			return nil
		}
		value := m.promptSuggestions[clampInt(m.promptSuggestionSel, 0, len(m.promptSuggestions)-1)]
		m.promptFields[field] = value
		m.ensurePromptCursors()
		m.promptCursors[field] = len([]rune(value))
		m.clearPromptSuggestionFocus()
		return app.Batch(m.setCreateEndpointForPromptField(field, value), m.refreshCreateSuggestions(true))
	case "left":
		field := m.promptSuggestionField
		if field < 0 || field >= len(m.promptFields) {
			m.clearPromptSuggestions()
			return nil
		}
		m.promptFields[field] = promptPathParent(m.promptFields[field])
		m.ensurePromptCursors()
		m.promptCursors[field] = len([]rune(m.promptFields[field]))
		m.clearPromptSuggestionFocus()
		return app.Batch(m.setCreateEndpointForPromptField(field, m.promptFields[field]), m.refreshCreateSuggestions(true))
	default:
		// Editing starts from the current field value; a focused list only owns
		// navigation keys and must not swallow typed input.
		m.clearPromptSuggestions()
		return m.handleCreatePromptKey(key, char)
	}
}

func (m *model) setCreateEndpointForPromptField(field int, value string) app.Cmd {
	if field != 2 {
		return nil
	}
	return m.setCreateEndpoint(value)
}

func (m *model) submitCreatePrompt() app.Cmd {
	if m.promptRef == "" || len(m.promptFields) < 5 {
		m.overlay = ""
		return nil
	}
	name := strings.TrimSpace(m.promptFields[0])
	if name == "" {
		m.promptError = "name is required"
		return nil
	}
	endpoint := strings.TrimSpace(m.promptFields[2])
	if endpoint == "" {
		endpoint = m.promptEndpoint
	}
	if endpoint == "" {
		m.promptError = "server is required"
		return nil
	}
	if normalized, ok := m.normalizeCreateEndpoint(endpoint); ok {
		endpoint = normalized
	}
	defaults := m.defaultsForEndpoint(endpoint)
	previousEndpoint := m.promptEndpoint
	previousCwd := m.promptDefaults.cwd
	command, err := parsePromptCommand(m.promptFields[1])
	if err != nil {
		m.promptError = err.Error()
		return nil
	}
	if len(command) == 0 {
		if !defaults.loaded || defaults.err != "" || len(defaults.command) == 0 {
			m.promptError = defaults.err
			if m.promptError == "" {
				m.promptError = "endpoint defaults are not loaded"
			}
			return nil
		}
		command = append([]string(nil), defaults.command...)
	}
	cwd := strings.TrimSpace(m.promptFields[3])
	if cwd == "" || (endpoint != previousEndpoint && cwd == previousCwd) {
		cwd = defaults.cwd
	}
	params := &pb.MethodParams{
		Endpoint: endpoint,
		Title:    name,
		Argv:     command,
		Cwd:      cwd,
		Tags:     m.promptTagsValue(),
	}
	m.overlay = ""
	m.promptKind = "command"
	m.promptPublicTagList = false
	m.promptError = ""
	m.promptEndpoint = endpoint
	if m.createDrafts == nil {
		m.createDrafts = map[string]createDraft{}
	}
	m.createDrafts[endpoint] = createDraft{command: strings.TrimSpace(m.promptFields[1]), cwd: strings.TrimSpace(params.GetCwd())}
	m.toast = "create requested"
	return m.bindPending(m.promptRef, "", params)
}

func (m *model) normalizeCreateEndpoint(value string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, tab := range m.pickerTabs() {
		candidates := []string{tab.name, tab.label}
		if tab.label != "" && tab.label != tab.name {
			candidates = append(candidates, fmt.Sprintf("%s (%s)", tab.label, tab.name))
		}
		if open := strings.LastIndex(value, "("); open >= 0 && strings.HasSuffix(value, ")") {
			candidates = append(candidates, strings.TrimSpace(value[open+1:len(value)-1]))
		}
		for _, candidate := range candidates {
			if candidate != "" && strings.EqualFold(value, candidate) {
				return tab.name, true
			}
		}
	}
	return "", false
}

func createCommandDisplay(command []string) string {
	return strings.Join(command, " ")
}

func (m *model) defaultsForEndpoint(endpoint string) createDefaults {
	if m.createDefaults == nil {
		m.createDefaults = map[string]createDefaults{}
	}
	if defaults, ok := m.createDefaults[endpoint]; ok {
		return defaults
	}
	return createDefaults{}
}

func (m *model) applyCreateDefaults(endpoint string, defaults createDefaults) {
	if m.overlay != overlayPrompt || m.promptKind != "terminal.create" || m.promptEndpoint != endpoint {
		return
	}
	previousCwd := m.promptDefaults.cwd
	previousCommand := createCommandDisplay(m.promptDefaults.command)
	m.promptDefaults = defaults
	if len(m.promptFields) < 5 {
		return
	}
	// Empty command stays empty so the form keeps the endpoint default as a
	// placeholder. A user-entered command is never overwritten by a refresh.
	if strings.TrimSpace(m.promptFields[1]) == previousCommand {
		m.promptFields[1] = ""
		m.ensurePromptCursors()
	}
	// Workdir is an endpoint default value in main. Replace only the previous
	// automatic value; preserve a path the user typed.
	if strings.TrimSpace(m.promptFields[3]) == "" || strings.TrimSpace(m.promptFields[3]) == previousCwd {
		m.promptFields[3] = defaults.cwd
		m.ensurePromptCursors()
		m.promptCursors[3] = len([]rune(defaults.cwd))
	}
}

func (m *model) setCreateEndpoint(endpoint string) app.Cmd {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil
	}
	normalized, ok := m.normalizeCreateEndpoint(endpoint)
	if !ok {
		return nil
	}
	endpoint = normalized
	if len(m.promptFields) >= 3 && strings.EqualFold(strings.TrimSpace(m.promptFields[2]), endpoint) {
		m.promptFields[2] = m.createEndpointPromptValue(endpoint)
		m.ensurePromptCursors()
		m.promptCursors[2] = len([]rune(m.promptFields[2]))
	}
	if endpoint == m.promptEndpoint {
		return nil
	}
	m.promptEndpoint = endpoint
	defaults := m.defaultsForEndpoint(endpoint)
	m.applyCreateDefaults(endpoint, defaults)
	if defaults.loaded || m.client == nil {
		return nil
	}
	command, err := gproto.Marshal(&apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_TerminalDefaults{TerminalDefaults: &apipb.TerminalDefaultsCommand{}},
	})
	if err != nil {
		return nil
	}
	return m.emit("access.call", &pb.MethodParams{Endpoint: endpoint, AccessCommand: command}, opMsg{op: "defaults", endpoint: endpoint})
}

func (m *model) requestCreateDefaults(endpoint string) app.Cmd {
	if endpoint == "" {
		return nil
	}
	// setCreateEndpoint also handles cached defaults and deduplicates requests.
	m.promptEndpoint = ""
	return m.setCreateEndpoint(endpoint)
}

// parsePromptCommand implements the small shell-like grammar used by the
// create form. It preserves quoted arguments without invoking a shell.
func parsePromptCommand(value string) ([]string, error) {
	var args []string
	var current []rune
	inSingle, inDouble, escaped, quoted := false, false, false, false
	flush := func() {
		if len(current) == 0 && !quoted {
			return
		}
		args = append(args, string(current))
		current = nil
		quoted = false
	}
	for _, r := range strings.TrimSpace(value) {
		switch {
		case escaped:
			current = append(current, r)
			escaped = false
		case r == '\\' && !inSingle:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
			quoted = true
		case r == '"' && !inSingle:
			inDouble = !inDouble
			quoted = true
		case !inSingle && !inDouble && unicode.IsSpace(r):
			flush()
		default:
			current = append(current, r)
		}
	}
	if escaped || inSingle || inDouble {
		return nil, fmt.Errorf("invalid command syntax")
	}
	flush()
	return args, nil
}

func (m *model) ensurePromptCursors() {
	if len(m.promptCursors) == len(m.promptFields) {
		for i := range m.promptCursors {
			m.promptCursors[i] = clampInt(m.promptCursors[i], 0, len([]rune(m.promptFields[i])))
		}
		return
	}
	m.promptCursors = make([]int, len(m.promptFields))
	for i := range m.promptFields {
		m.promptCursors[i] = len([]rune(m.promptFields[i]))
	}
}

func parsePromptTags(value string) map[string]string {
	tags := map[string]string{}
	position := 1
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "=") {
			pieces := strings.SplitN(part, "=", 2)
			key, val := strings.TrimSpace(pieces[0]), strings.TrimSpace(pieces[1])
			if key != "" && val != "" {
				tags[key] = val
			}
			continue
		}
		tags["tag"+strconv.Itoa(position)] = part
		position++
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

func parsePromptPublicTags(value string) map[string]string {
	tags := map[string]string{}
	seen := map[string]struct{}{}
	position := 1
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		tags["tag"+strconv.Itoa(position)] = part
		position++
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

func (m *model) promptTagsValue() map[string]string {
	if len(m.promptFields) < 5 {
		return nil
	}
	if m.promptPublicTagList {
		return parsePromptPublicTags(m.promptFields[4])
	}
	return parsePromptTags(m.promptFields[4])
}

func (t *tab) rebuild() {
	t.panes = nil
	for _, l := range leafNodes(t.root) {
		t.panes = append(t.panes, l.pane)
	}
}

func (t *tab) leafOf(p *pane) *leaf {
	for _, l := range leafNodes(t.root) {
		if l.pane == p {
			return l
		}
	}
	return nil
}

// floating is one floating window over the tab body.
type floating struct {
	id         string
	title      string
	x, y, w, h int
	collapsed  bool
	pane       *pane
}

// workspace groups tabs and remembers the active one.
type workspace struct {
	name   string
	tabs   []*tab
	active int
}

// ------------------------------------------------------------------- model

const (
	modeLive      = "live"
	modePane      = "pane"
	modeResize    = "resize"
	modeTab       = "tab"
	modeWorkspace = "workspace"
	modeSystem    = "system"
	modeFloating  = "floating"

	overlayPicker      = "picker"
	overlayPrompt      = "prompt"
	overlayHelp        = "help"
	overlayClipboard   = "clipboard"
	overlayConnections = "connections"
)

// connectionRow is one parsed endpoint.list row: the registered endpoint name
// (the wire target for endpoint.test/reconnect), its display label, kind and
// current health. The legacy TUI showed this same table behind
// system.open_connections.
type connectionRow struct {
	name   string
	label  string
	kind   string
	health string
}

// Picker status filter order matches main's terminal_picker.status_next cycle:
// Running -> Exited -> All. Running is the default.
const (
	pickerFilterRunning = iota
	pickerFilterExited
	pickerFilterAll
)

// model is the program state machine (the ChromeApp port plus the full v3
// scene/action table from the recommended yaml).
// emitter is the RESULT transport subset the model uses; *sdk.Client
// implements it and tests substitute a fake.
type emitter interface {
	Emit(method string, params *pb.MethodParams, onResponse func(*pb.Response)) (uint64, error)
}

type model struct {
	client emitter
	host   bool
	demo   bool

	viewID string
	epoch  uint64
	cols   int
	rows   int

	headerVisible bool
	footerVisible bool

	spaces []*workspace
	space  int

	tabSeq, paneSeq, floatSeq int

	mode           string
	shortcutLocked bool
	overlay        string
	picker         int
	pickerTab      int
	pickerQuery    string
	pickerFilter   int      // pickerFilterRunning/Exited/All
	pickerTagsOpen bool     // Ctrl-T opens the tag checkbox sub-view
	pickerTags     []string // selected public tag labels
	pickerTagSel   int
	pickerTagQuery string
	// emptyPaneSel is the highlighted CTA of the focused empty panel (main's
	// EmptyPaneCTA.SelectedIndex). Up/Down move it and Enter runs it.
	emptyPaneSel                   int
	prompt                         string
	promptCursor                   int
	promptSel                      int
	promptKind                     string
	promptRef                      string
	promptEndpoint                 string
	promptFields                   []string
	promptCursors                  []int
	promptField                    int
	promptError                    string
	promptPublicTagList            bool
	promptSuggestions              []string
	promptSuggestionSel            int
	promptSuggestionFocused        bool
	promptSuggestionTitle          string
	promptSuggestionEmpty          string
	promptSuggestionOffset         int
	promptSuggestionField          int
	promptSuggestionValue          string
	promptSuggestionCursor         int
	promptSuggestionEndpoint       string
	promptSuggestionFocusRequested bool
	promptDefaults                 createDefaults
	createDefaults                 map[string]createDefaults
	createDrafts                   map[string]createDraft
	clipboard                      []string
	clipboardIDs                   []string
	pasteLatestRef                 string
	clipSel                        int

	// connections backs the SYSTEM connections overlay (legacy
	// system.open_connections): the parsed endpoint.list rows and the selected
	// row. connSel indexes into connections while the overlay is open.
	connections []connectionRow
	connSel     int

	// workbench persistence via access.call storage (README §4). workbenchDirty
	// is set by every structural mutation and coalesced into one save by
	// workbenchReady/savePending, exactly herdr's layout save gate.
	workbenchDirty bool
	workbenchReady bool
	savePending    bool

	floatings   []*floating
	activeFloat string

	toast string

	sources       []*pb.Source
	sourcesReady  bool
	terminalCount int

	// copyPanes holds one copy/scrollback session per pane (the old
	// CopyModeByView model): presence = that pane's copy scene is open. Only
	// the focused pane's session owns input.
	copyPanes map[string]*copyState

	// ownerPaneBySource is the manual resize-owner designation (the legacy
	// panel.take_owner model): sourceID -> owning pane id. Ownership is never
	// inferred from focus, so clicking a follower pane cannot silently steal the
	// single terminal size. A missing entry falls back to the first pane bound
	// to that source in the active tab.
	ownerPaneBySource map[string]string

	zoomPane string

	dragging  string
	dragLastX int
	dragLastY int
	requestID uint64
	cursor    *cursorPos
	keys      sdk.Keys
	sentKeys  sdk.Keys
}

type cursorPos struct{ x, y int }

type createDefaults struct {
	command []string
	cwd     string
	loaded  bool
	err     string
}

type createDraft struct {
	command string
	cwd     string
}

const promptSuggestionVisibleRows = 6

func newModel(client emitter, demo bool) *model {
	m := &model{
		client:        client,
		host:          client != nil,
		demo:          demo,
		cols:          120,
		rows:          32,
		headerVisible: true,
		footerVisible: true,
		mode:          modeLive,
	}
	m.copyPanes = map[string]*copyState{}
	m.ownerPaneBySource = map[string]string{}
	m.createDefaults = map[string]createDefaults{}
	m.createDrafts = map[string]createDraft{}
	m.spaces = []*workspace{{name: "main"}}
	m.ws().tabs = []*tab{makeTab("tab-1", "main", []*pane{m.newPane("empty", nil)}, "row")}
	m.tabSeq = 1
	if demo {
		m.loadDemo()
	}
	return m
}

func (m *model) ws() *workspace { return m.spaces[m.space] }

func (m *model) activeTab() *tab {
	ws := m.ws()
	if len(ws.tabs) == 0 {
		return nil
	}
	if ws.active < 0 {
		ws.active = 0
	}
	if ws.active >= len(ws.tabs) {
		ws.active = len(ws.tabs) - 1
	}
	return ws.tabs[ws.active]
}

func (m *model) newPane(title string, lines []string) *pane {
	m.paneSeq++
	return &pane{id: fmt.Sprintf("pane-%d", m.paneSeq), title: title, lines: append([]string(nil), lines...)}
}

func (m *model) sourceByID(id string) *pb.Source {
	for _, src := range m.sources {
		if src.GetId() == id {
			return src
		}
	}
	return nil
}

func (m *model) terminals() []*pb.Source {
	var out []*pb.Source
	for _, src := range m.sources {
		if src.GetKind() == "terminal" && src.GetId() != "" {
			out = append(out, src)
		}
	}
	return out
}

func (m *model) focusPane() *pane {
	t := m.activeTab()
	if t == nil || len(t.panes) == 0 {
		return nil
	}
	if t.focus < 0 {
		t.focus = 0
	}
	if t.focus >= len(t.panes) {
		t.focus = len(t.panes) - 1
	}
	return t.panes[t.focus]
}

func (m *model) paneByID(id string) *pane {
	for _, ws := range m.spaces {
		for _, t := range ws.tabs {
			for _, p := range t.panes {
				if p.id == id {
					return p
				}
			}
		}
	}
	for _, f := range m.floatings {
		if f.pane.id == id {
			return f.pane
		}
	}
	return nil
}

func (m *model) tabOfPane(p *pane) (int, *tab) {
	ws := m.ws()
	for index, t := range ws.tabs {
		for _, candidate := range t.panes {
			if candidate == p {
				return index, t
			}
		}
	}
	return -1, nil
}

// --------------------------------------------------------------- pane chrome

func (m *model) paneSource(p *pane) *pb.Source {
	if p.sourceID == "" {
		return nil
	}
	return m.sourceByID(p.sourceID)
}

func (m *model) paneTitle(p *pane) string {
	src := m.paneSource(p)
	if src == nil {
		if p.title != "" {
			return p.title
		}
		return p.id
	}
	terminal := strings.TrimSpace(src.GetTitle())
	if terminal == "" {
		terminal = strings.TrimSpace(src.GetTerminalId())
	}
	if terminal == "" {
		terminal = "terminal"
	}
	// The @suffix shows the endpoint's display label (machine name), falling
	// back to the raw endpoint id, like the picker tabs.
	endpoint := strings.TrimSpace(src.GetEndpointLabel())
	if endpoint == "" {
		endpoint = strings.TrimSpace(src.GetEndpoint())
	}
	if endpoint == "" {
		endpoint = "local"
	}
	return sdk.Truncate(terminal, 18) + "@" + endpoint
}

// sourceOwnerPane is the single pane that owns one terminal source's size.
// The daemon terminal has exactly ONE extent (its PTY cols/rows) and therefore
// exactly one owning pane; additional panes bound to the same source are
// followers that mirror that extent (legacy resize-ownership model). Ownership
// is a manual designation (ownerPaneBySource, the legacy panel.take_owner), so
// focus never decides it: this returns the recorded owner if it still exists in
// the active tab and still binds the source, otherwise the first such pane
// (recording it so the designation sticks), else nil.
func (m *model) sourceOwnerPane(sourceID string) *pane {
	if sourceID == "" {
		return nil
	}
	t := m.activeTab()
	if t == nil || len(t.panes) == 0 {
		return nil
	}
	if id := m.ownerPaneBySource[sourceID]; id != "" {
		for _, p := range t.panes {
			if p.id == id && p.sourceID == sourceID {
				return p
			}
		}
		// The recorded owner is gone or was unbound: fall through to the
		// decl-order fallback below.
		delete(m.ownerPaneBySource, sourceID)
	}
	for _, p := range t.panes {
		if p.sourceID == sourceID {
			if m.ownerPaneBySource == nil {
				m.ownerPaneBySource = map[string]string{}
			}
			m.ownerPaneBySource[sourceID] = p.id
			return p
		}
	}
	return nil
}

// sourceExtent is a source's authoritative terminal extent in cells. A live
// source reports its PTY cols/rows; a source that has not reported a size yet
// (0/0) falls back to the pane content rect, which is exactly the owner path's
// full-bleed behavior. The extent is (cols, rows) in the pane content rect
// because card content is inset by one cell on each side (see cardNodes).
func (m *model) sourceExtent(src *pb.Source, content rect) (int, int) {
	cols, rows := int(src.GetCols()), int(src.GetRows())
	if cols <= 0 || rows <= 0 {
		return maxInt(0, content.w), maxInt(0, content.h)
	}
	return cols, rows
}

// paneOwnsSource reports whether p is the single owning pane of its terminal
// source. A source-bound pane is a follower whenever another pane (or, when p
// lives in a floating window, no active-tab pane) owns the extent.
func (m *model) paneOwnsSource(p *pane) bool {
	if p == nil || p.sourceID == "" {
		return false
	}
	return m.sourceOwnerPane(p.sourceID) == p
}

func (m *model) paneState(p *pane, active bool) (string, string) {
	src := m.paneSource(p)
	if src != nil && src.GetExited() {
		return "\u00d7", stDanger
	}
	if p.pending != "" {
		return "\u25cc", stWarning
	}
	if src != nil {
		if active {
			return glyphRunning, stSuccess
		}
		return glyphRunning, stMuted
	}
	if active {
		return "active", stSuccess
	}
	return "idle", stMuted
}

func (m *model) paneOwner(p *pane) (string, string, string) {
	src := m.paneSource(p)
	if src == nil {
		return "", stMuted, ""
	}
	if p.pending == "owner" {
		return "owner?", stWarning, ""
	}
	// Ownership is per SOURCE and manually designated (paneOwnsSource is the
	// ownerPaneBySource record, never focus): two panes can bind the same
	// terminal, but only the designated pane in the active tab owns its size
	// (the daemon PTY has one extent). The projected owner is the pair (this
	// pane is the source's owner pane) AND (this view holds the resize lease).
	// Everything else follows: a different pane on the same source is muted
	// with the take-owner action, exactly the old terminalChromeVM projection.
	// The demo exporter (view:demo) keeps its single lease-holding pane green.
	if m.paneOwnsSource(p) && strings.TrimSpace(src.GetResizeOwner()) != "" &&
		(m.demo || src.GetResizeOwner() == m.viewID) {
		// Legacy terminalChromeVMFromBinding colors the projected owner
		// (this view owns resize) with StyleSuccess, not the accent; only
		// the pending/acquire state stays warning and the follower muted.
		return "owner", stSuccess, ""
	}
	return "follow", stMuted, "pane:" + p.id + ":take-owner"
}

func (m *model) attachCount(p *pane) int {
	if m.paneSource(p) == nil {
		return 0
	}
	return 1
}

// paneRun is one run of a pane's top border row.
type paneRun struct {
	text  string
	style string
	node  string
	mouse bool
}

func actionGroupWidth(n int) int {
	if n == 0 {
		return 0
	}
	return 2 + 3*n
}

// paneRuns builds one pane top-border row (paneChromeTopSlots port).
func (m *model) paneRuns(p *pane, active bool, width int) []paneRun {
	return m.paneRunsRect(p, active, width, 0)
}

func (m *model) paneRunsRect(p *pane, active bool, width, height int) []paneRun {
	frame := stPanelBorder
	if active {
		frame = stAccent
	}
	// Legacy panel_chrome.paneChromeStyle checks the copy-history content kind
	// BEFORE Active: a pane with an open copy/scrollback session draws its whole
	// frame in the yellow history-border color, focused or not. The title text
	// and action glyphs below keep their own accent styling, exactly as the old
	// renderer separated the border style from paneChromeTitleStyle/actionStyle.
	if m.copyFor(p) != nil {
		frame = stHistoryBorder
	}
	runs := []paneRun{{"\u250c", frame, "", false}, {"\u2500", frame, "", false}}
	// The legacy renderer overlays the top clipping marker on the second
	// border cell (render/content_overflow_marker.go), ahead of the lock/title
	// slot, so it never merges with the corner or the left marker. A live
	// follower only clips at the right/bottom (its extent box is top-aligned),
	// but the shared paneOverflow keeps the frozen copy path's top marker.
	if width >= 3 {
		if _, _, top, _ := m.paneOverflow(p, width-2, maxInt(0, height-2)); top {
			runs[1] = paneRun{glyphOverflowTop, stOverflowStyle, "", false}
		}
	}
	if width < 4 {
		return runs
	}
	innerRight := width - 1
	terminal := m.paneSource(p) != nil
	type glyphAction struct {
		name  string
		glyph string
	}
	var actions []glyphAction
	if terminal {
		full := []glyphAction{{"zoom", m.zoomGlyph(p)}, {"split-v", glyphSplitV}, {"split-h", glyphSplitH}, {"close", glyphClose}}
		if actionGroupWidth(len(full)) <= width-6 {
			actions = full
		} else if actionGroupWidth(1) <= width-5 {
			actions = full[len(full)-1:]
		}
	} else {
		// Empty panels still have the structural split/close actions. main keeps
		// these available before a terminal is attached; attachment is an
		// independent content action below the panel chrome.
		full := []glyphAction{{"split-v", glyphSplitV}, {"split-h", glyphSplitH}, {"close", glyphClose}}
		if actionGroupWidth(len(full)) <= width-6 {
			actions = full
		} else if actionGroupWidth(1) <= width-5 {
			actions = full[len(full)-1:]
		}
	}
	actionWidth := actionGroupWidth(len(actions))
	actionX := innerRight
	rightLimit := innerRight
	if len(actions) > 0 {
		actionX = innerRight - actionWidth - 1
		rightLimit = actionX - 1
	}

	type rightSlot struct {
		text  string
		style string
		node  string
		mouse bool
	}
	var right []rightSlot
	if terminal {
		ownerText, ownerStyle, ownerNode := m.paneOwner(p)
		stateText, stateStyle := m.paneState(p, active)
		if m.paneSizeMismatch(p, width, height) {
			stateText, stateStyle = "size?", stWarning
		}
		right = []rightSlot{
			{centerPad(stateText, 3), stateStyle, "", false},
			{centerPad("x"+strconv.Itoa(m.attachCount(p)), 4), frame, "", false},
			{centerPad(ownerText, maxInt(8, sdk.DisplayWidth(ownerText))), ownerStyle, ownerNode, ownerNode != ""},
		}
	} else {
		stateText, stateStyle := m.paneState(p, active)
		right = []rightSlot{{centerPad(" "+stateText+" ", 12), stateStyle, "", false}}
	}

	rightWidth := 0
	for _, slot := range right {
		rightWidth += sdk.DisplayWidth(slot.text)
	}
	leftWidth := maxInt(0, rightLimit-2)
	prefix := " " + glyphUnlock + " "
	if p.locked {
		prefix = " " + glyphLocked + " "
	}
	lockNode := "pane:" + p.id + ":lock"
	title := m.paneTitle(p)
	minTitle := 0
	if title != "" {
		minTitle = 3
	}
	if sdk.DisplayWidth(prefix)+minTitle+rightWidth > leftWidth {
		prefix = ""
	}
	if sdk.DisplayWidth(prefix)+minTitle+rightWidth > leftWidth {
		right = nil
		rightWidth = 0
	}
	titleWidth := maxInt(0, leftWidth-sdk.DisplayWidth(prefix)-rightWidth)
	titleText := ""
	if title != "" {
		if titleWidth > 2 {
			titleText = " " + sdk.Truncate(title, titleWidth-2) + " "
		} else if titleWidth > 0 {
			titleText = sdk.Truncate(title, titleWidth)
		}
	}

	// v3 order: prefix -> title (advance eats the gap) -> right slots -> one
	// rule cell (rightLimit) -> action group -> trailing rule/corner.
	x := 2
	if prefix != "" {
		node := ""
		if terminal {
			node = lockNode
		}
		runs = append(runs, paneRun{prefix, frame, node, node != ""})
		x += sdk.DisplayWidth(prefix)
	}
	gap := titleWidth - sdk.DisplayWidth(titleText)
	if titleText != "" {
		titleStyle := stMuted
		if active {
			titleStyle = stAccent
		}
		runs = append(runs, paneRun{titleText, titleStyle, "", false})
		x += sdk.DisplayWidth(titleText)
	}
	if gap > 0 {
		runs = append(runs, paneRun{strings.Repeat("\u2500", gap), frame, "", false})
		x += gap
	}
	for _, slot := range right {
		runs = append(runs, paneRun{slot.text, slot.style, slot.node, slot.mouse})
		x += sdk.DisplayWidth(slot.text)
	}
	if len(actions) > 0 {
		if x < actionX {
			runs = append(runs, paneRun{strings.Repeat("\u2500", actionX-x), frame, "", false})
		}
		for index, action := range actions {
			if index == 0 {
				runs = append(runs, paneRun{edgeL, frame, "", false})
			}
			itemStyle := stInactiveGroup
			if active {
				itemStyle = stAccentGroup
			}
			runs = append(runs, paneRun{" " + action.glyph + " ", itemStyle, "pane:" + p.id + ":" + action.name, true})
		}
		runs = append(runs, paneRun{edgeRoundR, frame, "", false})
		x = actionX + actionWidth
	}
	tail := width - x
	if tail > 1 {
		runs = append(runs, paneRun{strings.Repeat("\u2500", tail-1), frame, "", false})
	}
	runs = append(runs, paneRun{"\u2510", frame, "", false})
	return runs
}

func (m *model) paneSizeMismatch(p *pane, width, height int) bool {
	src := m.paneSource(p)
	if src == nil || width <= 0 || height <= 0 || src.GetCols() <= 0 || src.GetRows() <= 0 {
		return false
	}
	// Only the source's owner pane can be "mismatched": the host resizes the
	// PTY to the owner, so a correctly-sized owner matches the terminal extent.
	// A follower's pane rect is unrelated to the terminal size by design (it
	// shows the extent at the owner's size), so it must never expose `size?`.
	if !m.paneOwnsSource(p) {
		return false
	}
	// card content is inset by one cell on each side/top/bottom.
	return int(src.GetCols()) != maxInt(1, width-2) || int(src.GetRows()) != maxInt(1, height-2)
}

// zoomGlyph is the zoom action's glyph: "↙" (unzoom) while this pane is the
// zoomed one, exactly the old pane.zoom toggle display rule.
func (m *model) zoomGlyph(p *pane) string {
	if m.zoomPane == p.id {
		return "\u2199"
	}
	return glyphZoom
}

// --------------------------------------------------------------- header

func (m *model) headerRuns() []paneRun {
	ws := m.ws()
	runs := []paneRun{
		{edgeL, stWSL, "", false},
		{" " + wsIcon + " " + ws.name + " ", stWSBody, "hdr:workspace", true},
		{edgeR, stWSR, "hdr:workspace", true},
	}
	for index, t := range ws.tabs {
		active := index == ws.active
		edgeLeft, body, edgeRight := stITabL, stITabBody, stITabR
		space, closeStyle := stITabSpace, stITabClose
		if active {
			edgeLeft, body, edgeRight = stTabL, stTabBody, stTabR
			space, closeStyle = stTabSpace, stTabClose
		}
		selectNode := "hdr:tab:" + strconv.Itoa(index)
		closeNode := "hdr:tabclose:" + strconv.Itoa(index)
		label := " " + tabIcon + " " + sdk.Truncate(t.title, 14) + " \u00b7 T" + strconv.Itoa(index+1) + " "
		runs = append(runs,
			paneRun{edgeR, edgeLeft, selectNode, true},
			paneRun{label, body, selectNode, true},
			paneRun{edgeR, edgeRight, selectNode, true},
			paneRun{" ", space, selectNode, true},
			paneRun{tabClose, closeStyle, closeNode, true},
			paneRun{" ", space, selectNode, true},
		)
	}
	runs = append(runs,
		paneRun{edgeR, stCreateL, "hdr:create", true},
		paneRun{" " + tabCreate + " ", stCreateBody, "hdr:create", true},
		paneRun{edgeRoundR, stCreateR, "hdr:create", true},
	)
	return runs
}

// --------------------------------------------------------------- footer

func (m *model) scene() string {
	switch m.overlay {
	case overlayPicker:
		if m.pickerTagsOpen {
			return "terminal-picker-tags"
		}
		return "terminal-picker"
	case overlayHelp:
		return "help"
	case overlayPrompt:
		return "prompt"
	case overlayClipboard:
		return "copy"
	case overlayConnections:
		return "connections"
	}
	switch m.mode {
	case modePane, modeResize, modeTab, modeWorkspace, modeSystem, modeFloating:
		return m.mode
	}
	if m.copyActive() {
		return "copy"
	}
	return "live"
}

type footerRun struct {
	text     string
	style    string
	node     string
	priority int
}

func (m *model) footerRightRuns() []footerRun {
	return []footerRun{
		{" " + wsIcon + " " + m.ws().name, stFooter, "", 2},
		{" " + glyphFloat + " " + strconv.Itoa(len(m.floatings)), stFooterAccent, "", 1},
		{" " + glyphTerm + " " + strconv.Itoa(m.terminalCount), stFooter, "", 4},
		{" ", stFooter, "", 4},
	}
}

// selectActions ports selectFooterActionTokens: greedy fill, then tail.
func selectActions(actions []footerAction, limit int) []footerAction {
	var selected []footerAction
	used := 0
	truncated := false
	for _, action := range actions {
		tokenWidth := 1 + sdk.DisplayWidth(action.label)
		if len(selected) > 0 {
			tokenWidth += 3
		}
		if len(selected) > 0 && used+tokenWidth > limit {
			truncated = true
			break
		}
		if len(selected) == 0 && tokenWidth > limit {
			truncated = true
			break
		}
		selected = append(selected, action)
		used += tokenWidth
	}
	if !truncated || len(selected) == len(actions) {
		return selected
	}
	tail := actions[len(actions)-1]
	tailWidth := 1 + sdk.DisplayWidth(tail.label)
	if len(selected) > 0 {
		tailWidth += 3
	}
	for len(selected) > 0 && used+tailWidth > limit {
		dropped := selected[len(selected)-1]
		selected = selected[:len(selected)-1]
		used -= 1 + sdk.DisplayWidth(dropped.label)
		if len(selected) > 0 {
			used -= 3
		}
		tailWidth = 1 + sdk.DisplayWidth(tail.label)
		if len(selected) > 0 {
			tailWidth += 3
		}
	}
	if tailWidth <= limit && used+tailWidth <= limit {
		selected = append(selected, tail)
	}
	return selected
}

func (m *model) footerRuns() ([]footerRun, []footerRun) {
	spec := scenes[m.scene()]
	var runs []footerRun
	if spec.icon != "" || spec.label != "" {
		badge := " " + spec.icon + " "
		if spec.label != "" {
			badge = " " + spec.icon + " " + spec.label + " "
		}
		token := modeStyles[m.scene()]
		if token == "" {
			token = "footer-accent"
		}
		runs = append(runs, footerRun{badge, footerStyleTokens[token], "", 1})
	}
	right := m.footerRightRuns()
	reserve := 0
	for _, run := range right {
		reserve += sdk.DisplayWidth(run.text)
	}
	limit := m.cols
	if m.cols >= 120 && m.scene() == "live" {
		limit = maxInt(0, m.cols-reserve)
	}
	selected := selectActions(spec.actions, limit)
	for _, action := range selected {
		if len(runs) > 0 {
			runs = append(runs, footerRun{" \u00b7 ", stFooter, "", 1})
		}
		runs = append(runs, footerRun{" " + action.label, footerActionStyle(action), action.node, 1})
	}
	return runs, right
}

// trimRuns ports shell_bar.trimBarSegments: drop the highest-priority-number
// run until the width fits.
func trimRuns(runs []footerRun, width int) []footerRun {
	out := append([]footerRun(nil), runs...)
	total := func() int {
		sum := 0
		for _, run := range out {
			sum += sdk.DisplayWidth(run.text)
		}
		return sum
	}
	for total() > width && len(out) > 0 {
		index := 0
		for i, run := range out {
			if run.priority >= out[index].priority {
				index = i
			}
		}
		out = append(out[:index], out[index+1:]...)
	}
	return out
}

func (m *model) footerLine() string {
	runs, right := m.footerRuns()
	runs = trimRuns(runs, m.cols)
	leftWidth := 0
	for _, run := range runs {
		leftWidth += sdk.DisplayWidth(run.text)
	}
	right = trimRuns(right, m.cols-leftWidth)
	rightWidth := 0
	for _, run := range right {
		rightWidth += sdk.DisplayWidth(run.text)
	}
	pad := maxInt(0, m.cols-leftWidth-rightWidth)
	var b strings.Builder
	for _, run := range runs {
		b.WriteString(run.text)
	}
	b.WriteString(strings.Repeat(" ", pad))
	for _, run := range right {
		b.WriteString(run.text)
	}
	return b.String()
}

// ------------------------------------------------------------------- demo

// demoLeftLines / demoRightLines are the 1.txt placeholder contents.
var demoLeftLines = []string{
	"",
	"  " + glyphGutter,
	"  " + glyphGutter + "  " + collapseHint,
	"",
	"  复刻：老 v3 的 card pane + bracket 动作组",
	"  chrome 逐字符对齐（120 列 golden）",
	"",
	"  Ctrl-P PANE 模式：",
	"    % / \" 分屏 · x 关闭 · t 重启 · q kill+close",
	"    h / l 焦点 · z 折叠提示行",
	"  Ctrl-T TAB · Ctrl-O FLOAT · Ctrl-F PICK",
	"  Ctrl-G SYSTEM · Ctrl-Q 退出",
	"",
}

var demoRightLines = []string{
	"",
	"  OpenCode / 任意终端内容区",
	"",
	"  content.self = terminal:<endpoint>:<id>",
	"  组件边框已关闭（chrome.inset=0），",
	"  pane 框由程序自绘：",
	"",
	"  \u250c\u2500 \u25a1  anytty-surface@hs \u2500\u2500\u2500 \u25cf  x1  owner \u2500\u2500 " + edgeL + " " + glyphZoom + "  " + glyphSplitV + "  " + glyphSplitH + "  " + glyphClose + " " + edgeRoundR + " \u2500\u2510",
	"",
}

// loadDemo preloads the 1.txt target state.
func (m *model) loadDemo() {
	m.ws().name = "main"
	m.tabSeq = 2
	m.paneSeq = 0
	left := m.newPane("anytty-surface@hs", demoLeftLines)
	right := m.newPane("opencode@hs", demoRightLines)
	tab1 := makeTab("tab-1", "auto-push", []*pane{left, right}, "row")
	tab1.focus = 0
	third := m.newPane("local@hs", []string{"", "  local \u00b7 T2"})
	tab2 := makeTab("tab-2", "local", []*pane{third}, "row")
	m.ws().tabs = []*tab{tab1, tab2}
	m.ws().active = 0
	var pool []*pb.Source
	for index := 0; index < 11; index++ {
		terminalID := fmt.Sprintf("term-%d", index)
		switch index {
		case 0:
			terminalID = "anytty-surface"
		case 1:
			terminalID = "opencode"
		case 2:
			terminalID = "local"
		}
		source := &pb.Source{
			Id:         "terminal:hs:" + terminalID,
			Kind:       "terminal",
			Title:      terminalID,
			Endpoint:   "hs",
			TerminalId: terminalID,
			Attached:   true,
		}
		if index == 0 {
			source.ResizeOwner = "view:demo"
		}
		pool = append(pool, source)
	}
	m.sources = pool
	m.sourcesReady = true
	m.terminalCount = len(pool)
	left.sourceID = pool[0].GetId()
	right.sourceID = pool[1].GetId()
	third.sourceID = pool[2].GetId()
}

// ------------------------------------------------------------------ update

func (m *model) Init() app.Cmd {
	return chain(app.SetKeys(m.claim()), m.loadWorkbenchCmd())
}

func (m *model) Reset(epoch uint64) {
	m.epoch = epoch
	m.toast = ""
	m.dragging = ""
	m.copyPanes = map[string]*copyState{}
	m.pendingClear()
}

// pendingClear drops transient per-pane pending flags after a host restart.
func (m *model) pendingClear() {
	for _, ws := range m.spaces {
		for _, t := range ws.tabs {
			for _, p := range t.panes {
				p.pending = ""
			}
		}
	}
	for _, f := range m.floatings {
		f.pane.pending = ""
	}
}

func (m *model) Update(msg app.Msg) app.Cmd {
	var cmd app.Cmd
	switch v := msg.(type) {
	case app.HelloMsg:
		hello := v.Hello
		m.viewID = hello.GetViewId()
		if cols, rows := int(hello.GetCols()), int(hello.GetRows()); cols > 0 && rows > 0 {
			m.cols, m.rows = cols, rows
		}
	case app.SourcesMsg:
		cmd = m.onSources(v.Items)
	case app.TickMsg:
		cmd = m.applySearchScan()
	case app.ResizeMsg:
		m.cols, m.rows = v.Cols, v.Rows
		m.clampFloatings()
	case app.KeyMsg:
		cmd = m.onKey(v.Key.GetKey(), v.Key.GetChar())
	case app.PasteMsg:
		if m.overlay == overlayPrompt && m.promptKind == "terminal.create" && v.Paste != nil && v.Paste.GetText() != "" {
			cmd = m.insertCreatePromptText(v.Paste.GetText())
		} else if m.overlay == overlayPrompt && m.promptKind != "terminal.create" && v.Paste != nil && v.Paste.GetText() != "" {
			m.insertPromptText(v.Paste.GetText())
		} else if v.Paste != nil {
			if p := m.focusContentPane(); p != nil && p.sourceID == "" && v.Paste.GetText() != "" {
				p.lines = append(p.lines, "> "+v.Paste.GetText())
			}
		}
	case app.MouseMsg:
		cmd = m.onMouse(v.Mouse)
	case app.WheelMsg:
		cmd = m.onWheel(v.Wheel)
	case app.NoticeMsg:
		// Endpoint connectivity notices are already reflected by the picker's
		// endpoint tabs (health glyph), exactly like main. Keeping them as a
		// persistent toast only duplicates that state and covers the footer,
		// so they are suppressed here. Protocol/crash warnings still show.
		if !isEndpointHealthNotice(v.Message) {
			m.toast = v.Level + ": " + v.Message
		}
	case app.ResponseMsg:
		// Responses matched by an emit command arrive as opMsg instead.
	case opMsg:
		cmd = m.onOp(v)
	case app.ViewRejectedMsg:
		m.toast = "view rejected: " + v.Reason
	case app.ErrorMsg:
		m.toast = "transport: " + v.Err.Error()
	}
	want := m.claim()
	if !keysEqual(want, m.sentKeys) {
		m.sentKeys = want
		cmd = chain(cmd, app.SetKeys(want))
	}
	// A structural edit marks the workbench dirty; schedule the coalesced save
	// here so every mutation path persists without threading a command back.
	return chain(cmd, m.saveWorkbenchCmd())
}

func keysEqual(a, b sdk.Keys) bool {
	if a.All != b.All || len(a.Claim) != len(b.Claim) {
		return false
	}
	for i := range a.Claim {
		if a.Claim[i] != b.Claim[i] {
			return false
		}
	}
	return true
}

// isEndpointHealthNotice reports whether a host notice is an endpoint
// connectivity/route status update. main surfaces that only through the
// picker's endpoint tabs, never as a persistent toast over the footer.
func isEndpointHealthNotice(message string) bool {
	message = strings.ToLower(message)
	return strings.HasPrefix(message, "endpoint ") && (strings.Contains(message, " offline") ||
		strings.Contains(message, " connected") ||
		strings.Contains(message, "route") ||
		strings.Contains(message, "unreachable") ||
		strings.Contains(message, "reconnect"))
}

func chain(cmds ...app.Cmd) app.Cmd {
	var alive []app.Cmd
	for _, cmd := range cmds {
		if cmd != nil {
			alive = append(alive, cmd)
		}
	}
	switch len(alive) {
	case 0:
		return nil
	case 1:
		return alive[0]
	default:
		return app.Batch(alive...)
	}
}

func (m *model) onSources(items []*pb.Source) app.Cmd {
	if m.demo {
		return nil
	}
	m.sources = nil
	for _, src := range items {
		if src.GetId() != "" {
			m.sources = append(m.sources, src)
		}
	}
	m.terminalCount = len(m.terminals())
	present := map[string]bool{}
	for _, src := range m.terminals() {
		present[src.GetId()] = true
	}
	for _, ws := range m.spaces {
		for _, t := range ws.tabs {
			for _, p := range t.panes {
				if p.sourceID != "" && !present[p.sourceID] {
					m.forgetOwner(p)
					p.sourceID = ""
					p.lines = nil
					delete(m.copyPanes, p.id)
				}
			}
		}
	}
	for _, f := range m.floatings {
		if f.pane.sourceID != "" && !present[f.pane.sourceID] {
			m.forgetOwner(f.pane)
			f.pane.sourceID = ""
			f.pane.lines = nil
			delete(m.copyPanes, f.pane.id)
		}
	}
	if !m.sourcesReady {
		m.sourcesReady = true
		if len(m.terminals()) == 0 && !m.demo {
			m.openPicker()
		}
	}
	m.pickerTab = clampInt(m.pickerTab, 0, maxInt(0, len(m.pickerTabs())-1))
	m.picker = clampInt(m.picker, 0, maxInt(0, len(m.pickerRows())-1))
	return nil
}

// -------------------------------------------------------------- claims

// claim is the routing declaration: live mode with a focused terminal claims
// only the global chords so typing reaches the PTY; every modal scene (and
// every overlay) claims every key, exactly the old UI's modal input model.
func (m *model) claim() sdk.Keys {
	if m.overlay != "" || m.mode != modeLive || m.copyActive() {
		return sdk.Keys{All: true}
	}
	if m.shortcutLocked {
		// Keep the system chord available as the unlock path. Ctrl-Q remains
		// host-reserved by the protocol; other shortcuts fall through to PTY.
		return sdk.Keys{Claim: []string{"ctrl-g"}}
	}
	base := []string{
		"ctrl-p", "ctrl-r", "ctrl-o", "ctrl-t", "ctrl-w", "ctrl-f",
		"ctrl-shift-c", "ctrl-shift-h", "ctrl-shift-v", "ctrl-g",
		// Legacy global copy.enter also binds Ctrl-V and PageUp.
		"ctrl-v", "page-up",
	}
	for n := 1; n <= 5; n++ {
		base = append(base, fmt.Sprintf("ctrl-alt-%d", n))
	}
	return sdk.Keys{Claim: base}
}

// ------------------------------------------------------------------- input

// ------------------------------------------------------------------- input

func (m *model) onKey(key, char string) app.Cmd {
	if m.toast != "" {
		m.toast = ""
	}
	if m.overlay != "" {
		return m.handleOverlayKey(key, char)
	}
	if m.mode == modeLive && m.copyActive() {
		return m.handleCopyKey(key, char)
	}
	// A focused empty panel owns Up/Down/Enter for its CTA list in every scene
	// (main's EmptyPaneCTA keyboard selection); other keys fall through so the
	// global chords keep working.
	if p := m.focusedEmptyPane(); p != nil {
		switch key {
		case "up":
			m.moveEmptyPaneSelection(-1)
			return nil
		case "down":
			m.moveEmptyPaneSelection(1)
			return nil
		case "enter":
			return m.runEmptyPaneSelection(p)
		}
	}
	switch m.mode {
	case modePane:
		return m.handlePaneKey(key)
	case modeResize:
		return m.handleResizeKey(key)
	case modeTab:
		return m.handleTabKey(key)
	case modeWorkspace:
		return m.handleWorkspaceKey(key)
	case modeSystem:
		return m.handleSystemKey(key)
	case modeFloating:
		return m.handleFloatingKey(key)
	}
	return m.handleLiveKey(key)
}

func (m *model) handleLiveKey(key string) app.Cmd {
	switch key {
	case "ctrl-p":
		m.mode = modePane
	case "ctrl-r":
		m.mode = modeResize
	case "ctrl-o":
		m.mode = modeFloating
		m.openFloatMenu()
	case "ctrl-t":
		m.mode = modeTab
	case "ctrl-w":
		m.mode = modeWorkspace
	case "ctrl-f":
		m.openPicker()
	case "ctrl-g":
		m.mode = modeSystem
	case "ctrl-shift-c", "ctrl-v", "page-up":
		// Legacy global copy.enter binds Ctrl-V and PageUp too.
		if p := m.focusContentPane(); p != nil {
			if st := m.copyFor(p); st != nil {
				return m.resetCopyToLatest(p, st)
			}
		}
		return m.enterCopy()
	case "ctrl-shift-h":
		return m.openClipboardHistory()
	case "ctrl-shift-v":
		return m.pasteSystem()
	case "ctrl-q":
		return m.quitCmd()
	case "esc":
		m.mode = modeLive
	default:
		if strings.HasPrefix(key, "ctrl-alt-") {
			digits := strings.TrimPrefix(key, "ctrl-alt-")
			if n, err := strconv.Atoi(digits); err == nil {
				index := n - 1
				if index >= 0 && index < len(m.ws().tabs) {
					m.ws().active = index
					m.mode = modeLive
				}
			}
		}
	}
	return nil
}

func (m *model) handlePaneKey(key string) app.Cmd {
	t := m.activeTab()
	p := m.focusPane()
	switch key {
	case "ctrl-p", "esc":
		m.mode = modeLive
	case "ctrl-f":
		m.mode = modeLive
		m.openPicker()
	case "ctrl-t":
		m.mode = modeLive
		m.newTab()
	case "ctrl-o":
		m.mode = modeFloating
		m.openFloatMenu()
	case "ctrl-g":
		m.mode = modeSystem
	case "ctrl-shift-c":
		return m.enterCopy()
	case "x", "w":
		// Legacy panel.close binds both x and w.
		m.closePane(t, p)
		m.mode = modeLive
	case "%", "ctrl-d":
		return m.splitPane("row")
	case "\"", "ctrl-e":
		return m.splitPane("col")
	case "q":
		cmd := m.killClosePane(t, p)
		m.mode = modeLive
		return cmd
	case "k":
		return m.killPane(p)
	case "X":
		// Legacy panel.kill.
		m.mode = modeLive
		return m.killPane(p)
	case "R":
		// Legacy panel.restart.
		return m.restartPane(p)
	case "t":
		return m.restartPane(p)
	case "a":
		return m.takeOwner(p)
	case "s":
		if p != nil {
			p.locked = !p.locked
		}
	case "z":
		m.toggleZoom(p)
	case "b":
		m.resetTabSplits(t)
	case "c", "p":
		m.toast = "presentation: card (default)"
	case "d":
		if p != nil && p.sourceID != "" {
			return m.emit("terminal.detach", m.clipboardPasteParams(p), opMsg{op: "detach", ref: p.id})
		}
		m.toast = "detach: no terminal"
	case "r":
		if p != nil {
			sourceID := p.sourceID
			if sourceID == "" {
				sourceID = p.detachedSourceID
			}
			if sourceID != "" {
				return m.emit("terminal.reconnect", m.terminalParamsForSource(sourceID), opMsg{op: "reconnect", ref: p.id, source: sourceID})
			}
		}
		m.toast = "reconnect: no terminal"
	case "h", "left", "up":
		m.focusPaneDelta(-1)
	case "l", "right", "down":
		m.focusPaneDelta(1)
	case ":":
		m.mode = modeLive
		m.openPrompt()
	case "?":
		m.mode = modeLive
		m.overlay = overlayHelp
	default:
		if n, err := strconv.Atoi(key); err == nil && n >= 1 && n <= 9 {
			index := n - 1
			if index < len(m.ws().tabs) {
				m.ws().active = index
				m.mode = modeLive
			}
		}
	}
	return nil
}

func (m *model) handleResizeKey(key string) app.Cmd {
	t := m.activeTab()
	switch key {
	case "ctrl-p", "esc":
		m.mode = modeLive
	case "ctrl-g":
		m.mode = modeSystem
	case ":":
		m.mode = modeLive
		m.openPrompt()
	case "h", "left":
		m.resizeFocused(-2, false)
	case "l", "right":
		m.resizeFocused(2, false)
	case "k", "up":
		m.resizeFocused(-2, true)
	case "j", "down":
		m.resizeFocused(2, true)
	case "r":
		m.resetTabSplits(t)
	case "s":
		if p := m.focusPane(); p != nil {
			p.locked = !p.locked
		}
	case "=", "b":
		m.resetTabSplits(t)
	case "space":
		m.toggleLayout(t)
	case "H":
		// Legacy resize.left_large (bias delta 6).
		m.resizeFocused(-6, false)
	case "L":
		m.resizeFocused(6, false)
	case "K":
		m.resizeFocused(-6, true)
	case "J":
		m.resizeFocused(6, true)
	case "m":
		// Legacy resize.center: even the split axis (this replica has no
		// content letterbox, so the meaningful visible result of the tiled
		// "center" is the even split).
		m.centerFocused()
	case "0", "$", "^", "B", "x", "y", "|", "_", "shift-left", "shift-right", "shift-up", "shift-down":
		// Legacy resize.align_*/center_x/center_y/pan_* are per-view CONTENT
		// layout (the terminal extent inside a fixed pane viewport). v3shell's
		// terminal box rect is the PTY winsize, so a real letterbox needs a
		// component/host content-offset capability that does not exist yet;
		// keep an explicit notice instead of a silently wrong resize.
		m.toast = "layout: align/center/pan needs a host content-offset capability"
	case "ctrl-left", "alt-h":
		m.resizeFocused(-2, false)
	case "ctrl-right", "alt-l":
		m.resizeFocused(2, false)
	case "ctrl-up", "alt-k":
		m.resizeFocused(-2, true)
	case "ctrl-down", "alt-j":
		m.resizeFocused(2, true)
	}
	return nil
}

func (m *model) handleTabKey(key string) app.Cmd {
	ws := m.ws()
	switch key {
	case "ctrl-t", "esc":
		m.mode = modeLive
	case "c":
		m.newTab()
		m.mode = modeLive
	case "n", "l", "]":
		if len(ws.tabs) > 0 {
			ws.active = (ws.active + 1) % len(ws.tabs)
			m.markWorkbenchDirty()
		}
	case "p", "h", "[":
		if len(ws.tabs) > 0 {
			ws.active = (ws.active - 1 + len(ws.tabs)) % len(ws.tabs)
			m.markWorkbenchDirty()
		}
	case "x":
		m.closeTab(ws.active)
	case "X":
		// Legacy tab.kill.
		return m.killTab(ws.active)
	case "k":
		return m.killTab(ws.active)
	case "r":
		m.openRename("tab", m.activeTab().id, m.activeTab().title)
	default:
		if n, err := strconv.Atoi(key); err == nil && n >= 1 && n <= 9 {
			index := n - 1
			if index < len(ws.tabs) {
				ws.active = index
				m.mode = modeLive
				m.markWorkbenchDirty()
			}
		}
	}
	return nil
}

func (m *model) handleWorkspaceKey(key string) app.Cmd {
	switch key {
	case "ctrl-w", "esc":
		m.mode = modeLive
	case "c":
		m.createWorkspace()
	case "n", "l", "]":
		m.space = (m.space + 1) % len(m.spaces)
		m.markWorkbenchDirty()
	case "p", "h", "[":
		m.space = (m.space - 1 + len(m.spaces)) % len(m.spaces)
		m.markWorkbenchDirty()
	case "x":
		m.deleteWorkspace(m.space)
	case "r":
		m.openRename("workspace", m.ws().name, m.ws().name)
	case "t", "f", "s":
		m.mode = modeLive
		m.toast = "workbench tree: use the connections overlay (Ctrl-G e)"
	}
	return nil
}

func (m *model) handleSystemKey(key string) app.Cmd {
	switch key {
	case "ctrl-g", "esc":
		m.mode = modeLive
	case "q":
		return m.quitCmd()
	case "o":
		m.mode = modeLive
		m.openPrompt()
	case "?":
		m.mode = modeLive
		m.overlay = overlayHelp
	case "h":
		m.headerVisible = !m.headerVisible
		m.markWorkbenchDirty()
	case "f":
		m.footerVisible = !m.footerVisible
		m.markWorkbenchDirty()
	case "c", "x":
		m.toast = ""
	case "T":
		// Legacy system.close_toast.
		m.toast = ""
	case "p", "m", "t":
		m.mode = modeLive
		m.openPicker()
	case "w":
		m.mode = modeLive
		m.toast = "workbench tree: host storage owns workspaces"
	case "e":
		m.mode = modeLive
		// Legacy system.open_connections: list endpoints in an overlay so they
		// can be tested/reconnected, not just a toast count.
		return m.openConnections()
	case "l":
		m.shortcutLocked = !m.shortcutLocked
		m.mode = modeLive
		if m.shortcutLocked {
			m.toast = "shortcut lock: on"
		} else {
			m.toast = "shortcut lock: off"
		}
	case "a", "A":
		m.toast = "plugins: not available in the v2 host"
	}
	return nil
}

func (m *model) handleFloatingKey(key string) app.Cmd {
	if key == "ctrl-o" || key == "esc" {
		m.mode = modeLive
		return nil
	}
	f := m.activeFloating()
	switch key {
	case "n":
		return m.newFloating()
	case "o":
		m.toast = fmt.Sprintf("floating: %d window(s)", len(m.floatings))
	case "x":
		if f != nil {
			m.closeFloating(f)
		}
	case "z", "m":
		if f != nil {
			m.toggleFloatingCollapse(f)
		}
	case "c":
		if f != nil {
			m.centerFloating(f)
		}
	case "v":
		allCollapsed := true
		for _, item := range m.floatings {
			if !item.collapsed {
				allCollapsed = false
			}
		}
		for _, item := range m.floatings {
			item.collapsed = !allCollapsed
		}
	case "=":
		if f != nil {
			m.maximizeFloating(f)
		}
	case "s":
		m.toast = "auto-fit: host geometry"
	case "f":
		m.openPicker()
	case "a":
		return m.takeOwner(f.pane)
	case "h", "left":
		if f != nil {
			f.x = maxInt(0, f.x-2)
			m.raiseFloating(f)
		}
	case "l", "right":
		if f != nil {
			f.x = minInt(maxInt(0, m.cols-f.w), f.x+2)
			m.raiseFloating(f)
		}
	case "k", "up":
		if f != nil {
			f.y = maxInt(1, f.y-1)
			m.raiseFloating(f)
		}
	case "j", "down":
		if f != nil {
			f.y = minInt(maxInt(1, m.rows-1-f.h), f.y+1)
			m.raiseFloating(f)
		}
	case ",":
		m.resizeFloating(f, -4, 0)
	case ".":
		m.resizeFloating(f, 4, 0)
	case ";":
		m.resizeFloating(f, 0, -2)
	case "/":
		m.resizeFloating(f, 0, 2)
	case "H":
		// Legacy floating.narrow.
		m.resizeFloating(f, -4, 0)
	case "L":
		// Legacy floating.wide.
		m.resizeFloating(f, 4, 0)
	case "K":
		// Legacy floating.short.
		m.resizeFloating(f, 0, -2)
	case "J":
		// Legacy floating.tall.
		m.resizeFloating(f, 0, 2)
	default:
		if n, err := strconv.Atoi(key); err == nil && n >= 1 && n <= 9 {
			index := n - 1
			if index >= 0 && index < len(m.floatings) {
				target := m.floatings[index]
				target.collapsed = false
				m.raiseFloating(target)
			}
		}
	}
	return nil
}

func (m *model) handleOverlayKey(key, char string) app.Cmd {
	switch m.overlay {
	case overlayPicker:
		if m.pickerTagsOpen {
			return m.handlePickerTagsKey(key, char)
		}
		rows := m.pickerRows()
		switch key {
		case "backspace":
			if m.pickerQuery != "" {
				r := []rune(m.pickerQuery)
				m.pickerQuery = string(r[:len(r)-1])
				m.picker = 0
			}
		case "delete":
			if m.pickerQuery != "" {
				r := []rune(m.pickerQuery)
				m.pickerQuery = string(r[:len(r)-1])
				m.picker = 0
			}
		case "/":
			m.pickerQuery = ""
		case "ctrl-t":
			m.openPickerTags()
		case "up":
			m.picker = clampInt(m.picker-1, 0, maxInt(0, len(rows)-1))
		case "down":
			m.picker = clampInt(m.picker+1, 0, maxInt(0, len(rows)-1))
		case "enter", "tab":
			return m.attach(m.picker, key == "tab")
		case "esc":
			m.overlay = ""
		case "left":
			m.pickerTab--
			m.picker = 0
		case "right":
			m.pickerTab++
			m.picker = 0
		case "shift-left":
			m.pickerFilter = (m.pickerFilter + 2) % 3
			m.picker = 0
		case "shift-right":
			m.pickerFilter = (m.pickerFilter + 1) % 3
			m.picker = 0
		case "ctrl-e":
			if m.picker < len(rows) && rows[m.picker].source != nil {
				src := rows[m.picker].source
				m.openRename("terminal.rename", src.GetId(), src.GetTitle())
			}
		case "ctrl-k":
			if m.picker < len(rows) && rows[m.picker].source != nil {
				return m.killSource(rows[m.picker].source)
			}
		case "ctrl-x":
			if m.picker < len(rows) && rows[m.picker].source != nil {
				return m.removeSource(rows[m.picker].source)
			}
		default:
			if char != "" && len([]rune(char)) == 1 && !strings.HasPrefix(key, "ctrl-") {
				m.pickerQuery += char
				m.picker = 0
			}
		}
	case overlayPrompt:
		if m.promptKind == "terminal.create" {
			return m.handleCreatePromptKey(key, char)
		}
		matches := m.promptMatches()
		switch key {
		case "esc":
			m.overlay = ""
		case "up":
			m.promptSel = clampInt(m.promptSel-1, 0, maxInt(0, len(matches)-1))
		case "down":
			m.promptSel = clampInt(m.promptSel+1, 0, maxInt(0, len(matches)-1))
		case "backspace":
			if m.prompt != "" {
				runes := []rune(m.prompt)
				cursor := clampInt(m.promptCursor, 0, len(runes))
				if cursor > 0 {
					m.prompt = string(append(runes[:cursor-1], runes[cursor:]...))
					m.promptCursor = cursor - 1
				}
				m.promptSel = 0
			}
		case "delete":
			runes := []rune(m.prompt)
			cursor := clampInt(m.promptCursor, 0, len(runes))
			if cursor < len(runes) {
				m.prompt = string(append(runes[:cursor], runes[cursor+1:]...))
				m.promptSel = 0
			}
		case "left":
			m.promptCursor = clampInt(m.promptCursor-1, 0, len([]rune(m.prompt)))
		case "right":
			m.promptCursor = clampInt(m.promptCursor+1, 0, len([]rune(m.prompt)))
		case "home":
			m.promptCursor = 0
		case "end":
			m.promptCursor = len([]rune(m.prompt))
		case "enter":
			if m.promptKind == "rename" || m.promptKind == "terminal.rename" {
				return m.applyRename()
			} else if m.promptSel >= 0 && m.promptSel < len(matches) {
				m.runCommand(matches[m.promptSel])
			}
		default:
			if char != "" && !strings.HasPrefix(key, "ctrl-") && !strings.HasPrefix(key, "alt-") {
				m.insertPromptText(char)
				m.promptSel = 0
			} else if len([]rune(key)) == 1 && !strings.HasPrefix(key, "ctrl-") && !strings.HasPrefix(key, "alt-") {
				m.insertPromptText(key)
				m.promptSel = 0
			}
		}
	case overlayHelp:
		switch key {
		case "esc", "?", "enter", "q":
			m.overlay = ""
		}
	case overlayClipboard:
		switch key {
		case "esc":
			m.overlay = ""
		case "up":
			m.clipSel = clampInt(m.clipSel-1, 0, maxInt(0, len(m.clipboard)-1))
		case "down":
			m.clipSel = clampInt(m.clipSel+1, 0, maxInt(0, len(m.clipboard)-1))
		case "enter":
			if len(m.clipboard) > 0 {
				m.overlay = ""
				p := m.focusContentPane()
				if p == nil || p.sourceID == "" {
					m.toast = "clipboard: no focused terminal"
					return nil
				}
				params := m.clipboardPasteParams(p)
				if len(m.clipboardIDs) > m.clipSel {
					params.ClipboardId = m.clipboardIDs[m.clipSel]
				}
				return m.emit("clipboard.paste", params, opMsg{op: "paste"})
			}
		}
	case overlayConnections:
		switch key {
		case "esc":
			m.overlay = ""
		case "up":
			m.connSel = clampInt(m.connSel-1, 0, maxInt(0, len(m.connections)-1))
		case "down":
			m.connSel = clampInt(m.connSel+1, 0, maxInt(0, len(m.connections)-1))
		case "enter", "t":
			// Legacy system.open_connections test action: endpoint.test only
			// succeeds for a registered endpoint (health != unknown).
			return m.connTest()
		case "r":
			// Legacy reconnect action.
			return m.connReconnect()
		}
	}
	return nil
}

// ---------------------------------------------------------- picker tag view

// emptyPaneActions is the focused empty panel's CTA list, mirroring main's
// EmptyPaneCTAActions order: attach, create, manager, close.
var emptyPaneActions = []struct {
	label  string
	action string
}{
	{"Attach existing terminal", "empty-attach"},
	{"Create new terminal", "empty-create"},
	{"Open terminal manager", "empty-manager"},
	{"Close pane", "close"},
}

// focusedEmptyPane returns the focused pane when it is an unbound empty panel,
// which owns the CTA keyboard selection.
func (m *model) focusedEmptyPane() *pane {
	p := m.focusContentPane()
	if p == nil || p.sourceID != "" {
		return nil
	}
	return p
}

func (m *model) moveEmptyPaneSelection(delta int) {
	count := len(emptyPaneActions)
	m.emptyPaneSel = (m.emptyPaneSel + delta + count) % count
}

func (m *model) runEmptyPaneSelection(p *pane) app.Cmd {
	if p == nil {
		return nil
	}
	index := clampInt(m.emptyPaneSel, 0, len(emptyPaneActions)-1)
	return m.activateEmptyPaneAction(p, emptyPaneActions[index].action)
}

// activateEmptyPaneAction runs one empty-panel CTA (shared by keyboard Enter
// and mouse click).
func (m *model) activateEmptyPaneAction(p *pane, action string) app.Cmd {
	if p == nil {
		return nil
	}
	switch action {
	case "empty-attach":
		m.focusPaneObject(p)
		m.openPicker()
	case "empty-create":
		m.focusPaneObject(p)
		m.openPickerForCreate()
	case "empty-manager":
		m.focusPaneObject(p)
		m.openPicker()
		m.toast = "terminal manager: choose a terminal"
	case "close":
		_, t := m.tabOfPane(p)
		m.closePane(t, p)
	default:
		return nil
	}
	return nil
}

func (m *model) openPickerTags() {
	m.pickerTagsOpen = true
	m.pickerTagSel = 0
	m.pickerTagQuery = ""
}

// handlePickerTagsKey is the main terminal_picker_tags scene: a checkbox list
// with Space toggling the highlighted tag and Ctrl-T/Esc returning to a list.
func (m *model) handlePickerTagsKey(key, char string) app.Cmd {
	options := m.pickerTagOptions()
	switch key {
	case "ctrl-t", "esc":
		m.pickerTagsOpen = false
		m.picker = 0
	case "up":
		m.pickerTagSel = clampInt(m.pickerTagSel-1, 0, maxInt(0, len(options)-1))
	case "down":
		m.pickerTagSel = clampInt(m.pickerTagSel+1, 0, maxInt(0, len(options)-1))
	case "space", "enter":
		m.togglePickerTag(m.pickerTagSel)
	case "backspace":
		if m.pickerTagQuery != "" {
			r := []rune(m.pickerTagQuery)
			m.pickerTagQuery = string(r[:len(r)-1])
			m.pickerTagSel = 0
		}
	case "delete":
		if m.pickerTagQuery != "" {
			r := []rune(m.pickerTagQuery)
			m.pickerTagQuery = string(r[:len(r)-1])
			m.pickerTagSel = 0
		}
	case "/":
		m.pickerTagQuery = ""
		m.pickerTagSel = 0
	default:
		if char != "" && len([]rune(char)) == 1 && !strings.HasPrefix(key, "ctrl-") {
			m.pickerTagQuery += char
			m.pickerTagSel = 0
		}
	}
	return nil
}

func (m *model) togglePickerTag(index int) {
	options := m.pickerTagOptions()
	if index < 0 || index >= len(options) {
		return
	}
	label := options[index].label
	for i, existing := range m.pickerTags {
		if existing == label {
			m.pickerTags = append(m.pickerTags[:i], m.pickerTags[i+1:]...)
			return
		}
	}
	m.pickerTags = append(m.pickerTags, label)
	sort.Strings(m.pickerTags)
}

type pickerTagOption struct {
	label   string
	count   int
	checked bool
}

// pickerTagOptions derives the tag checkbox list for the selected endpoint,
// keeping selected filters visible even when their count drops to zero.
func (m *model) pickerTagOptions() []pickerTagOption {
	endpoint := m.pickerTabName()
	counts := map[string]int{}
	for _, src := range m.terminals() {
		if endpointOf(src) != endpoint {
			continue
		}
		for _, label := range publicTagLabels(src) {
			counts[label]++
		}
	}
	for _, selected := range m.pickerTags {
		if _, ok := counts[selected]; !ok {
			counts[selected] = 0
		}
	}
	query := strings.ToLower(strings.TrimSpace(m.pickerTagQuery))
	labels := make([]string, 0, len(counts))
	for label := range counts {
		if query == "" || strings.Contains(strings.ToLower(label), query) {
			labels = append(labels, label)
		}
	}
	sort.Strings(labels)
	options := make([]pickerTagOption, 0, len(labels))
	for _, label := range labels {
		checked := false
		for _, selected := range m.pickerTags {
			if selected == label {
				checked = true
				break
			}
		}
		options = append(options, pickerTagOption{label: label, count: counts[label], checked: checked})
	}
	return options
}

// publicTagLabels returns the display labels of a source's position tags; the
// host stores them as tag1..tagN.
func publicTagLabels(src *pb.Source) []string {
	tags := src.GetTags()
	if len(tags) == 0 {
		return nil
	}
	labels := make([]string, 0, len(tags))
	for key, value := range tags {
		if strings.HasPrefix(key, "tag") {
			if label := strings.TrimSpace(value); label != "" {
				labels = append(labels, label)
			}
		}
	}
	sort.Strings(labels)
	return labels
}

// ------------------------------------------------------------------- mouse

func (m *model) onMouse(ev *pb.MouseEvent) app.Cmd {
	action := ev.GetAction()
	node := ev.GetNode()
	x, y := int(ev.GetX()), int(ev.GetY())
	switch action {
	case "release":
		m.dragging = ""
		return nil
	case "drag":
		if strings.HasPrefix(m.dragging, "copy:") {
			paneID := strings.TrimPrefix(m.dragging, "copy:")
			if p := m.paneByID(paneID); p != nil {
				m.placeCopyCursorAtMouse(p, m.copyFor(p), x, y)
			}
			return nil
		}
		m.dragMove(x, y)
		return nil
	}
	// Copy sessions are per pane; a press only changes the active pane, so
	// input ownership moves with it while the other pane keeps its scrollback
	// position (the old CopyModeByView model).
	return m.handlePress(node, x, y)
}

func (m *model) handlePress(node string, x, y int) app.Cmd {
	if strings.HasPrefix(node, "divider:") {
		m.dragging = node
		m.dragLastX, m.dragLastY = x, y
		return nil
	}
	if strings.HasPrefix(node, "float:") && strings.HasSuffix(node, ":title") {
		parts := strings.SplitN(node, ":", 3)
		m.dragging = parts[1]
		m.dragLastX, m.dragLastY = x, y
		m.raiseFloating(m.floatingByID(m.dragging))
		return nil
	}
	if strings.HasPrefix(node, "picker:tab:") {
		index := atoiNode(node, "picker:tab:")
		if index >= 0 && index < len(m.pickerTabs()) {
			m.pickerTab = index
			m.picker = 0
		}
		return nil
	}
	if strings.HasPrefix(node, "picker-tag:") {
		index := atoiNode(node, "picker-tag:")
		m.pickerTagSel = index
		m.togglePickerTag(index)
		return nil
	}
	if strings.HasPrefix(node, "picker-status:") {
		m.pickerFilter = atoiNode(node, "picker-status:")
		m.picker = 0
		return nil
	}
	if strings.HasPrefix(node, "connection:") && m.overlay == overlayConnections {
		// Click selects the row; Enter/t/r act on it (mirrors picker rows).
		index := atoiNode(node, "connection:")
		if index >= 0 && index < len(m.connections) {
			m.connSel = index
		}
		return nil
	}
	if strings.HasPrefix(node, "prompt-field:") && m.overlay == overlayPrompt && m.promptKind == "terminal.create" {
		index := atoiNode(node, "prompt-field:")
		if index >= 0 && index < len(m.promptFields) {
			m.ensurePromptCursors()
			m.promptField = index
			// The row hitbox already identifies the field. Keep the caret at the
			// end, matching the old form's mouse-focus behavior.
			m.promptCursors[index] = len([]rune(m.promptFields[index]))
			m.promptError = ""
			m.clearPromptSuggestions()
			return m.refreshCreateSuggestions(false)
		}
		return nil
	}
	if strings.HasPrefix(node, "prompt-suggestion:") && m.overlay == overlayPrompt && m.promptKind == "terminal.create" {
		index := atoiNode(node, "prompt-suggestion:")
		if index >= 0 && index < len(m.promptSuggestions) {
			m.promptField = m.promptSuggestionField
			m.promptSuggestionSel = index
			return m.handleCreateSuggestionKey("enter", "")
		}
		return nil
	}
	if node == "picker-tags" {
		m.openPickerTags()
		return nil
	}
	if p := m.paneByID(node); p != nil {
		for _, f := range m.floatings {
			if f.pane == p {
				if f.collapsed {
					return nil
				}
				m.raiseFloating(f)
				m.mode = modeLive
				m.activeFloat = f.id
				return nil
			}
		}
		if m.copyActive() && m.focusContentPane() == p {
			st := m.copyFor(p)
			m.placeCopyCursorAtMouse(p, st, x, y)
			m.markCopyAtCursor(st)
			m.dragging = "copy:" + p.id
			return nil
		}
		m.activeFloat = ""
		m.focusPaneObject(p)
		return nil
	}
	switch {
	case strings.HasPrefix(node, "hdr:tabclose:"):
		m.closeTab(atoiNode(node, "hdr:tabclose:"))
	case strings.HasPrefix(node, "hdr:tab:"):
		index := atoiNode(node, "hdr:tab:")
		if index >= 0 && index < len(m.ws().tabs) {
			m.ws().active = index
			m.mode = modeLive
		}
	case node == "hdr:create":
		m.newTab()
	case node == "hdr:workspace":
		m.mode = modeLive
		m.overlay = overlayHelp
	case strings.HasPrefix(node, "pane:") || strings.HasPrefix(node, "float:") || node == "toast":
		return m.handleChromeClick(node)
	}
	return nil
}

func (m *model) handleChromeClick(node string) app.Cmd {
	if node == "toast" {
		m.toast = ""
		return nil
	}
	parts := strings.Split(node, ":")
	if len(parts) < 3 {
		return nil
	}
	switch parts[0] {
	case "pane":
		p := m.paneByID(parts[1])
		if p == nil {
			return nil
		}
		// Every pane chrome click selects the pane first. Empty panels have no
		// terminal focus box, so without this the action would run while the
		// previous panel kept the accent border.
		m.activeFloat = ""
		m.focusPaneObject(p)
		action := strings.Join(parts[2:], ":")
		switch action {
		case "focus":
			return nil
		case "empty-attach":
			return m.activateEmptyPaneAction(p, "empty-attach")
		case "empty-create":
			return m.activateEmptyPaneAction(p, "empty-create")
		case "empty-manager":
			return m.activateEmptyPaneAction(p, "empty-manager")
		case "close":
			_, t := m.tabOfPane(p)
			m.closePane(t, p)
			return nil
		case "split-v":
			return m.splitPaneFor("row", p)
		case "split-h":
			return m.splitPaneFor("col", p)
		case "zoom":
			m.toggleZoom(p)
			return nil
		case "lock":
			p.locked = !p.locked
			return nil
		case "take-owner":
			return m.takeOwner(p)
		}
	case "float":
		f := m.floatingByID(parts[1])
		if f == nil {
			return nil
		}
		action := parts[2]
		m.raiseFloating(f)
		switch action {
		case "lock":
			f.pane.locked = !f.pane.locked
		case "close":
			m.closeFloating(f)
		case "collapse":
			m.toggleFloatingCollapse(f)
		case "center":
			m.centerFloating(f)
		case "zoom":
			m.maximizeFloating(f)
		}
	}
	return nil
}

func (m *model) onWheel(ev *pb.WheelEvent) app.Cmd {
	node := ev.GetNode()
	delta := int(ev.GetDelta())
	p := m.paneByID(node)
	if p == nil {
		return nil
	}
	src := m.paneSource(p)
	if src != nil && src.GetTerminalId() != "" {
		// Wheel = interact with the pane under the cursor: activate it first
		// (the active view owns copy input), then apply the old wheel rules:
		// up enters/scrolls older, down only continues an open session.
		if m.focusPane() != p {
			m.focusPaneObject(p)
		}
		st := m.copyFor(p)
		if st == nil {
			if delta < 0 {
				return nil
			}
			st = &copyState{offset: 0}
			m.copyPanes[p.id] = st
			m.prepareCopyViewport(p, st)
			// The window is in flight; remember the wheel movement so the
			// cursor/selection follow once it arrives.
			st.pendingRows = -delta
			return m.fetchCopyWindow(p, st)
		}
		return m.moveCopyCursorRows(p, st, -delta)
	}
	p.scroll = clampInt(p.scroll-delta, 0, len(p.lines))
	return nil
}

// ---------------------------------------------------------------- actions

func (m *model) focusPaneObject(p *pane) {
	index, t := m.tabOfPane(p)
	if t == nil {
		return
	}
	m.ws().active = index
	for i, candidate := range t.panes {
		if candidate == p {
			t.focus = i
		}
	}
	m.markWorkbenchDirty()
}

func (m *model) focusPaneDelta(delta int) {
	t := m.activeTab()
	if t != nil && len(t.panes) > 0 {
		t.focus = (t.focus + delta + len(t.panes)) % len(t.panes)
	}
	m.activeFloat = ""
	m.markWorkbenchDirty()
}

func (m *model) splitPane(flow string) app.Cmd { return m.splitPaneFor(flow, nil) }

// splitLeafFor replaces the focused (or given) leaf with Split(original,
// new); sibling nodes keep their rects and ratios. It returns the new leaf's
// pane (nil when the target is not in the active tab).
func (m *model) splitLeafFor(flow string, target *pane) *pane {
	t := m.activeTab()
	if t == nil {
		return nil
	}
	source := target
	if source == nil {
		source = m.focusPane()
	}
	if source == nil {
		return nil
	}
	lf := t.leafOf(source)
	if lf == nil {
		return nil
	}
	clone := m.newPane("", nil)
	sp := &split{orient: flow, ratio: 0.5, a: lf, b: &leaf{pane: clone}, seq: t.nextSplitSeq()}
	m.replaceLeaf(t, lf, sp)
	t.rebuild()
	for i, candidate := range t.panes {
		if candidate == clone {
			t.focus = i
		}
	}
	m.mode = modePane
	m.activeFloat = ""
	m.markWorkbenchDirty()
	return clone
}

// splitPaneFor creates an empty panel. The panel itself offers attach/create/
// close actions, matching the main workbench lifecycle.
func (m *model) splitPaneFor(flow string, target *pane) app.Cmd {
	clone := m.splitLeafFor(flow, target)
	if clone == nil {
		return nil
	}
	if !m.demo {
		m.toast = "empty panel created · choose a terminal or create one"
	}
	return nil
}

func (m *model) replaceLeaf(t *tab, target *leaf, replacement treeNode) bool {
	if t.root == target {
		t.root = replacement
		return true
	}
	var walk func(node treeNode) bool
	walk = func(node treeNode) bool {
		sp, ok := node.(*split)
		if !ok {
			return false
		}
		if sp.a == target {
			sp.a = replacement
			return true
		}
		if sp.b == target {
			sp.b = replacement
			return true
		}
		return walk(sp.a) || walk(sp.b)
	}
	return walk(t.root)
}

func (m *model) closePane(t *tab, p *pane) {
	if t == nil || p == nil {
		return
	}
	target := t.leafOf(p)
	if target == nil {
		return
	}
	// Closing the source's owner drops the designation so another pane on that
	// source can take over (legacy owner release on view close).
	m.forgetOwner(p)
	index := 0
	for i, candidate := range t.panes {
		if candidate == p {
			index = i
		}
	}
	var remove func(node treeNode) treeNode
	remove = func(node treeNode) treeNode {
		if node == target {
			return nil
		}
		sp, ok := node.(*split)
		if !ok {
			return node
		}
		a := remove(sp.a)
		b := remove(sp.b)
		switch {
		case a == nil && b == nil:
			return nil
		case a == nil:
			return b
		case b == nil:
			return a
		}
		sp.a, sp.b = a, b
		return sp
	}
	delete(m.copyPanes, p.id)
	t.root = remove(t.root)
	if t.root == nil {
		t.root = &leaf{pane: m.newPane("empty", nil)}
	}
	t.rebuild()
	if index > len(t.panes)-1 {
		index = len(t.panes) - 1
	}
	if index < 0 {
		index = 0
	}
	t.focus = index
	if m.zoomPane == p.id {
		m.zoomPane = ""
	}
	m.markWorkbenchDirty()
}

func (m *model) killClosePane(t *tab, p *pane) app.Cmd {
	cmd := m.killPane(p)
	m.closePane(t, p)
	return cmd
}

func (m *model) killPane(p *pane) app.Cmd {
	src := m.paneSource(p)
	if src == nil || src.GetTerminalId() == "" {
		m.toast = "no terminal to kill"
		return nil
	}
	return m.emit("terminal.kill", &pb.MethodParams{
		Endpoint: endpointOf(src), Id: src.GetTerminalId(),
	}, opMsg{op: "kill"})
}

func (m *model) killSource(src *pb.Source) app.Cmd {
	if src == nil || src.GetTerminalId() == "" {
		return nil
	}
	return m.emit("terminal.kill", &pb.MethodParams{
		Endpoint: endpointOf(src), Id: src.GetTerminalId(),
	}, opMsg{op: "kill"})
}

func (m *model) removeSource(src *pb.Source) app.Cmd {
	if src == nil || src.GetTerminalId() == "" {
		return nil
	}
	return m.emit("terminal.remove", &pb.MethodParams{
		Endpoint: endpointOf(src), Id: src.GetTerminalId(),
	}, opMsg{op: "remove"})
}

func (m *model) restartPane(p *pane) app.Cmd {
	src := m.paneSource(p)
	if src == nil || src.GetTerminalId() == "" {
		m.toast = "no terminal to restart"
		return nil
	}
	p.pending = "restart"
	return m.emit("terminal.restart", &pb.MethodParams{
		Endpoint: endpointOf(src), Id: src.GetTerminalId(),
	}, opMsg{op: "restart", ref: p.id})
}

func (m *model) takeOwner(p *pane) app.Cmd {
	if p == nil {
		return nil
	}
	if src := m.paneSource(p); src != nil && src.GetTerminalId() != "" {
		// Manual ownership (legacy panel.take_owner): record this pane as the
		// source's resize owner. On success the pane stays owner even though
		// focus may sit on a follower; focus never changes ownership.
		p.pending = "owner"
		if m.ownerPaneBySource == nil {
			m.ownerPaneBySource = map[string]string{}
		}
		m.ownerPaneBySource[p.sourceID] = p.id
		fit := true
		expected := src.GetOwnerEpoch()
		return m.emit("terminal.attach", &pb.MethodParams{
			Endpoint: endpointOf(src), Id: src.GetTerminalId(), Fit: &fit, ExpectedOwnerEpoch: &expected,
		}, opMsg{op: "owner", ref: p.id})
	}
	m.toast = "resize owner: host arbitrates"
	return nil
}

// forgetOwner drops the manual ownership record when p was its source's owner,
// so sourceOwnerPane can fall back to another pane still bound to that source.
// It is called wherever a pane's sourceID is cleared (close, detach, source
// disappearance).
func (m *model) forgetOwner(p *pane) {
	if p == nil || p.sourceID == "" || m.ownerPaneBySource == nil {
		return
	}
	if m.ownerPaneBySource[p.sourceID] == p.id {
		delete(m.ownerPaneBySource, p.sourceID)
	}
}

func (m *model) toggleZoom(p *pane) {
	if p == nil {
		return
	}
	if m.zoomPane == p.id {
		m.zoomPane = ""
		return
	}
	m.zoomPane = p.id
}

func (m *model) toggleLayout(t *tab) {
	if t == nil {
		return
	}
	p := m.focusPane()
	if p == nil {
		return
	}
	lf := t.leafOf(p)
	if lf == nil {
		return
	}
	var best *split
	var walk func(node treeNode)
	walk = func(node treeNode) {
		sp, ok := node.(*split)
		if !ok {
			return
		}
		if m.subtreeHasLeaf(sp.a, lf) || m.subtreeHasLeaf(sp.b, lf) {
			best = sp
		}
		if m.subtreeHasLeaf(sp.a, lf) {
			walk(sp.a)
		} else {
			walk(sp.b)
		}
	}
	if t.root != nil {
		walk(t.root)
	}
	if best == nil {
		return
	}
	if best.orient == "row" {
		best.orient = "col"
	} else {
		best.orient = "row"
	}
	m.markWorkbenchDirty()
}

func (m *model) subtreeHasLeaf(node treeNode, target *leaf) bool {
	switch n := node.(type) {
	case nil:
		return false
	case *leaf:
		return n == target
	case *split:
		return m.subtreeHasLeaf(n.a, target) || m.subtreeHasLeaf(n.b, target)
	}
	return false
}

// ancestorSplit returns the deepest Split containing the leaf on the wanted
// axis, whether the leaf sits in its a child, and the split's rect.
func (m *model) ancestorSplit(t *tab, target *leaf, orient string) (treeNode, bool) {
	m.splitEntries(t)
	var best *split
	var inA bool
	var walk func(node treeNode, depth int)
	walk = func(node treeNode, depth int) {
		sp, ok := node.(*split)
		if !ok {
			return
		}
		aHas := m.subtreeHasLeaf(sp.a, target)
		bHas := m.subtreeHasLeaf(sp.b, target)
		if sp.orient == orient && (aHas || bHas) {
			if best == nil {
				best, inA = sp, aHas
			}
		}
		if aHas {
			walk(sp.a, depth+1)
		} else if bHas {
			walk(sp.b, depth+1)
		}
	}
	if t != nil && t.root != nil {
		walk(t.root, 0)
	}
	return best, inA
}

// centerFocused evens the focused split axis (legacy resize.center for tiled
// panes without a content letterbox): clear the additive bias so the default
// half is authoritative.
func (m *model) centerFocused() {
	t := m.activeTab()
	p := m.focusPane()
	if t == nil || p == nil {
		return
	}
	lf := t.leafOf(p)
	if lf == nil {
		return
	}
	var best *split
	var walk func(node treeNode)
	walk = func(node treeNode) {
		sp, ok := node.(*split)
		if !ok {
			return
		}
		if m.subtreeHasLeaf(sp.a, lf) || m.subtreeHasLeaf(sp.b, lf) {
			best = sp
		}
		if m.subtreeHasLeaf(sp.a, lf) {
			walk(sp.a)
		} else {
			walk(sp.b)
		}
	}
	if t.root != nil {
		walk(t.root)
	}
	if best == nil || m.subtreeLocked(best) {
		return
	}
	best.ratio = 0.5
	best.bias = 0
	m.markWorkbenchDirty()
}

func (m *model) resetTabSplits(t *tab) {
	if t == nil {
		return
	}
	var walk func(node treeNode)
	walk = func(node treeNode) {
		sp, ok := node.(*split)
		if !ok {
			return
		}
		if m.subtreeLocked(sp) {
			return
		}
		// panel.balance / resize.layout_reset clear every hint: 0.5 is an even
		// split and 0 bias reproduces the default geometry exactly.
		sp.ratio = 0.5
		sp.bias = 0
		walk(sp.a)
		walk(sp.b)
	}
	walk(t.root)
	m.markWorkbenchDirty()
}

func (m *model) resizeFocused(delta int, vertical bool) {
	t := m.activeTab()
	p := m.focusPane()
	if t == nil || p == nil {
		return
	}
	lf := t.leafOf(p)
	if lf == nil {
		return
	}
	// A locked pane keeps its size, and a split containing a locked sibling
	// cannot be moved because changing its ratio would resize that pane too.
	if p.locked {
		m.toast = "panel size locked"
		return
	}
	orient := "row"
	if vertical {
		orient = "col"
	}
	node, inA := m.ancestorSplit(t, lf, orient)
	sp, ok := node.(*split)
	if !ok || sp == nil {
		return
	}
	if m.subtreeLocked(sp) {
		m.toast = "panel size locked"
		return
	}
	// Legacy resizeSplitNode accumulates an additive BiasCells on the first
	// child and clears any Ratio hint, so the bias fully determines the new
	// extent. Moving the divider toward the focused side shrinks it, so the
	// sign is relative to which child holds the focused pane (resizeBiasDelta),
	// and the v3shell deltas already encode the divider direction.
	if inA {
		sp.bias += delta
	} else {
		sp.bias -= delta
	}
	sp.ratio = 0
	m.markWorkbenchDirty()
}

func (m *model) subtreeLocked(node treeNode) bool {
	switch n := node.(type) {
	case *leaf:
		return n != nil && n.pane != nil && n.pane.locked
	case *split:
		return m.subtreeLocked(n.a) || m.subtreeLocked(n.b)
	default:
		return false
	}
}

func (m *model) newTab() {
	ws := m.ws()
	m.tabSeq++
	p := m.newPane("empty", nil)
	ws.tabs = append(ws.tabs, makeTab(fmt.Sprintf("tab-%d", m.tabSeq), fmt.Sprintf("tab-%d", m.tabSeq), []*pane{p}, "row"))
	ws.active = len(ws.tabs) - 1
	m.mode = modeLive
	m.markWorkbenchDirty()
	if !m.demo {
		m.openPicker()
	}
}

func (m *model) closeTab(index int) {
	ws := m.ws()
	if index < 0 || index >= len(ws.tabs) || len(ws.tabs) == 1 {
		return
	}
	ws.tabs = append(ws.tabs[:index], ws.tabs[index+1:]...)
	if ws.active >= len(ws.tabs) {
		ws.active = len(ws.tabs) - 1
	}
	m.markWorkbenchDirty()
}

func (m *model) killTab(index int) app.Cmd {
	ws := m.ws()
	if index < 0 || index >= len(ws.tabs) {
		return nil
	}
	var cmds []app.Cmd
	for _, p := range ws.tabs[index].panes {
		cmds = append(cmds, m.killPane(p))
	}
	m.closeTab(index)
	return chain(cmds...)
}

func (m *model) createWorkspace() {
	m.spaces = append(m.spaces, &workspace{name: fmt.Sprintf("ws-%d", len(m.spaces)+1)})
	m.space = len(m.spaces) - 1
	p := m.newPane("empty", nil)
	m.ws().tabs = []*tab{makeTab("tab-1", "main", []*pane{p}, "row")}
	m.ws().active = 0
	m.markWorkbenchDirty()
}

func (m *model) deleteWorkspace(index int) {
	if index < 0 || index >= len(m.spaces) || len(m.spaces) == 1 {
		return
	}
	m.spaces = append(m.spaces[:index], m.spaces[index+1:]...)
	if m.space >= len(m.spaces) {
		m.space = len(m.spaces) - 1
	}
	m.markWorkbenchDirty()
}

func (m *model) openPicker() {
	m.overlay = overlayPicker
	m.picker = 0
	m.pickerQuery = ""
	// main defaults the picker to Running terminals; Exited/All need an
	// explicit Shift+Left/Right.
	m.pickerFilter = pickerFilterRunning
	m.pickerTagsOpen = false
	m.pickerTagSel = 0
	m.pickerTagQuery = ""
	if target := m.attachTarget(); target != nil {
		if src := m.paneSource(target); src != nil {
			for i, tab := range m.pickerTabs() {
				if tab.name == endpointOf(src) {
					m.pickerTab = i
					break
				}
			}
		}
	}
}

func (m *model) openPickerForCreate() {
	m.openPicker()
	// The create row is the first selectable row.
	for i, row := range m.pickerRows() {
		if row.create {
			m.picker = i
			break
		}
	}
}

func (m *model) openPrompt() {
	m.overlay = overlayPrompt
	m.prompt = ""
	m.promptCursor = 0
	m.promptSel = 0
	m.promptKind = "command"
	m.promptRef = ""
	m.promptEndpoint = ""
	m.promptFields = nil
	m.promptCursors = nil
	m.promptField = 0
	m.promptError = ""
	m.clearPromptSuggestions()
}

func (m *model) openRename(kind, ref, current string) {
	m.overlay = overlayPrompt
	m.prompt = current
	m.promptCursor = len([]rune(current))
	m.promptSel = 0
	m.promptKind = "rename"
	if kind == "terminal.rename" {
		m.promptKind = kind
	}
	m.promptRef = ref
	m.promptFields = nil
	m.promptCursors = nil
	m.promptField = 0
	m.promptError = ""
	m.clearPromptSuggestions()
}

func (m *model) applyRename() app.Cmd {
	value := strings.TrimSpace(m.prompt)
	var cmd app.Cmd
	if value != "" {
		switch m.promptKind {
		case "rename":
			for _, t := range m.ws().tabs {
				if t.id == m.promptRef {
					t.title = value
				}
			}
			if m.promptRef == m.ws().name {
				m.ws().name = value
			}
			m.markWorkbenchDirty()
		case "terminal.rename":
			src := m.sourceByID(m.promptRef)
			if src == nil {
				src = m.paneSource(m.focusContentPane())
			}
			if src != nil {
				cmd = m.emit("terminal.rename", &pb.MethodParams{
					Endpoint: endpointOf(src), Id: terminalIDOf(src, m.promptRef), Title: value,
				}, opMsg{op: "rename"})
			}
		}
	}
	m.overlay = ""
	m.prompt = ""
	m.promptCursor = 0
	m.promptKind = "command"
	return cmd
}

func (m *model) focusContentPane() *pane {
	if m.activeFloat != "" {
		if f := m.floatingByID(m.activeFloat); f != nil && !f.collapsed {
			return f.pane
		}
	}
	return m.focusPane()
}

func (m *model) pasteSystem() app.Cmd {
	p := m.focusContentPane()
	if p == nil || p.sourceID == "" {
		m.toast = "paste: no focused terminal"
		return nil
	}
	return m.emit("clipboard.paste", m.clipboardPasteParams(p), opMsg{op: "paste"})
}

// pasteLatestClipboard pastes the newest clipboard history entry (the legacy
// copy-scene `p` = clipboard.paste_latest). It lists the history first so the
// program learns the newest id, then pastes it when the list response arrives.
func (m *model) pasteLatestClipboard() app.Cmd {
	p := m.focusContentPane()
	if p == nil || p.sourceID == "" {
		m.toast = "paste: no focused terminal"
		return nil
	}
	m.pasteLatestRef = p.id
	return m.emit("clipboard.history.list", nil, opMsg{op: "clipboard.latest", ref: p.id})
}

func (m *model) clipboardPasteParams(p *pane) *pb.MethodParams {
	return m.terminalParamsForSource(p.sourceID)
}

func (m *model) terminalParamsForSource(sourceID string) *pb.MethodParams {
	src := m.sourceByID(sourceID)
	return &pb.MethodParams{Endpoint: endpointOf(src), Id: terminalIDOf(src, sourceID)}
}

func (m *model) openClipboardHistory() app.Cmd {
	m.overlay = overlayClipboard
	m.clipSel = 0
	m.clipboard = nil
	m.clipboardIDs = nil
	return m.emit("clipboard.history.list", nil, opMsg{op: "clipboard.list"})
}

// openConnections opens the SYSTEM connections overlay (legacy
// system.open_connections) and asks the host for the registered endpoint
// table. The response is parsed in onOp under the "connections.list" op.
func (m *model) openConnections() app.Cmd {
	m.overlay = overlayConnections
	m.connSel = 0
	m.connections = nil
	return m.emit("endpoint.list", nil, opMsg{op: "connections.list"})
}

// connSelected returns the highlighted connection, or nil when the table is
// empty (or the selection is out of range).
func (m *model) connSelected() *connectionRow {
	if m.connSel < 0 || m.connSel >= len(m.connections) {
		return nil
	}
	return &m.connections[m.connSel]
}

// storeConnections parses raw endpoint.list JSON rows into connectionRow
// entries and clamps the selection. It is shared by the direct "endpoint.list"
// op and the "connections.list" op the overlay uses.
func (m *model) storeConnections(rows []string) {
	m.connections = nil
	for _, raw := range rows {
		var row struct {
			Name   string `json:"name"`
			Label  string `json:"label"`
			Kind   string `json:"kind"`
			Health string `json:"health"`
		}
		if err := json.Unmarshal([]byte(raw), &row); err == nil && row.Name != "" {
			m.connections = append(m.connections, connectionRow{
				name: row.Name, label: row.Label, kind: row.Kind, health: row.Health,
			})
		}
	}
	m.connSel = clampInt(m.connSel, 0, maxInt(0, len(m.connections)-1))
}

// connTest tests the selected endpoint (legacy connections test key). A
// successful test re-lists so the health column refreshes; the toast reports
// the outcome.
func (m *model) connTest() app.Cmd {
	row := m.connSelected()
	if row == nil {
		m.toast = "connections: nothing selected"
		return nil
	}
	return m.emit("endpoint.test", &pb.MethodParams{Endpoint: row.name},
		opMsg{op: "connections.test", endpoint: row.name})
}

// connReconnect reconnects the selected endpoint (legacy connections
// reconnect key), then re-lists so health reflects the fresh dial.
func (m *model) connReconnect() app.Cmd {
	row := m.connSelected()
	if row == nil {
		m.toast = "connections: nothing selected"
		return nil
	}
	return m.emit("endpoint.reconnect", &pb.MethodParams{Endpoint: row.name},
		opMsg{op: "connections.reconnect", endpoint: row.name})
}

func (m *model) openFloatMenu() {
	for i := len(m.floatings) - 1; i >= 0; i-- {
		if !m.floatings[i].collapsed {
			m.activeFloat = m.floatings[i].id
			return
		}
	}
	if len(m.floatings) > 0 {
		m.activeFloat = m.floatings[len(m.floatings)-1].id
	}
}

func (m *model) activeFloating() *floating {
	for _, f := range m.floatings {
		if f.id == m.activeFloat {
			return f
		}
	}
	return nil
}

func (m *model) floatingByID(id string) *floating {
	for _, f := range m.floatings {
		if f.id == id {
			return f
		}
	}
	return nil
}

func (m *model) raiseFloating(f *floating) {
	if f == nil {
		return
	}
	for i, candidate := range m.floatings {
		if candidate == f {
			m.floatings = append(m.floatings[:i], m.floatings[i+1:]...)
			break
		}
	}
	m.floatings = append(m.floatings, f)
	if !f.collapsed {
		m.activeFloat = f.id
	}
}

func (m *model) toggleFloatingCollapse(f *floating) {
	f.collapsed = !f.collapsed
	if f.collapsed {
		m.dragging = ""
		m.toast = "collapsed " + f.id
	} else {
		m.raiseFloating(f)
	}
}

func (m *model) centerFloating(f *floating) {
	f.x = maxInt(0, (m.cols-f.w)/2)
	f.y = maxInt(1, (m.rows-maxInt(2, f.h))/2)
	m.raiseFloating(f)
}

func (m *model) maximizeFloating(f *floating) {
	f.w = maxInt(20, m.cols-8)
	f.h = maxInt(6, m.rows-4)
	f.x = 4
	f.y = 2
	m.raiseFloating(f)
}

func (m *model) resizeFloating(f *floating, dw, dh int) {
	if f == nil {
		return
	}
	f.w = clampInt(f.w+dw, 8, maxInt(8, m.cols-2))
	f.h = clampInt(f.h+dh, 3, maxInt(3, m.rows-2))
	f.x = clampInt(f.x, 0, maxInt(0, m.cols-f.w))
	f.y = clampInt(f.y, 1, maxInt(1, m.rows-1-f.h))
	m.raiseFloating(f)
}

func (m *model) newFloating() app.Cmd {
	m.floatSeq++
	width := clampInt(m.cols*4/5,
		minInt(64, maxInt(16, m.cols-8)),
		minInt(112, maxInt(16, m.cols-4)))
	height := clampInt(m.rows*3/4,
		minInt(18, maxInt(4, m.rows-4)),
		minInt(32, maxInt(4, m.rows-2)))
	width = minInt(width, maxInt(8, m.cols-2))
	height = minInt(height, maxInt(3, m.rows-2))
	cascade := 0
	for _, f := range m.floatings {
		if !f.collapsed {
			cascade++
		}
	}
	x := clampInt(maxInt(0, (m.cols-width)/2)+cascade*4, 0, maxInt(0, m.cols-width))
	y := clampInt(maxInt(1, (m.rows-height)/2)+cascade, 1, maxInt(1, m.rows-1-height))
	p := m.newPane(fmt.Sprintf("floating-%d", m.floatSeq),
		[]string{"", "  floating pane", "  Ctrl-O z 折叠 · 拖动标题移动"})
	f := &floating{id: fmt.Sprintf("float-%d", m.floatSeq), title: p.title, x: x, y: y, w: width, h: height, pane: p}
	m.floatings = append(m.floatings, f)
	m.activeFloat = f.id
	if !m.demo {
		m.openPicker()
	}
	return nil
}

func (m *model) closeFloating(f *floating) {
	for i, candidate := range m.floatings {
		if candidate == f {
			m.floatings = append(m.floatings[:i], m.floatings[i+1:]...)
			break
		}
	}
	var top *floating
	for i := len(m.floatings) - 1; i >= 0; i-- {
		if !m.floatings[i].collapsed {
			top = m.floatings[i]
			break
		}
	}
	if top != nil {
		m.activeFloat = top.id
	} else {
		m.activeFloat = ""
	}
	if len(m.floatings) == 0 {
		m.mode = modeLive
	}
}

func (m *model) clampFloatings() {
	for _, f := range m.floatings {
		f.w = minInt(f.w, maxInt(8, m.cols-2))
		f.h = minInt(f.h, maxInt(3, m.rows-2))
		f.x = clampInt(f.x, 0, maxInt(0, m.cols-f.w))
		f.y = clampInt(f.y, 1, maxInt(1, m.rows-1-f.h))
	}
}

func (m *model) dragMove(x, y int) {
	if strings.HasPrefix(m.dragging, "divider:") {
		m.dragResize(x, y)
		return
	}
	f := m.floatingByID(m.dragging)
	if f == nil {
		return
	}
	dx := x - m.dragLastX
	dy := y - m.dragLastY
	f.x = clampInt(f.x+dx, 0, maxInt(0, m.cols-f.w))
	f.y = clampInt(f.y+dy, 1, maxInt(1, m.rows-1-f.h))
	m.dragLastX, m.dragLastY = x, y
}

// dragResize writes the dragged boundary back into exactly one Split ratio.
// Event x/y are 1-based SGR cells; the separator sits at 0-based
// origin+first+2, so first = x - origin - 3.
func (m *model) dragResize(x, y int) {
	t := m.activeTab()
	parts := strings.Split(m.dragging, ":")
	if len(parts) < 3 {
		return
	}
	seq, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}
	var target *split
	for _, entry := range m.splitEntries(t) {
		if entry.node.seq == seq {
			target = entry.node
			break
		}
	}
	if target == nil {
		return
	}
	sp := target
	// The divider drag writes an absolute first-extent ratio; clear the
	// additive bias so the ratio is authoritative.
	sp.bias = 0
	if sp.orient == "row" {
		avail := maxInt(2, sp.rect.w)
		first := clampInt(x-sp.rect.x, 1, avail-1)
		sp.ratio = float64(first) / float64(avail)
	} else {
		avail := maxInt(2, sp.rect.h)
		first := clampInt(y-sp.rect.y, 1, avail-1)
		sp.ratio = float64(first) / float64(avail)
	}
	m.markWorkbenchDirty()
}

func (m *model) promptMatches() []string {
	query := strings.ToLower(strings.TrimSpace(m.prompt))
	if query == "" {
		return append([]string(nil), promptCommands...)
	}
	var out []string
	for _, command := range promptCommands {
		if matchSearchValue(command, query) != nil {
			out = append(out, command)
		}
	}
	return out
}

// insertPromptText keeps command and rename prompts editable at an arbitrary
// rune cursor, matching the create-terminal form's field editor.
func (m *model) insertPromptText(text string) {
	if text == "" {
		return
	}
	runes := []rune(m.prompt)
	cursor := clampInt(m.promptCursor, 0, len(runes))
	insert := []rune(text)
	updated := make([]rune, 0, len(runes)+len(insert))
	updated = append(updated, runes[:cursor]...)
	updated = append(updated, insert...)
	updated = append(updated, runes[cursor:]...)
	m.prompt = string(updated)
	m.promptCursor = cursor + len(insert)
}

func (m *model) runCommand(command string) app.Cmd {
	m.overlay = ""
	switch command {
	case "split row":
		return m.splitPane("row")
	case "split col":
		return m.splitPane("col")
	case "close pane":
		t := m.activeTab()
		m.closePane(t, m.focusPane())
	case "new tab":
		m.newTab()
	case "close tab":
		m.closeTab(m.ws().active)
	case "kill pane":
		return m.killPane(m.focusPane())
	case "help":
		m.overlay = overlayHelp
	case "quit":
		return m.quitCmd()
	}
	return nil
}

// ------------------------------------------------------------- picker model

// pickerTab is one endpoint tab of the terminal picker: the old picker
// partitions terminals by machine (endpoint) with a tab per endpoint.
type pickerTab struct {
	name   string
	label  string
	status string
	count  int
}

// pickerTabs derives the endpoint tabs from the sources snapshot: endpoint
// placeholder sources carry the machine label and health, terminals add the
// per-endpoint counts. `local` always sorts first.
func (m *model) pickerTabs() []pickerTab {
	index := map[string]int{}
	var tabs []pickerTab
	add := func(name, label, status string) int {
		if at, ok := index[name]; ok {
			if label != "" && tabs[at].label == "" {
				tabs[at].label = label
			}
			if status != "" {
				tabs[at].status = status
			}
			return at
		}
		index[name] = len(tabs)
		tabs = append(tabs, pickerTab{name: name, label: label, status: status})
		return len(tabs) - 1
	}
	for _, src := range m.sources {
		switch src.GetKind() {
		case "endpoint":
			name := strings.TrimSpace(src.GetEndpoint())
			if name == "" {
				name = strings.TrimPrefix(src.GetId(), "endpoint:")
			}
			if name != "" {
				add(name, strings.TrimSpace(src.GetTitle()), src.GetHealth())
			}
		case "terminal":
			if src.GetId() == "" {
				continue
			}
			// The endpoint label (machine name) is carried on the terminal
			// source itself; fall back to the raw endpoint id.
			at := add(endpointOf(src), strings.TrimSpace(src.GetEndpointLabel()), src.GetHealth())
			tabs[at].count++
		}
	}
	if len(tabs) == 0 {
		add("local", "", "")
	}
	// local first, everything else keeps the snapshot order.
	for i := 1; i < len(tabs); i++ {
		if tabs[i].name == "local" {
			local := tabs[i]
			tabs = append(tabs[:i], tabs[i+1:]...)
			tabs = append([]pickerTab{local}, tabs...)
			break
		}
	}
	for i := range tabs {
		if tabs[i].label == "" {
			tabs[i].label = tabs[i].name
		}
		if tabs[i].status == "" && (tabs[i].count > 0 || tabs[i].name == "local") {
			tabs[i].status = "ok"
		}
	}
	return tabs
}

func (m *model) pickerTabName() string {
	tabs := m.pickerTabs()
	m.pickerTab = clampInt(m.pickerTab, 0, len(tabs)-1)
	return tabs[m.pickerTab].name
}

// pickerRow is one selectable terminal-picker row (a terminal of the selected
// endpoint, or the trailing `+ New terminal`).
type pickerRow struct {
	source *pb.Source
	create bool
}

func (m *model) pickerRows() []pickerRow {
	endpoint := m.pickerTabName()
	var rows []pickerRow
	for _, src := range m.terminals() {
		if endpointOf(src) != endpoint {
			continue
		}
		if m.pickerFilter == pickerFilterRunning && src.GetExited() {
			continue
		}
		if m.pickerFilter == pickerFilterExited && !src.GetExited() {
			continue
		}
		if !m.pickerMatchesTags(src) {
			continue
		}
		if !pickerSourceMatches(src, m.pickerQuery) {
			continue
		}
		rows = append(rows, pickerRow{source: src})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		li := strings.ToLower(strings.TrimSpace(rows[i].source.GetTitle()))
		lj := strings.ToLower(strings.TrimSpace(rows[j].source.GetTitle()))
		return li < lj
	})
	if m.pickerFilter != pickerFilterExited && pickerCreateRowMatches(m.pickerQuery) {
		rows = append([]pickerRow{{create: true}}, rows...)
	}
	return rows
}

// pickerCreateRowMatches reports whether the "+ New terminal" row should be
// visible for the query, using main's create-row haystack.
func pickerCreateRowMatches(query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}
	for _, value := range []string{"New terminal", "create terminal", "new terminal"} {
		if matchSearchValue(value, query) != nil {
			return true
		}
	}
	return false
}

// pickerMatchesTags applies the Ctrl-T tag checkbox filter. With no tags
// selected every terminal matches.
func (m *model) pickerMatchesTags(src *pb.Source) bool {
	if len(m.pickerTags) == 0 {
		return true
	}
	present := map[string]bool{}
	for _, key := range []string{"tag", "tag1", "tag2", "tag3", "tag4", "tag5"} {
		if label := strings.TrimSpace(src.GetTags()[key]); label != "" {
			present[label] = true
		}
	}
	for _, tag := range m.pickerTags {
		if !present[tag] {
			return false
		}
	}
	return true
}

// endpointGlyph maps an endpoint health string to the recommended yaml
// `chrome.picker.endpoint_status` glyph/style pair.
func endpointGlyph(status string) (string, string) {
	switch strings.ToLower(status) {
	case "ok":
		return glyphRunning, stSuccess
	case "connecting":
		return "◐", stAccent
	case "offline":
		return "×", stWarning
	case "reconnect_required":
		return "!", stWarning
	case "unregistered":
		return "?", stWarning
	}
	return "○", stMuted
}

func (m *model) attachTarget() *pane {
	if m.mode == modeFloating && m.activeFloat != "" {
		if f := m.floatingByID(m.activeFloat); f != nil && !f.collapsed {
			return f.pane
		}
	}
	return m.focusPane()
}

func (m *model) attach(index int, split bool) app.Cmd {
	rows := m.pickerRows()
	if index >= 0 && index < len(rows) && rows[index].source != nil {
		src := rows[index].source
		p := m.attachTarget()
		if p == nil {
			return nil
		}
		if split {
			p = m.splitLeafFor("row", p)
			if p == nil {
				return nil
			}
		}
		fit := true
		params := &pb.MethodParams{
			Endpoint: endpointOf(src), Id: src.GetTerminalId(), Fit: &fit,
		}
		m.addOwnerEpoch(p, src, &fit, params)
		cmd := m.bindPending(p.id, src.GetId(), params)
		m.overlay = ""
		m.toast = "attach requested \u00b7 " + src.GetTerminalId()
		return cmd
	}
	p := m.attachTarget()
	if p == nil {
		return nil
	}
	// main opens a terminal.create form for the + New terminal row; creation
	// happens only after the form is submitted.
	m.overlay = overlayPrompt
	m.promptKind = "terminal.create"
	m.promptRef = p.id
	m.promptPublicTagList = true
	endpoint := m.pickerTabName()
	m.promptEndpoint = ""
	// Keep the same field order and endpoint-aware routing as main's form:
	// name, command, server, workdir, tags.
	m.promptFields = []string{"", "", m.createEndpointPromptValue(endpoint), "", ""}
	// Defaults are loaded from the owning endpoint. Keep the command field
	// empty until that response arrives so a local shell never leaks into a
	// remote create form.
	m.promptDefaults = createDefaults{}
	if m.createDrafts == nil {
		m.createDrafts = map[string]createDraft{}
	}
	if draft, ok := m.createDrafts[endpoint]; ok {
		m.promptFields[1] = draft.command
		m.promptFields[3] = draft.cwd
	}
	m.promptCursors = make([]int, len(m.promptFields))
	for i := range m.promptFields {
		m.promptCursors[i] = len([]rune(m.promptFields[i]))
	}
	m.promptField = 0
	m.promptError = ""
	m.clearPromptSuggestions()
	return m.requestCreateDefaults(endpoint)
}

func (m *model) bindPending(paneID, sourceID string, params *pb.MethodParams) app.Cmd {
	method := "terminal.attach"
	if sourceID == "" {
		method = "terminal.create"
	}
	if p := m.paneByID(paneID); p != nil {
		p.pending = "bind"
		if params != nil && fitRequested(params) && sourceID != "" {
			fit := fitRequested(params)
			m.addOwnerEpoch(p, m.sourceByID(sourceID), &fit, params)
		}
	}
	return m.emit(method, params, opMsg{
		op: "bind", ref: paneID, source: sourceID,
		reqEndpoint: params.GetEndpoint(), reqID: params.GetId(),
	})
}

// addOwnerEpoch carries the source snapshot's owner epoch on a fit attach.
// The host and remote endpoint use it as a compare-and-swap fence, preventing
// a stale client from silently stealing a resize lease.
func (m *model) addOwnerEpoch(_ *pane, src *pb.Source, fit *bool, params *pb.MethodParams) {
	if params == nil || fit == nil || !*fit || src == nil || src.GetAttached() == false {
		return
	}
	if owner := strings.TrimSpace(src.GetResizeOwner()); owner != "" && owner != m.viewID {
		if epoch := src.GetOwnerEpoch(); epoch > 0 {
			params.ExpectedOwnerEpoch = &epoch
		}
	}
}

func fitRequested(params *pb.MethodParams) bool {
	return params == nil || params.Fit == nil || params.GetFit()
}

func (m *model) bindPane(paneID, sourceID string) {
	p := m.paneByID(paneID)
	if p == nil {
		return
	}
	if p.sourceID != "" && p.sourceID != sourceID {
		// Rebinding a pane to a different source releases any ownership it held.
		m.forgetOwner(p)
	}
	p.sourceID = sourceID
	p.detachedSourceID = ""
	p.pending = ""
	p.scroll = 0
	// A freshly bound pane owns its source until the user designates another
	// (the legacy single-pane default). This is why the demo and single-pane
	// cases keep working without an explicit take-owner.
	if sourceID != "" {
		if m.ownerPaneBySource == nil {
			m.ownerPaneBySource = map[string]string{}
		}
		if m.ownerPaneBySource[sourceID] == "" {
			m.ownerPaneBySource[sourceID] = p.id
		}
	}
	m.mode = modeLive
	m.markWorkbenchDirty()
	for _, f := range m.floatings {
		if f.pane == p {
			f.collapsed = false
			m.raiseFloating(f)
			m.activeFloat = f.id
			m.mode = modeLive
			return
		}
	}
}

func (m *model) quitCmd() app.Cmd {
	return m.emit("system.quit", &pb.MethodParams{}, opMsg{op: "quit"})
}

// ------------------------------------------------------------------- wire

// opMsg is the result of one terminal.*/system.quit method call.
type opMsg struct {
	op       string
	ref      string
	source   string
	ok       bool
	err      string
	endpoint string
	id       string
	delta    int
	// rows/offset carry terminal.scroll and history.window responses: the
	// window the host actually shows and its clamped offset.
	rows                 []string
	offset               int
	found, wrapped       bool
	matchStart, matchEnd int
	// exit closes the copy session after a successful copy (Enter).
	exit bool
	// seq is the copy search generation: a window response older than the
	// newest request is dropped.
	seq uint64
	// reqEndpoint/reqID remember the request's target so a failed attach can
	// be retried with fit=false (the follow mode) on an owner conflict.
	reqEndpoint  string
	reqID        string
	accessResult []byte
	promptField  int
	promptValue  string
	promptCursor int
	promptFocus  bool
}

func (m *model) emit(method string, params *pb.MethodParams, base opMsg) app.Cmd {
	client := m.client
	if client == nil {
		return nil
	}
	return func() app.Msg {
		responses := make(chan *pb.Response, 1)
		if _, err := client.Emit(method, params, func(resp *pb.Response) { responses <- resp }); err != nil {
			base.err = err.Error()
			return base
		}
		resp := <-responses
		base.ok = resp.GetOk()
		base.err = resp.GetError()
		if endpoint := resp.GetData().GetEndpoint(); endpoint != "" {
			base.endpoint = endpoint
		}
		base.id = resp.GetData().GetId()
		base.accessResult = append([]byte(nil), resp.GetData().GetAccessResult()...)
		base.rows = resp.GetData().GetRows()
		base.offset = int(resp.GetData().GetOffset())
		base.found, base.wrapped = resp.GetData().GetFound(), resp.GetData().GetWrapped()
		base.matchStart, base.matchEnd = int(resp.GetData().GetMatchStart()), int(resp.GetData().GetMatchEnd())
		return base
	}
}

func (m *model) onCreateDefaults(v opMsg) {
	endpoint := strings.TrimSpace(v.endpoint)
	defaults := createDefaults{loaded: true, err: v.err}
	if v.ok && len(v.accessResult) > 0 {
		var envelope apipb.ResultEnvelope
		if err := gproto.Unmarshal(v.accessResult, &envelope); err != nil {
			defaults.err = "invalid terminal defaults response"
		} else if apiErr := envelope.GetError(); apiErr != nil {
			defaults.err = apiErr.GetMessage()
		} else if result := envelope.GetTerminalDefaults(); result != nil && result.GetDefaults() != nil {
			defaults.command = append([]string(nil), result.GetDefaults().GetDefaultCommand()...)
			defaults.cwd = strings.TrimSpace(result.GetDefaults().GetDefaultCwd())
		} else {
			defaults.err = "terminal defaults unavailable"
		}
	}
	if endpoint == "" {
		return
	}
	if m.createDefaults == nil {
		m.createDefaults = map[string]createDefaults{}
	}
	m.createDefaults[endpoint] = defaults
	if m.overlay == overlayPrompt && m.promptKind == "terminal.create" && m.promptEndpoint == endpoint {
		m.applyCreateDefaults(endpoint, defaults)
	}
}

func (m *model) onPromptPath(v opMsg) {
	if m.overlay != overlayPrompt || m.promptKind != "terminal.create" ||
		m.promptField != v.promptField || m.promptField != 3 ||
		m.promptEndpoint != v.endpoint || len(m.promptFields) <= 3 || len(m.promptCursors) <= 3 {
		return
	}
	value := m.promptFields[3]
	cursor := clampInt(m.promptCursors[3], 0, len([]rune(value)))
	if value != v.promptValue || cursor != v.promptCursor {
		return
	}
	title := "path"
	var items []string
	empty := ""
	if v.ok && len(v.accessResult) > 0 {
		var envelope apipb.ResultEnvelope
		if err := gproto.Unmarshal(v.accessResult, &envelope); err != nil {
			empty = "(invalid path response)"
		} else if apiErr := envelope.GetError(); apiErr != nil {
			empty = "(" + apiErr.GetMessage() + ")"
		} else if result := envelope.GetPathListDirectories(); result != nil {
			if result.GetBasePath() != "" {
				title = "path: " + result.GetBasePath()
			}
			for _, entry := range result.GetEntries() {
				if entry.GetPath() != "" {
					items = append(items, entry.GetPath())
				}
			}
			if len(items) == 0 {
				empty = "(no matching directories)"
			}
		} else {
			empty = "(path completion unavailable)"
		}
	} else if v.err != "" {
		empty = "(" + v.err + ")"
	}
	m.setPromptSuggestions(3, title, items, empty, v.promptFocus)
}

func (m *model) onOp(v opMsg) app.Cmd {
	switch v.op {
	case "defaults":
		m.onCreateDefaults(v)
	case "path":
		m.onPromptPath(v)
	case "bind":
		if !v.ok {
			// Owner conflict on attach: retry as a follower (fit=false) so the
			// pane still mirrors the terminal, like the old take_owner path.
			if v.source != "" && isOwnerConflict(v.err) {
				m.toast = "following " + shortSourceID(v.source) + " (owner held)"
				follow := false
				return m.emit("terminal.attach", &pb.MethodParams{
					Endpoint: v.reqEndpoint, Id: v.reqID, Fit: &follow,
				}, opMsg{op: "bind", ref: v.ref, source: v.source})
			}
			m.toast = "bind failed: " + v.err
			if p := m.paneByID(v.ref); p != nil {
				p.pending = ""
			}
			return nil
		}
		sourceID := v.source
		if sourceID == "" {
			endpoint := v.endpoint
			if endpoint == "" {
				endpoint = "local"
			}
			sourceID = "terminal:" + endpoint + ":" + v.id
		}
		m.bindPane(v.ref, sourceID)
		if v.source != "" {
			m.toast = "bound " + shortSourceID(sourceID)
		} else {
			m.toast = "bound " + v.id
		}
	case "restart":
		if p := m.paneByID(v.ref); p != nil {
			p.pending = ""
		}
		if !v.ok {
			m.toast = "restart failed: " + v.err
		}
	case "rename":
		if !v.ok {
			m.toast = "rename failed: " + v.err
		} else {
			m.toast = "terminal renamed"
		}
	case "detach":
		if !v.ok {
			m.toast = "detach failed: " + v.err
		} else if p := m.paneByID(v.ref); p != nil {
			m.forgetOwner(p)
			p.detachedSourceID = p.sourceID
			p.sourceID = ""
			p.lines = nil
			m.toast = "detached; terminal kept alive"
		}
	case "reconnect":
		if !v.ok {
			m.toast = "reconnect failed: " + v.err
		} else if p := m.paneByID(v.ref); p != nil {
			if p.sourceID == "" {
				p.sourceID = p.detachedSourceID
			}
			p.detachedSourceID = ""
			m.toast = "reconnected"
			// A reconnected pane owns its source again until another pane is
			// designated (legacy first-bind default).
			if p.sourceID != "" && m.ownerPaneBySource[p.sourceID] == "" {
				m.ownerPaneBySource[p.sourceID] = p.id
			}
		}
	case "owner":
		if p := m.paneByID(v.ref); p != nil {
			p.pending = ""
		}
		if !v.ok {
			m.toast = "owner conflict: " + v.err
		}
	case "kill":
		if !v.ok {
			m.toast = "kill failed: " + v.err
		}
	case "remove":
		if !v.ok {
			m.toast = "remove failed: " + v.err
		}
	case "scroll":
		if !v.ok {
			m.toast = "scroll failed: " + v.err
			return nil
		}
		if p := m.paneByID(v.ref); p != nil {
			st := m.copyFor(p)
			if st == nil {
				return nil
			}
			// Drop a late response from an older direction: a persistent
			// provider can complete the previous request after the user has
			// already reversed the wheel, and applying it would put the
			// viewport back where it was (visible up/down oscillation).
			if v.seq != 0 && v.seq != st.scrollSeq {
				return nil
			}
			return m.applyCopyScroll(p, st, v.rows, v.offset, v.delta)
		}
	case "search":
		if p := m.paneByID(v.ref); p != nil {
			if st := m.copyFor(p); st != nil {
				m.applyTerminalSearch(st, v)
			}
		}
	case "resetCopy":
		if p := m.paneByID(v.ref); p != nil {
			if st := m.copyFor(p); st != nil && v.seq == st.searchSeq {
				if !v.ok {
					m.toast = "copy reset: " + v.err
					return nil
				}
				return m.fetchCopyWindow(p, st)
			}
		}
	case "window":
		if !v.ok {
			return nil
		}
		if p := m.paneByID(v.ref); p != nil {
			st := m.copyFor(p)
			if st == nil {
				return nil
			}
			return m.applyCopyWindow(p, st, v.rows, v.offset, v.seq)
		}
	case "copy":
		if !v.ok {
			m.toast = "copy failed: " + v.err
			return nil
		}
		if p := m.paneByID(v.ref); p != nil && m.copyFor(p) != nil {
			if v.exit {
				m.toast = "copied selection"
				return m.endCopy(p)
			}
			m.toast = "copied selection"
		}
	case "clipboard.list":
		if !v.ok {
			m.toast = "clipboard history: " + v.err
			return nil
		}
		m.clipboard = nil
		m.clipboardIDs = nil
		for _, row := range v.rows {
			var entry struct {
				ID   string `json:"id"`
				Text string `json:"text"`
			}
			if json.Unmarshal([]byte(row), &entry) == nil && entry.ID != "" {
				m.clipboardIDs = append(m.clipboardIDs, entry.ID)
				m.clipboard = append(m.clipboard, entry.Text)
			}
		}
	case "clipboard.latest":
		// Legacy copy-scene p: paste the newest history entry, if any.
		if !v.ok {
			m.toast = "paste: " + v.err
			return nil
		}
		var newest string
		for _, row := range v.rows {
			var entry struct {
				ID string `json:"id"`
			}
			if json.Unmarshal([]byte(row), &entry) == nil && entry.ID != "" {
				newest = entry.ID
				break
			}
		}
		if newest == "" {
			m.toast = "clipboard history is empty"
			return nil
		}
		p := m.paneByID(v.ref)
		if p == nil || p.sourceID == "" {
			return nil
		}
		params := m.clipboardPasteParams(p)
		params.ClipboardId = newest
		return m.emit("clipboard.paste", params, opMsg{op: "paste"})
	case "paste":
		if !v.ok {
			m.toast = "paste failed: " + v.err
		} else {
			m.toast = "pasted clipboard"
		}
	case "endpoint.list":
		// Legacy SYSTEM connections overlay: parse the registered endpoint
		// table into connectionRow entries. A failed list closes the overlay
		// and surfaces the error, since there is nothing to act on.
		if !v.ok {
			m.toast = "connections: " + v.err
			if m.overlay == overlayConnections {
				m.overlay = ""
			}
			return nil
		}
		m.storeConnections(v.rows)
		m.toast = fmt.Sprintf("connections: %d registered", len(m.connections))
	case "connections.list":
		if !v.ok {
			// A failed list has nothing to show: close the overlay and toast.
			m.toast = "connections: " + v.err
			if m.overlay == overlayConnections {
				m.overlay = ""
			}
			return nil
		}
		m.storeConnections(v.rows)
	case "connections.test":
		if !v.ok {
			m.toast = "test " + v.endpoint + ": " + v.err
		} else {
			m.toast = "test " + v.endpoint + ": ok"
		}
		// Re-list so the health column reflects the test result.
		return m.emit("endpoint.list", nil, opMsg{op: "connections.list"})
	case "connections.reconnect":
		if !v.ok {
			m.toast = "reconnect " + v.endpoint + ": " + v.err
		} else {
			m.toast = "reconnect " + v.endpoint + ": ok"
		}
		// Re-list so the health column reflects the fresh dial.
		return m.emit("endpoint.list", nil, opMsg{op: "connections.list"})
	case "workbench.get":
		return m.onWorkbenchGet(v)
	case "workbench.set":
		return m.onWorkbenchSet(v)
	case "quit":
		if !v.ok {
			m.toast = "quit rejected: " + v.err
			return nil
		}
		return app.Quit()
	}
	return nil
}

// isOwnerConflict reports whether an attach error is the resize-owner CAS
// conflict (the retry-as-follower trigger).
func isOwnerConflict(errText string) bool {
	return strings.Contains(strings.ToLower(errText), "owner")
}

func endpointOf(src *pb.Source) string {
	if src == nil {
		return "local"
	}
	if endpoint := strings.TrimSpace(src.GetEndpoint()); endpoint != "" {
		return endpoint
	}
	return "local"
}

func terminalIDOf(src *pb.Source, sourceID string) string {
	if src != nil && strings.TrimSpace(src.GetTerminalId()) != "" {
		return src.GetTerminalId()
	}
	parts := strings.SplitN(sourceID, ":", 3)
	if len(parts) == 3 {
		return parts[2]
	}
	return sourceID
}

func shortSourceID(sourceID string) string {
	parts := strings.SplitN(sourceID, ":", 3)
	if len(parts) == 3 {
		return parts[2]
	}
	return sourceID
}

func atoiNode(node, prefix string) int {
	value := strings.TrimPrefix(node, prefix)
	if value == node || value == "" {
		return -1
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return n
}

// ----------------------------------------------------------------- helpers

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func centerPad(text string, cells int) string {
	text = sdk.Truncate(text, cells)
	pad := cells - sdk.DisplayWidth(text)
	left := pad / 2
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", pad-left)
}
