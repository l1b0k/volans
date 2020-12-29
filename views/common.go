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

package views

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Field struct {
	Text string
	Hide bool // default false ,show all

	Cell func(text string, v interface{}) *tview.TableCell
}

func CellTitle(text string) *tview.TableCell {
	return tview.NewTableCell(text).
		SetTextColor(tcell.ColorYellow).
		SetAlign(tview.AlignCenter).
		SetSelectable(false)
}

func CellAlignLeft(text string, v interface{}) *tview.TableCell {
	return CellColor(tview.NewTableCell(text).
		SetTextColor(tcell.ColorWhite).
		SetAlign(tview.AlignLeft).SetReference(v))
}

func CellAlignRight(text string, v interface{}) *tview.TableCell {
	return CellColor(tview.NewTableCell(text).
		SetTextColor(tcell.ColorWhite).
		SetAlign(tview.AlignRight).SetReference(v))
}

func CellAlignCenter(text string, v interface{}) *tview.TableCell {
	return CellColor(tview.NewTableCell(text).
		SetTextColor(tcell.ColorWhite).
		SetAlign(tview.AlignCenter).SetReference(v))
}

func CellColor(t *tview.TableCell) *tview.TableCell {
	if strings.Contains(t.Text, "off") ||
		strings.Contains(t.Text, "down") {
		t.SetTextColor(tcell.ColorRed)
	}
	return t
}
