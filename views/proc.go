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
	"github.com/rivo/tview"
)

// NewProcView show proc info relate to this namespace
func NewProcView() *tview.Table {
	view := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false).
		SetFixed(1, 0)
	view.SetBorder(true).SetTitle("proc")
	return view
}
