// SPDX-License-Identifier: Unlicense OR MIT

package externalDragDrop

import (
	"gioui.org/internal/ops"
	"gioui.org/io/event"
	"gioui.org/op"
)

// Event is generated when the clipboard content is requested.
type Event struct {
	Text string
}

// ReadOp requests the text of the clipboard, delivered to
// the current handler through an Event.
type ReadOp struct {
	Tag event.Tag
}

func (h ReadOp) Add(o *op.Ops) {
	data := ops.Write1(&o.Internal, ops.TypeExternalDragDropLen, h.Tag)
	data[0] = byte(ops.TypeExternalDragDrop)
}

func (Event) ImplementsEvent() {}
