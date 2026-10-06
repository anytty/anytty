package widgets

import pb "github.com/anytty/anytty/proto/ui/protobuf"

// rowText flattens a box and its direct children into one string, the way a
// single rendered row is composed of text runs.
func rowText(b *pb.Box) string {
	if b == nil {
		return ""
	}
	text := boxText(b)
	for _, child := range b.GetChildren() {
		text += rowText(child)
	}
	return text
}
