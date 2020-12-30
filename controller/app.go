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
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var app *App
var once sync.Once

type App struct {
	Tables  []Interface
	Current int

	*tview.Application
	rootView *tview.Pages

	nsController    *NSController
	netNSController *NetNSController
	procController  *ProcController

	infoController *InfoController
}

// GetApp return instance
func GetApp() *App {
	once.Do(func() {
		app = &App{}
		app.createViews()
		app.setKeys()
	})
	return app
}

// createViews build view and layout
func (a *App) createViews() {
	a.Application = tview.NewApplication()

	a.infoController = NewInfoController()
	a.netNSController = NewNetNSController()
	a.procController = NewProcController()

	a.nsController = NewNSController()
	a.nsController.Reload(nil)
	a.nsController.SetSelectionChangedFunc(func(row, column int) {
		if row <= 0 {
			return
		}
		a.ReloadDetail(row, 0)
	})
	if a.nsController.GetRowCount() > 1 {
		a.nsController.Select(1, 0)
	}

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(a.nsController, 0, 1, true).
			AddItem(tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(a.netNSController, 0, 1, false).
				AddItem(a.procController, 0, 1, false),
				0, 5, false), 0, 1, true).
		AddItem(a.infoController, 1, 1, false)

	a.rootView = tview.NewPages()
	a.rootView.AddPage("main", layout, true, true)

	a.SetRoot(a.rootView, true)

	a.Tables = append(a.Tables, a.nsController, a.netNSController, a.procController)
}

func (a *App) setKeys() {
	a.nsController.SetKeybinding(a)
	a.netNSController.SetKeybinding(a)
	a.procController.SetKeybinding(a)
}

func (a *App) setGlobalKeybinding(event *tcell.EventKey) {
	switch event.Key() {
	case tcell.KeyTab:
		a.Next()
	case tcell.KeyBacktab:
	//a.Previous()
	case tcell.KeyF5:
	}
}

// Next
func (a *App) Next() {
	if len(a.Tables) == 0 {
		return
	}
	i := (a.Current + 1) % len(a.Tables)
	a.Tables[a.Current].UnFocus()
	a.Tables[i].SetFocus()
	a.Current = i
	a.SetFocus(a.Tables[i])
}

// Previous
func (a *App) Previous() {
	if len(a.Tables) == 0 {
		return
	}
	// TODO
}

func (a *App) ReloadDetail(row, col int) {
	a.netNSController.Reload(a.nsController.GetCell(row, 0).Text)
	a.procController.Reload(a.nsController.GetCell(row, 0).Text)
}
