// Copyright 2020 l1b0k
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	//"github.com/c9s/goprocinfo/linux"
	"github.com/gdamore/tcell/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rivo/tview"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var wantedNS = []string{"net"}

type ProcNS struct {
	Pid       string `gorm:"column:pid"`
	Namespace string `gorm:"column:namespace"`
	NSType    string `gorm:"column:ns_type"`
}

// TableName overrides the table name used by User to `profiles`
func (ProcNS) TableName() string {
	return "proc"
}

type Namespace struct {
	Type  string // the name of subsystem
	Inode string // inode number of this namespace
}

type Process struct {
	PID       string
	Namespace map[string]Namespace
}

var (
	infoView     *tview.TextView
	nsTableView  *tview.Table
	nsDetailView *tview.TextView
	rootView     *tview.Flex
)

func main() {
	dao := NewDao()
	CreateViews()

	data := dao.GetNSWithPidCount()
	cols := []string{"NS", "TYPE", "NPROCS"}
	for r := 0; r < len(data)+1; r++ {
		if r == 0 {
			nsTableView.SetCell(r, 0,
				tview.NewTableCell(cols[0]).
					SetTextColor(tcell.ColorYellow).
					SetAlign(tview.AlignCenter).SetSelectable(false))
			nsTableView.SetCell(r, 1,
				tview.NewTableCell(cols[1]).
					SetTextColor(tcell.ColorYellow).
					SetAlign(tview.AlignCenter).SetSelectable(false))
			nsTableView.SetCell(r, 2,
				tview.NewTableCell(cols[2]).
					SetTextColor(tcell.ColorYellow).
					SetAlign(tview.AlignCenter).SetSelectable(false))
			continue
		}

		nsTableView.SetCell(r, 0,
			tview.NewTableCell(data[r-1][0]).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))
		nsTableView.SetCell(r, 1,
			tview.NewTableCell(data[r-1][1]).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))
		nsTableView.SetCell(r, 2,
			tview.NewTableCell(data[r-1][2]).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))
	}
	nsTableView.SetSelectionChangedFunc(func(row, column int) {
		if row <= 0 || column < 0 {
			return
		}
		nsDetailView.Clear()
		// TODO
		bb := dao.GetNSDetail(nsTableView.GetCell(row, 0).Text)

		fmt.Fprintf(nsDetailView, "%s ", bb)
	})
	if nsTableView.GetRowCount() > 1 {
		nsTableView.Select(1, 0)
	}

	app := tview.NewApplication()
	if err := app.SetRoot(rootView, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func initDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&ProcNS{})
	if err != nil {
		panic("failed to migrate database")
	}
	return db
}

func initProcData(db *gorm.DB) {
	procs, err := GetAllProcess()
	if err != nil {
		panic(err)
	}
	flatted := ProcessToProcNS(procs)
	for _, ns := range flatted {
		db.Create(&ns)
	}
}

// GetAllProcess return pid list sorted
func GetAllProcess() ([]Process, error) {
	dirs, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var pids []int
	for i := 0; i < len(dirs); i++ {
		pid, err := strconv.Atoi(dirs[i].Name())
		if err != nil {
			// not pid
			continue
		}
		pids = append(pids, pid)
	}
	sort.Ints(pids)

	var processes []Process
	for i := 0; i < len(pids); i++ {
		// read all ns for the pid
		ns := make(map[string]Namespace, len(wantedNS))
		for _, nsType := range wantedNS {
			inode, err := GetNSByPid(strconv.Itoa(pids[i]), nsType)
			if err != nil {
				continue
			}
			ns[nsType] = Namespace{
				Type:  nsType,
				Inode: inode,
			}
		}

		p := Process{
			PID:       strconv.Itoa(pids[i]),
			Namespace: ns,
		}
		processes = append(processes, p)
	}

	return processes, nil
}

// GetNSByPid get namespace inode id by pid and namespace type
func GetNSByPid(pid string, nsType string) (string, error) {
	info, err := os.Readlink(filepath.Join("/proc", pid, "ns", nsType))
	if err != nil {
		return "", err
	}

	return info[len(nsType)+2 : len(info)-1], nil
}

// ProcessToProcNS flat the record
func ProcessToProcNS(procs []Process) []ProcNS {
	result := make([]ProcNS, 0, len(procs))
	for _, p := range procs {
		for _, nsType := range wantedNS {
			ns := p.Namespace[nsType]
			result = append(result, ProcNS{
				Pid:       p.PID,
				Namespace: ns.Inode,
				NSType:    nsType,
			})
		}
	}
	return result
}

func CreateViews() {
	infoView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(false)
	hints := []string{"help", "ns"}
	for i := 0; i < len(hints); i++ {
		fmt.Fprintf(infoView, `F%d ["%d"][darkcyan]%s[white][""]  `, i+1, i, hints[i])
	}
	nsTableView = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	nsDetailView = tview.NewTextView()

	rootView = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(nsTableView, 0, 1, true).
			AddItem(tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(nsDetailView, 0, 1, false).
				AddItem(tview.NewBox().SetBorder(true), 0, 1, false),
				0, 3, true), 0, 1, true).
		AddItem(infoView, 1, 1, false)
}

type Dao struct {
	DB *gorm.DB
}

func NewDao() *Dao {
	db := initDB()
	initProcData(db)
	return &Dao{DB: db}
}

func (d *Dao) GetNSWithPidCount() [][]string {
	var data [][]string
	rows, err := d.DB.Raw("select namespace, ns_type, count(*) as count from proc group by namespace,ns_type").Rows()
	if err != nil {
		//TODO log
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var ns string
		var nsType string
		var count int
		err := rows.Scan(&ns, &nsType, &count)
		if err != nil {
			panic(err)
		}

		data = append(data, []string{ns, nsType, strconv.Itoa(count)})
	}
	return data
}

func (d *Dao) GetNSDetail(ns string) []byte {
	rows, err := d.DB.Raw("select pid from proc where namespace=?", ns).Rows()
	if err != nil {
		//TODO log
		return nil
	}
	defer rows.Close()
	var pids []string
	for rows.Next() {
		var pid string
		err := rows.Scan(&pid)
		if err != nil {
			panic(err)
		}
		pids = append(pids, pid)
	}

	bb, err := ioutil.ReadFile(filepath.Join("/proc", pids[0], "net/dev"))
	if err != nil {
		//TODO log

		return nil
	}
	return bb
	//networkStats, err := linux.ReadNetworkStat(filepath.Join("/proc", pids[0], "net/dev"))
	//if err != nil {
	//	//TODO log
	//	return
	//}
	//networkStats[0].
}
