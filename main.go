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

package main

import (
	"fmt"
	"log"

	"github.com/l1b0k/volans/controller"

	"github.com/gdamore/tcell/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rivo/tview"
)

var (
	app             *tview.Application
	infoView        *tview.TextView
	nsController    *controller.NSController
	detailLayout    *tview.Flex
	netNSController *controller.NetNSController
	procController  *controller.ProcController
	debugView       *tview.TextView

	layout   *tview.Flex
	rootView *tview.Pages
)

func main() {
	CreateViews()
	SetKey()

	if err := app.Run(); err != nil {
		panic(err)
	}
}

// CreateViews create all views
func CreateViews() {
	app = tview.NewApplication()
	infoView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(false)
	hints := [][]string{{"F1", "ns"}, {"F5", "refresh"}, {"F12", "quit"}}
	for i := 0; i < len(hints); i++ {
		fmt.Fprintf(infoView, `%s ["%d"][darkcyan]%s[white][""]  `, hints[i][0], i, hints[i][1])
	}

	netNSController = controller.NewNetNSController()
	procController = controller.NewProcController()

	nsController = controller.NewNSController().Reload(nil)
	nsController.SetSelectionChangedFunc(func(row, column int) {
		if row <= 0 {
			return
		}
		ReloadDetail(row, 0)
	})
	if nsController.GetRowCount() > 1 {
		nsController.Select(1, 0)
	}

	detailLayout = tview.NewFlex().
		SetDirection(tview.FlexRow)

	debugView = tview.NewTextView().
		SetWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	debugView.SetBorder(true).SetTitle(" debug").SetBorderAttributes(tcell.AttrBold)
	log.SetOutput(debugView)

	layout = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(nsController, 0, 1, true).
			AddItem(detailLayout.
				AddItem(netNSController, 0, 1, false).
				AddItem(procController, 0, 1, false).
				AddItem(debugView, 0, 1, false),
				0, 5, false), 0, 1, true).
		AddItem(infoView, 1, 1, false)

	rootView = tview.NewPages()
	rootView.AddPage("main", layout, true, true)

	app.SetRoot(rootView, true)
}

// SetKey set shortcuts
func SetKey() {
	// for global
	nsController.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			// TODO witch ns type
			//form := tview.NewForm()
			//for _, i := range wantedNS {
			//	form.AddCheckbox(i, true, func(checked bool) {
			//
			//	})
			//}
			//
			//modal := tview.NewFlex().SetDirection(tview.FlexColumn).
			//	AddItem(nil, 0, 1, false).
			//	AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			//		AddItem(nil, 0, 1, false).
			//		AddItem(form, 13, 1, true).
			//		AddItem(nil, 0, 1, false), 55, 1, true).
			//	AddItem(nil, 0, 1, false)
			//rootView.AddPage("modal", modal,true,true)
			//modal.set
		case tcell.KeyF2:
		case tcell.KeyF5:
			row, _ := nsController.GetSelection()
			ReloadDetail(row, 0)
		case tcell.KeyF12:
			app.Stop()

		case tcell.KeyTAB:
		}

		return event
	})
}

func ReloadDetail(row, col int) {
	netNSController.Reload(nsController.GetCell(row, 0).Text)
	procController.Reload(nsController.GetCell(row, 0).Text)
}
