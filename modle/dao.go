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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/c9s/goprocinfo/linux"
	netns "github.com/containernetworking/plugins/pkg/ns"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/safchain/ethtool"
	"github.com/shirou/gopsutil/process"
	"github.com/vishvananda/netlink"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TODO
var supportedNS = []string{"net"}

type Proc struct {
	Pid       string `gorm:"primaryKey,column:pid"`
	Namespace string `gorm:"column:namespace"`
	NSType    string `gorm:"column:ns_type"`
}

// TableName overrides the table name
func (Proc) TableName() string {
	return "proc"
}

type Container struct {
	Pid          string `gorm:"primaryKey,column:pid"`
	PodNamespace string `gorm:"column:pod_namespace"`
	PodName      string `gorm:"column:pod_name"`
	Type         string `gorm:"column:type"` // io.kubernetes.docker.type podsandbox/container
}

// TableName overrides the table name
func (Container) TableName() string {
	return "container"
}

func initDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&Proc{}, &Container{})
	if err != nil {
		panic("failed to migrate database")
	}
	return db
}

type Dao struct {
	DB           *gorm.DB
	DockerClient *docker.Client
}

var dao *Dao
var once sync.Once

func GetDao() *Dao {
	once.Do(func() {
		db := initDB()

		client, err := docker.NewClientWithOpts(
			docker.WithVersion("v1.21"),
		)
		if err != nil {
			panic(err)
		}

		dao = &Dao{
			DB:           db,
			DockerClient: client,
		}
		dao.Run()
		dao.LoadProcData()
	})
	return dao
}

func (d *Dao) LoadProcData() {
	pids, err := process.Pids()
	if err != nil {
		return
	}
	var procs []Proc
	result := d.DB.Find(&procs)
	if result.Error == nil {
		for _, proc := range procs {
			ok := false
			for _, id := range pids {
				if strconv.Itoa(int(id)) == proc.Pid {
					ok = true
					break
				}
			}
			if !ok {
				// delete
				d.DB.Delete(&proc)
			}
		}
	}
	// create
	for _, id := range pids {
		ok := false
		for _, proc := range procs {
			if strconv.Itoa(int(id)) == proc.Pid {
				ok = true
				break
			}
		}
		if ok {
			continue
		}
		for _, nsType := range supportedNS {
			inode, err := GetNSByPid(id, nsType)
			if err != nil {
				continue
			}
			p := Proc{
				Pid:       strconv.Itoa(int(id)),
				Namespace: inode,
				NSType:    nsType,
			}

			d.DB.Create(&p)
		}
	}

}

func (d *Dao) Run() {
	if d.DockerClient == nil {
		return
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	containers, err := d.DockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return
	}

	for _, s := range containers {
		state, err := d.DockerClient.ContainerInspect(ctx, s.ID)
		if err != nil {
			continue
		}
		// only care sandbox
		if s.Labels["io.kubernetes.docker.type"] != "podsandbox" {
			continue
		}

		var c Container
		result := d.DB.Where("pod_namespace = ? and pod_name = ?", s.Labels["io.kubernetes.pod.namespace"], s.Labels["io.kubernetes.pod.name"]).First(&c)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				d.DB.Create(&Container{
					Pid:          strconv.Itoa(state.State.Pid),
					PodNamespace: s.Labels["io.kubernetes.pod.namespace"],
					PodName:      s.Labels["io.kubernetes.pod.name"],
					Type:         s.Labels["io.kubernetes.docker.type"],
				})
			}
		} else {
			// update pid only
			c.Pid = strconv.Itoa(state.State.Pid)
			d.DB.Save(c)
		}
	}
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
	var containers []Container
	d.DB.Model(&Container{}).Find(&containers)

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

		podInfo := map[string]interface{}{}
		d.DB.Raw("select b.pod_namespace as pod_namespace, b.pod_name as pod_name from"+
			" proc as a , container as b where a.pid = b.pid and a.namespace = ?", ns).First(&podInfo)
		data = append(data, []string{ns, nsType, strconv.Itoa(count), fmt.Sprintf("%s/%s", podInfo["pod_namespace"], podInfo["pod_name"])})
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

// GetNSByPid get namespace inode id by pid and namespace type
func GetNSByPid(pid int32, nsType string) (string, error) {
	info, err := os.Readlink(fmt.Sprintf("/proc/%d/ns/%s", pid, nsType))
	if err != nil {
		return "", err
	}

	return info[len(nsType)+2 : len(info)-1], nil
}
