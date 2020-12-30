/*
 Copyright 2020  l1b0k

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package controller

import (
	"github.com/gdamore/tcell/v2"
	"github.com/l1b0k/volans/modle"
	"github.com/l1b0k/volans/views"

	"github.com/rivo/tview"
)

type NetNSController struct {
	*tview.Table

	Dao    *modle.Dao
	Fields []views.Field
}

func NewNetNSController() *NetNSController {
	return &NetNSController{
		Table: views.NewNetNSView(),
		Dao:   modle.GetDao(),
		Fields: []views.Field{
			{Text: "IF", Cell: views.CellAlignLeft},
			{Text: "Type", Cell: views.CellAlignRight},
			{Text: "MAC", Cell: views.CellAlignRight},
			{Text: "CH", Cell: views.CellAlignRight},
			{Text: "IP", Cell: views.CellAlignRight},
			{Text: "rxErr", Cell: views.CellAlignRight},
			{Text: "rxDrop", Cell: views.CellAlignRight},
			{Text: "txErr", Cell: views.CellAlignRight},
			{Text: "txDrop", Cell: views.CellAlignRight},
			{Text: "MTU", Cell: views.CellAlignRight},
			{Text: "Flag", Cell: views.CellAlignRight},
			{Text: "GSO", Cell: views.CellAlignRight},
			{Text: "TSO", Cell: views.CellAlignRight},
			{Text: "LRO", Cell: views.CellAlignRight},
			{Text: "GRO", Cell: views.CellAlignRight},
			{Text: "SG", Cell: views.CellAlignRight},
			{Text: "CSUM[rx/tx]", Cell: views.CellAlignRight},
		},
	}
}

func (n *NetNSController) Reload(v interface{}) {
	ns, ok := v.(string)
	if !ok {
		return
	}
	// clear table
	n.Clear()
	// fill head
	skipped := 0
	for c, f := range n.Fields {
		if f.Hide {
			skipped++
			continue
		}
		n.SetCell(0, c-skipped, views.CellTitle(f.Text))
	}
	// fill data
	data := n.Dao.GetNetNSDetail(ns)
	skipped = 0
	for r := 0; r < len(data); r++ {
		for c := 0; c < len(data[0]); c++ {
			f := n.Fields[c]
			if f.Hide {
				skipped++
				continue
			}
			n.SetCell(r+1, c-skipped, f.Cell(data[r][c], data[r][c]))
		}
	}
}

func (n *NetNSController) SetKeybinding(a *App) {
	n.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		a.setGlobalKeybinding(event)
		return event
	})
}

func (n *NetNSController) SetFocus() {
	n.SetSelectable(true, false)
}

func (n *NetNSController) UnFocus() {
	n.SetSelectable(false, false)
}

func (n *NetNSController) Info() {

}
