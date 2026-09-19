package sdk

import "github.com/anytty/anytty/clients/tui/sdk/builder"

// Builder is a chainable builder for one view-tree node (sdk/builder).
type Builder = builder.Builder

// Box starts an empty box.
func Box() *Builder { return builder.Box() }

// Col starts a container that stacks its children vertically.
func Col(children ...*Builder) *Builder { return builder.Col(children...) }

// Row starts a container that places its children left to right.
func Row(children ...*Builder) *Builder { return builder.Row(children...) }

// Stack starts a container that places every child on the same content rect.
func Stack(children ...*Builder) *Builder { return builder.Stack(children...) }

// Text starts a text box; lines split on "\n".
func Text(text string) *Builder { return builder.Text(text) }

// Terminal starts a content-source box bound to a terminal source id.
func Terminal(sourceID string) *Builder { return builder.Terminal(sourceID) }

// Divider starts a one-cell separator box.
func Divider(vertical bool) *Builder { return builder.Divider(vertical) }
