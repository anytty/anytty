package builder

import pb "github.com/anytty/anytty/proto/ui/protobuf"

// Builder is a chainable builder for one view-tree node. Every setter returns the
// same builder; Build materializes the protobuf node. Zero values keep the
// protocol defaults (PROTOCOL §9.1), so a builder only carries what the
// program actually declared.
type Builder struct {
	node pb.Box
}

// Box starts an empty box: no size, no content, no input, visible.
func Box() *Builder { return &Builder{} }

// Col starts a container that stacks its children vertically (the default
// flow).
func Col(children ...*Builder) *Builder { return Box().Flow("col").Child(children...) }

// Row starts a container that places its children left to right.
func Row(children ...*Builder) *Builder { return Box().Flow("row").Child(children...) }

// Stack starts a container that places every child on the same content rect.
func Stack(children ...*Builder) *Builder { return Box().Flow("stack").Child(children...) }

// Text starts a text box; lines split on "\n".
func Text(text string) *Builder { return Box().Content(text) }

// Terminal starts a content-source box bound to a terminal source id.
func Terminal(sourceID string) *Builder { return Box().Self(sourceID) }

// Divider starts a one-cell separator box suitable for pane drags.
func Divider(vertical bool) *Builder {
	b := Box().Input("mouse")
	if vertical {
		return b.Width(1)
	}
	return b.Height(1)
}

// ID sets the hit-test / event node id.
func (b *Builder) ID(id string) *Builder { b.node.Id = id; return b }

// Size sets the declared box geometry; width/height <= 0 mean "unset" and
// flex participates in the parent's leftover distribution.
func (b *Builder) Size(width, height, flex int) *Builder {
	return b.Width(width).Height(height).Flex(flex)
}

// Width sets the declared width (>0) or leaves it unset (<=0).
func (b *Builder) Width(width int) *Builder { b.size().Width = int32(width); return b }

// Height sets the declared height (>0) or leaves it unset (<=0).
func (b *Builder) Height(height int) *Builder { b.size().Height = int32(height); return b }

// Flex sets the leftover-distribution weight.
func (b *Builder) Flex(flex int) *Builder { b.size().Flex = int32(flex); return b }

// Pos takes the box out of the regular flow into an overlay subtree at
// (x, y) relative to the parent content rect.
func (b *Builder) Pos(x, y int) *Builder {
	b.node.Pos = &pb.Pos{X: int32(x), Y: int32(y)}
	return b
}

// Style sets an opaque content style: an explicit style string
// ("fg:#RRGGBB;bg:#RRGGBB;bold") or a host-internal token. Empty keeps the
// host default style.
func (b *Builder) Style(token string) *Builder {
	b.node.Style = token
	return b
}

// Focused marks the box as the keyboard focus target (at most one per view).
func (b *Builder) Focused(focused bool) *Builder { b.node.Focused = focused; return b }

// Input declares the input kinds the box accepts ("key", "paste", "mouse",
// "wheel").
func (b *Builder) Input(kinds ...string) *Builder {
	b.node.Input = append(b.node.Input, kinds...)
	return b
}

// Cursor declares the program cursor relative to the content rect.
func (b *Builder) Cursor(row, col int, shape string) *Builder {
	b.node.Cursor = &pb.Cursor{Row: int32(row), Col: int32(col), Shape: shape}
	return b
}

// Content sets single-string content.
func (b *Builder) Content(text string) *Builder {
	b.node.Content = &pb.Content{Text: text}
	return b
}

// Lines sets pre-split content lines (wins over Content text).
func (b *Builder) Lines(lines ...string) *Builder {
	b.node.Content = &pb.Content{Lines: lines}
	return b
}

// Self binds the box to a host content source (component reference).
func (b *Builder) Self(sourceID string) *Builder {
	b.node.Content = &pb.Content{Self: sourceID}
	return b
}

// Props merges program-declared component properties/styles (content.props)
// into the box content. The host passes them through untouched; the component
// interprets the keys it knows and ignores every other key. Calling Props on
// a box without content creates an empty content carrier.
func (b *Builder) Props(props map[string]string) *Builder {
	if b.node.Content == nil {
		b.node.Content = &pb.Content{}
	}
	if b.node.Content.Props == nil {
		b.node.Content.Props = make(map[string]string, len(props))
	}
	for key, value := range props {
		b.node.Content.Props[key] = value
	}
	return b
}

// Visible sets explicit visibility; false hides the box and takes it out of
// the layout entirely.
func (b *Builder) Visible(visible bool) *Builder {
	b.node.Visible = &visible
	return b
}

// Flow sets the child placement ("col", "row", "stack").
func (b *Builder) Flow(flow string) *Builder { b.node.Flow = flow; return b }

// Child appends children in declaration (z) order.
func (b *Builder) Child(children ...*Builder) *Builder {
	for _, child := range children {
		if child == nil {
			continue
		}
		b.node.Children = append(b.node.Children, child.Build())
	}
	return b
}

// Build returns the protocol node. Every call returns a fresh copy.
func (b *Builder) Build() *pb.Box {
	out := &pb.Box{
		Id:      b.node.Id,
		Flow:    b.node.Flow,
		Style:   b.node.Style,
		Focused: b.node.Focused,
		Input:   append([]string(nil), b.node.Input...),
	}
	if b.node.Size != nil {
		out.Size = &pb.Size{Width: b.node.Size.Width, Height: b.node.Size.Height, Flex: b.node.Size.Flex}
	}
	if b.node.Pos != nil {
		out.Pos = &pb.Pos{X: b.node.Pos.X, Y: b.node.Pos.Y}
	}
	if b.node.Cursor != nil {
		out.Cursor = &pb.Cursor{Row: b.node.Cursor.Row, Col: b.node.Cursor.Col, Shape: b.node.Cursor.Shape}
	}
	if b.node.Content != nil {
		out.Content = &pb.Content{
			Text:  b.node.Content.Text,
			Lines: append([]string(nil), b.node.Content.Lines...),
			Self:  b.node.Content.Self,
			Props: cloneProps(b.node.Content.Props),
		}
	}
	if b.node.Visible != nil {
		visible := *b.node.Visible
		out.Visible = &visible
	}
	if len(b.node.Children) > 0 {
		out.Children = append([]*pb.Box(nil), b.node.Children...)
	}
	return out
}

func (b *Builder) size() *pb.Size {
	if b.node.Size == nil {
		b.node.Size = &pb.Size{}
	}
	return b.node.Size
}

// cloneProps copies a props map so Build snapshots never alias the builder.
func cloneProps(props map[string]string) map[string]string {
	if len(props) == 0 {
		return nil
	}
	out := make(map[string]string, len(props))
	for key, value := range props {
		out[key] = value
	}
	return out
}
