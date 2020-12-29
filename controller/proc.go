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
	"github.com/l1b0k/volans/modle"
	"github.com/l1b0k/volans/views"

	"github.com/rivo/tview"
)

type ProcController struct {
	*tview.Table

	Dao    *modle.Dao
	Fields []views.Field
}

func NewProcController() *ProcController {
	return &ProcController{
		Table: views.NewProcView(),
		Dao:   modle.GetDao(),
		Fields: []views.Field{
			{Text: "PID", Cell: views.CellAlignLeft},
			{Text: "Name", Cell: views.CellAlignRight},
			{Text: "S", Cell: views.CellAlignRight},
			{Text: "CPU", Cell: views.CellAlignRight},
			{Text: "CMD", Cell: views.CellAlignLeft},
		},
	}
}

func (n *ProcController) Reload(v interface{}) *ProcController {
	ns, ok := v.(string)
	if !ok {
		return n
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
	data := n.Dao.GetProcDetail(ns)
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
	return n
}
