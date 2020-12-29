package views

import (
	"github.com/rivo/tview"
)

// NewNetNSView show basic net info for a single process
func NewNetNSView() *tview.Table {
	view := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	view.SetBorder(true).SetTitle("net")
	return view
}
