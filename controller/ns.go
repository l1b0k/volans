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

type NSController struct {
	*tview.Table
	Dao *modle.Dao

	Fields []views.Field
}

func NewNSController() *NSController {
	return &NSController{
		Table: views.NewNSView(),
		Dao:   modle.GetDao(),
		Fields: []views.Field{
			{Text: "NS", Cell: views.CellAlignLeft},
			{Text: "TYPE", Cell: views.CellAlignRight},
			{Text: "NPROCS", Cell: views.CellAlignRight},
			{Text: "POD", Cell: views.CellAlignRight},
		},
	}
}

func (n *NSController) Reload(v interface{}) {
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
	data := n.Dao.GetNSWithPidCount()
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
	if n.GetRowCount() > 1 {
		n.Select(1, 0)
	}
}

func (n *NSController) SetKeybinding(a *App) {
	n.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		a.setGlobalKeybinding(event)
		return event
	})
}

func (n *NSController) SetFocus() {
	n.SetSelectable(true, false)
}

func (n *NSController) UnFocus() {
	n.SetSelectable(false, false)
}

func (n *NSController) Info() {

}
