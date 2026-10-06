package app

import (
	"strings"
	"sync"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Binding maps one key chord to a Msg.
//
// Key is a normalized key name from the host ("esc", "up", "enter", "p",
// "ctrl-p") or a bare base key paired with Mods ("p" + []string{"ctrl"}).
// Mods, when set, must match the modifiers encoded in the event's normalized
// key name exactly (order-insensitive). Context scopes the binding to the
// Keymap's current context; an empty Context binds globally.
type Binding struct {
	Key     string
	Mods    []string
	Msg     Msg
	Context string
}

// Keymap maps host key events to program messages. A nil *Keymap accepts
// every key (Bind reports false). Keymap is safe for concurrent use.
type Keymap struct {
	mu       sync.RWMutex
	bindings []Binding
	context  string
}

// NewKeymap returns a keymap with the given bindings.
func NewKeymap(bindings ...Binding) *Keymap {
	k := &Keymap{}
	k.Add(bindings...)
	return k
}

// Add registers bindings after the existing ones: within the same context
// pass the earliest registered binding wins, and contextual bindings win
// over global ones.
func (k *Keymap) Add(bindings ...Binding) {
	k.mu.Lock()
	k.bindings = append(k.bindings, bindings...)
	k.mu.Unlock()
}

// SetContext sets the context contextual bindings are matched against.
func (k *Keymap) SetContext(context string) {
	k.mu.Lock()
	k.context = context
	k.mu.Unlock()
}

// Context returns the current context.
func (k *Keymap) Context() string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.context
}

// Bind returns the message bound to ev, if any. Contextual bindings are
// consulted before global ones; the first match wins.
func (k *Keymap) Bind(ev *pb.KeyEvent) (Msg, bool) {
	if k == nil || ev == nil {
		return nil, false
	}
	base, mods := splitChord(ev.GetKey())
	k.mu.RLock()
	defer k.mu.RUnlock()
	for _, b := range k.bindings {
		if b.Context != "" && b.Context == k.context && matchBinding(b, base, mods) {
			return b.Msg, true
		}
	}
	for _, b := range k.bindings {
		if b.Context == "" && matchBinding(b, base, mods) {
			return b.Msg, true
		}
	}
	return nil, false
}

// ComponentHandler is implemented by focus entries that consume component
// events directly. Component returns an optional Msg for Update (nil
// swallows the event).
type ComponentHandler interface {
	Component(*pb.ComponentEvent) Msg
}

// Focus is a LIFO stack of focus entries used to route component events: the
// current entry, when it implements ComponentHandler, sees each event before
// Update does; otherwise the event reaches Update as a ComponentMsg. The
// loop touches Focus on its own goroutine and models may push/pop from
// Update, so Focus itself is not goroutine-safe.
type Focus struct {
	stack []any
}

// Push puts entry on top of the stack (nil entries are ignored).
func (f *Focus) Push(entry any) {
	if entry == nil {
		return
	}
	f.stack = append(f.stack, entry)
}

// Pop removes and returns the top entry, or nil when empty.
func (f *Focus) Pop() any {
	if len(f.stack) == 0 {
		return nil
	}
	top := f.stack[len(f.stack)-1]
	f.stack[len(f.stack)-1] = nil
	f.stack = f.stack[:len(f.stack)-1]
	return top
}

// Current returns the top entry without removing it, or nil when empty.
func (f *Focus) Current() any {
	if len(f.stack) == 0 {
		return nil
	}
	return f.stack[len(f.stack)-1]
}

// Len returns the stack depth.
func (f *Focus) Len() int { return len(f.stack) }

// Reset clears the stack; the loop calls it on every new HELLO epoch.
func (f *Focus) Reset() { f.stack = nil }

// matchBinding compares one binding against an already split event key.
func matchBinding(b Binding, base string, mods []string) bool {
	if b.Key == "" {
		return false
	}
	bBase, bMods := splitChord(b.Key)
	if len(b.Mods) > 0 {
		bMods = b.Mods
	}
	return bBase == base && sameMods(bMods, mods)
}

// splitChord splits a normalized key name into its base key and modifier
// prefixes ("ctrl-shift-v" -> "v", ["ctrl", "shift"]).
func splitChord(key string) (string, []string) {
	parts := strings.Split(key, "-")
	if len(parts) == 1 {
		return key, nil
	}
	return parts[len(parts)-1], parts[:len(parts)-1]
}

// sameMods reports set equality (order-insensitive).
func sameMods(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, x := range a {
		found := false
		for _, y := range b {
			if x == y {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
