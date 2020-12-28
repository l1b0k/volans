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
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/c9s/goprocinfo/linux"
	netns "github.com/containernetworking/plugins/pkg/ns"
	"github.com/gdamore/tcell/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rivo/tview"
	"github.com/vishvananda/netlink"
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
	dao *Dao

	app           *tview.Application
	infoView      *tview.TextView
	nsTableView   *tview.Table
	detailLayout  *tview.Flex
	nsDetailView  *tview.Table
	pidDetailView *tview.Table
	debugView     *tview.TextView

	layout   *tview.Flex
	rootView *tview.Pages
)

func init() {

}

func main() {
	dao = NewDao()
	CreateViews()
	SetKey()

	data := dao.GetNSWithPidCount()
	TableHelper(nsTableView, data)
	nsTableView.SetSelectionChangedFunc(func(row, column int) {
		//fmt.Fprint(debugView, fmt.Sprintf("select row %d co %d\n", row, column))
		log.Printf("select row %d co %d", row, column)
		UpdateDetailView(row)
	})
	if nsTableView.GetRowCount() > 1 {
		nsTableView.Select(1, 0)
	}

	if err := app.Run(); err != nil {
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

	nsTableView = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	nsDetailView = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	pidDetailView = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

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
			AddItem(nsTableView, 0, 1, true).
			AddItem(detailLayout.
				AddItem(nsDetailView, 0, 2, false).
				AddItem(pidDetailView, 0, 2, false).
				AddItem(debugView, 0, 1, false),
				0, 3, false), 0, 1, true).
		AddItem(infoView, 1, 1, false)

	rootView = tview.NewPages()
	rootView.AddPage("main", layout, true, true)

	app.SetRoot(rootView, true)
}

// UpdateDetailView
func UpdateDetailView(row int) {
	if row <= 0 {
		return
	}
	nsDetailView.Clear()
	// TODO
	bb := dao.GetNSDetail(nsTableView.GetCell(row, 0).Text)
	TableHelper(nsDetailView, bb)

	pidDetailView.Clear()
	bb = dao.GetProcessDetail(nsTableView.GetCell(row, 0).Text)
	TableHelper(pidDetailView, bb)
}

// SetKey set shortcuts
func SetKey() {
	// for global
	nsTableView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
			row, _ := nsTableView.GetSelection()
			UpdateDetailView(row)
		case tcell.KeyF12:
			app.Stop()
		}

		return event
	})
}

// TableHelper wrap to fill table data
func TableHelper(t *tview.Table, data [][]string) {
	if len(data) == 0 {
		return
	}
	for r := 0; r < len(data); r++ {
		for c := 0; c < len(data[0]); c++ {
			cell := tview.NewTableCell(data[r][c]).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter)
			if r == 0 {
				cell.SetTextColor(tcell.ColorYellow).SetSelectable(false)
			}
			t.SetCell(r, c, cell)
		}
	}
}

type Dao struct {
	DB *gorm.DB
}

func NewDao() *Dao {
	db := initDB()
	initProcData(db)
	return &Dao{DB: db}
}

func (d *Dao) GetPids(ns string) []int {
	var pids []int
	rows, err := d.DB.Raw("select pid from proc where namespace=?", ns).Rows()
	if err != nil {
		//TODO log
		return pids
	}
	defer rows.Close()
	for rows.Next() {
		var pid string
		err := rows.Scan(&pid)
		if err != nil {
			panic(err)
		}
		_, err = os.Stat(filepath.Join("/proc", pid))
		if err != nil {
			continue
		}
		p, _ := strconv.Atoi(pid)
		pids = append(pids, p)
	}
	sort.Ints(pids)
	return pids
}

func (d *Dao) GetNSWithPidCount() [][]string {
	data := [][]string{{"NS", "TYPE", "NPROCS"}}
	rows, err := d.DB.Raw("select namespace, ns_type, count(*) as count from proc group by namespace,ns_type").Rows()
	if err != nil {
		//TODO log
		return data
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

func (d *Dao) GetNSDetail(ns string) [][]string {
	data := [][]string{{"IF", "Type", "Addr", "rxErr", "rxDrop", "txErr", "txDrop", "MTU", "F"}}

	pids := d.GetPids(ns)

	netNS, err := netns.GetNS(fmt.Sprintf("/proc/%d/ns/net", pids[0]))
	if err != nil {
		return data
	}
	defer netNS.Close()
	var links []netlink.Link
	err = netNS.Do(func(ns netns.NetNS) error {
		var err error
		links, err = netlink.LinkList()
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return data
	}

	//linux.ReadNetStat()
	networkStats, err := linux.ReadNetworkStat(fmt.Sprintf("/proc/%d/net/dev", pids[0]))
	if err != nil {
		//TODO log
		return data
	}
	for _, stat := range networkStats {
		i := 0
		for ; i < len(links); i++ {
			if links[i].Attrs().Name == stat.Iface {
				break
			}
		}
		data = append(data, []string{
			stat.Iface,
			links[i].Type(),
			links[i].Attrs().HardwareAddr.String(),
			strconv.FormatUint(stat.RxErrs, 10),
			strconv.FormatUint(stat.RxDrop, 10),
			strconv.FormatUint(stat.TxErrs, 10),
			strconv.FormatUint(stat.TxDrop, 10),
			strconv.Itoa(links[i].Attrs().MTU),
			links[i].Attrs().Flags.String(),
		})

	}
	return data
}

func (d *Dao) GetProcessDetail(ns string) [][]string {
	data := [][]string{{"PID", "Name", "S", "Cpu", "Command"}}
	pids := d.GetPids(ns)
	for _, pid := range pids {
		p, err := linux.ReadProcess(uint64(pid), "/proc")
		if err != nil {
			continue
		}
		cmd := p.Cmdline
		if len(p.Cmdline) > 20 {
			cmd = p.Cmdline[:20]
		}

		data = append(data, []string{
			strconv.Itoa(pid),
			p.Status.Name,
			p.Stat.State,
			formatStr(p.Status.CpusAllowed),
			cmd,
		})
	}

	return data
}

// ugly...
func formatStr(b []uint32) string {
	var ss []string
	for _, i := range b {
		ss = append(ss, fmt.Sprintf("%b", i))
	}
	return strings.Join(ss, " ")
}
