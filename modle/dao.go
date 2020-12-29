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

package modle

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/c9s/goprocinfo/linux"
	netns "github.com/containernetworking/plugins/pkg/ns"
	"github.com/safchain/ethtool"
	"github.com/vishvananda/netlink"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TODO
var supportedNS = []string{"net"}

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

type Dao struct {
	DB *gorm.DB
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
		ns := make(map[string]Namespace, len(supportedNS))
		for _, nsType := range supportedNS {
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
		for _, nsType := range supportedNS {
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

var dao *Dao
var once sync.Once

func GetDao() *Dao {
	once.Do(func() {
		db := initDB()
		initProcData(db)
		dao = &Dao{DB: db}
	})
	return dao
}

func (d *Dao) GetPIDs(ns string) []int {
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
	var data [][]string
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

func (d *Dao) GetNetNSDetail(ns string) [][]string {
	var data [][]string
	pids := d.GetPIDs(ns)

	netNS, err := netns.GetNS(fmt.Sprintf("/proc/%d/ns/net", pids[0]))
	if err != nil {
		return data
	}
	defer netNS.Close()

	tool, err := ethtool.NewEthtool()
	if err != nil {
		return nil
	}
	defer tool.Close()

	networkStats, err := linux.ReadNetworkStat(fmt.Sprintf("/proc/%d/net/dev", pids[0]))
	if err != nil {
		//TODO log
		return data
	}

	var links []netlink.Link
	err = netNS.Do(func(ns netns.NetNS) error {
		var err error
		links, err = netlink.LinkList()
		if err != nil {
			return err
		}

		for _, stat := range networkStats {
			i := 0
			for ; i < len(links); i++ {
				if links[i].Attrs().Name == stat.Iface {
					break
				}
			}
			var s []string
			addrs, err := netlink.AddrList(links[i], netlink.FAMILY_ALL)
			if err == nil {
				for _, a := range addrs {
					if a.IP.IsLinkLocalUnicast() {
						continue
					}
					s = append(s, a.IP.String())
				}
			}
			channelStr := ""
			channel, err := tool.GetChannels(stat.Iface)
			if err == nil {
				channelStr = fmt.Sprintf("%d/%d", channel.CombinedCount, channel.MaxCombined)
			}

			feature, _ := tool.Features(stat.Iface)
			data = append(data, []string{
				stat.Iface,
				links[i].Type(),
				links[i].Attrs().HardwareAddr.String(),
				channelStr,
				strings.Join(s, ","),
				strconv.FormatUint(stat.RxErrs, 10),
				strconv.FormatUint(stat.RxDrop, 10),
				strconv.FormatUint(stat.TxErrs, 10),
				strconv.FormatUint(stat.TxDrop, 10),
				strconv.Itoa(links[i].Attrs().MTU),
				links[i].Attrs().Flags.String(),
				formatBool(feature["tx-generic-segmentation"]), // see https://kernel.googlesource.com/pub/scm/network/ethtool/ethtool/+/v3.4.2/ethtool.c#140
				formatBool(feature["tx-tcp-segmentation"]),
				formatBool(feature["rx-lro"]),
				formatBool(feature["rx-gro"]),
				formatBool(feature["tx-scatter-gather"]),
				fmt.Sprintf("%s/%s", formatBool(feature["rx-checksum"]), formatBool(feature["tx-checksum-ip-generic"])),
			})

		}

		return nil
	})
	if err != nil {
		return data
	}

	return data
}

func (d *Dao) GetProcDetail(ns string) [][]string {
	var data [][]string
	pids := d.GetPIDs(ns)
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

func formatBool(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
