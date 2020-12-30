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
	"fmt"

	"github.com/rivo/tview"
)

type InfoController struct {
	*tview.TextView
}

func NewInfoController() *InfoController {
	infoView := &InfoController{
		TextView: tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetWrap(false),
	}
	hints := [][]string{{"Tab", "toggle"}, {"F5", "refresh"}, {"F12", "quit"}}
	for i := 0; i < len(hints); i++ {
		fmt.Fprintf(infoView, `%s ["%d"][darkcyan]%s[white][""]  `, hints[i][0], i, hints[i][1])
	}
	return infoView
}

func (n *InfoController) Reload(v interface{}) *InfoController {
	return n
}
