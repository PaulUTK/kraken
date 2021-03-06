/* pxe.go: provides generic PXE/iPXE-boot capabilities
 *           this manages both DHCP and TFTP/HTTP services.
 *			 If <file> doesn't exist, but <file>.tpl does, tftp will fill it as as template.
 *
 * Author: J. Lowell Wofford <lowell@lanl.gov>
 *
 * This software is open source software available under the BSD-3 license.
 * Copyright (c) 2018, Triad National Security, LLC
 * See LICENSE file for details.
 */

//go:generate protoc -I ../../core/proto/include -I proto --go_out=plugins=grpc:proto proto/pxe.proto

package pxe

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/mdlayher/raw"

	"github.com/golang/protobuf/ptypes"

	"github.com/golang/protobuf/proto"
	"github.com/hpc/kraken/core"
	cpb "github.com/hpc/kraken/core/proto"
	"github.com/hpc/kraken/extensions/IPv4"
	pxepb "github.com/hpc/kraken/extensions/PXE/proto"
	"github.com/hpc/kraken/lib"
	pb "github.com/hpc/kraken/modules/pxe/proto"
)

const (
	PXEStateURL = "type.googleapis.com/proto.PXE/State"
	SrvStateURL = "/Services/pxe/State"
)

type pxmut struct {
	f       pxepb.PXE_State
	t       pxepb.PXE_State
	reqs    map[string]reflect.Value
	timeout string
}

var muts = map[string]pxmut{
	"NONEtoWAIT": {
		f:       pxepb.PXE_NONE,
		t:       pxepb.PXE_WAIT,
		reqs:    reqs,
		timeout: "10s",
	},
	"INITtoCOMP": {
		f: pxepb.PXE_INIT,
		t: pxepb.PXE_COMP,
		reqs: map[string]reflect.Value{
			"/PhysState": reflect.ValueOf(cpb.Node_POWER_ON),
			"/RunState":  reflect.ValueOf(cpb.Node_SYNC),
		},
		timeout: "180s",
	},
}

// modify these if you want different requires for mutations
var reqs = map[string]reflect.Value{
	"/PhysState": reflect.ValueOf(cpb.Node_POWER_ON),
}

// modify this if you want excludes
var excs = map[string]reflect.Value{}

/* we use channels and a node manager rather than locking
   to make our node store safe.  This is a simpple query
   language for that service */

type nodeQueryBy string

const (
	queryByIP  nodeQueryBy = "IP"
	queryByMAC nodeQueryBy = "MAC"
)

//////////////////
// PXE Object /
////////////////

// PXE provides PXE-boot capabilities
type PXE struct {
	api   lib.APIClient
	cfg   *pb.PXEConfig
	mchan <-chan lib.Event
	dchan chan<- lib.Event

	selfIP  net.IP
	selfNet net.IP

	options   layers.DHCPOptions
	leaseTime time.Duration

	iface     *net.Interface
	rawHandle *raw.Conn

	// for maintaining our list of currently booting nodes

	mutex  sync.RWMutex
	nodeBy map[nodeQueryBy]map[string]lib.Node
}

/*
 * concurrency safe accessors for nodeBy
 */

// NodeGet gets a node that we know about -- concurrency safe
func (px *PXE) NodeGet(qb nodeQueryBy, q string) (n lib.Node) { // returns nil for not found
	var ok bool
	px.mutex.RLock()
	if n, ok = px.nodeBy[qb][q]; !ok {
		px.api.Logf(lib.LLERROR, "tried to acquire node that doesn't exist: %s %s", qb, q)
		px.mutex.RUnlock()
		return
	}
	px.mutex.RUnlock()
	return
}

// NodeDelete deletes a node that we know about -- cuncurrency safe
func (px *PXE) NodeDelete(qb nodeQueryBy, q string) { // silently ignores non-existent nodes
	var n lib.Node
	var ok bool
	px.mutex.Lock()
	if n, ok = px.nodeBy[qb][q]; !ok {
		px.mutex.Unlock()
		return
	}
	v := n.GetValues([]string{px.cfg.IpUrl, px.cfg.MacUrl})
	ip := IPv4.BytesToIP(v[px.cfg.IpUrl].Bytes())
	mac := IPv4.BytesToMAC(v[px.cfg.MacUrl].Bytes())
	delete(px.nodeBy[queryByIP], ip.String())
	delete(px.nodeBy[queryByMAC], mac.String())
	px.mutex.Unlock()
}

// NodeCreate creates a new node in our node pool -- concurrency safe
func (px *PXE) NodeCreate(n lib.Node) (e error) {
	v := n.GetValues([]string{px.cfg.IpUrl, px.cfg.MacUrl})
	if len(v) != 2 {
		return fmt.Errorf("missing ip or mac for node, aborting")
	}
	ip := IPv4.BytesToIP(v[px.cfg.IpUrl].Bytes())
	mac := IPv4.BytesToMAC(v[px.cfg.MacUrl].Bytes())
	if ip == nil || mac == nil { // incomplete node
		return fmt.Errorf("won't add incomplete node: ip: %v, mac: %v", ip, mac)
	}
	px.mutex.Lock()
	px.nodeBy[queryByIP][ip.String()] = n
	px.nodeBy[queryByMAC][mac.String()] = n
	px.mutex.Unlock()
	return
}

/*
 * lib.Module
 */

var _ lib.Module = (*PXE)(nil)

// Name returns the FQDN of the module
func (*PXE) Name() string { return "github.com/hpc/kraken/modules/pxe" }

/*
 * lib.ModuleWithConfig
 */

var _ lib.Module = (*PXE)(nil)

// NewConfig returns a fully initialized default config
func (*PXE) NewConfig() proto.Message {
	r := &pb.PXEConfig{
		SrvIfaceUrl: "type.googleapis.com/proto.IPv4OverEthernet/Ifaces/0/Eth/Iface",
		SrvIpUrl:    "type.googleapis.com/proto.IPv4OverEthernet/Ifaces/0/Ip/Ip",
		IpUrl:       "type.googleapis.com/proto.IPv4OverEthernet/Ifaces/0/Ip/Ip",
		NmUrl:       "type.googleapis.com/proto.IPv4OverEthernet/Ifaces/0/Ip/Subnet",
		SubnetUrl:   "type.googleapis.com/proto.IPv4OverEthernet/Ifaces/0/Ip/Subnet",
		MacUrl:      "type.googleapis.com/proto.IPv4OverEthernet/Ifaces/0/Eth/Mac",
		TftpDir:     "tftp",
	}
	return r
}

// UpdateConfig updates the running config
func (px *PXE) UpdateConfig(cfg proto.Message) (e error) {
	if pxcfg, ok := cfg.(*pb.PXEConfig); ok {
		px.cfg = pxcfg
		return
	}
	return fmt.Errorf("invalid config type")
}

// ConfigURL gives the any resolver URL for the config
func (*PXE) ConfigURL() string {
	cfg := &pb.PXEConfig{}
	any, _ := ptypes.MarshalAny(cfg)
	return any.GetTypeUrl()
}

/*
 * lib.ModuleWithMutations & lib.ModuleWithDiscovery
 */
var _ lib.ModuleWithMutations = (*PXE)(nil)
var _ lib.ModuleWithDiscovery = (*PXE)(nil)

// SetMutationChan sets the current mutation channel
// this is generally done by the API
func (px *PXE) SetMutationChan(c <-chan lib.Event) { px.mchan = c }

// SetDiscoveryChan sets the current discovery channel
// this is generally done by the API
func (px *PXE) SetDiscoveryChan(c chan<- lib.Event) { px.dchan = c }

/*
 * lib.ModuleSelfService
 */
var _ lib.ModuleSelfService = (*PXE)(nil)

// Entry is the module's executable entrypoint
func (px *PXE) Entry() {
	nself, _ := px.api.QueryRead(px.api.Self().String())
	v, _ := nself.GetValue(px.cfg.SrvIpUrl)
	px.selfIP = IPv4.BytesToIP(v.Bytes())
	v, _ = nself.GetValue(px.cfg.SubnetUrl)
	px.selfNet = IPv4.BytesToIP(v.Bytes())
	v, _ = nself.GetValue(px.cfg.SrvIfaceUrl)
	go px.StartDHCP(v.String(), px.selfIP)
	go px.StartTFTP(px.selfIP)
	url := lib.NodeURLJoin(px.api.Self().String(), SrvStateURL)
	ev := core.NewEvent(
		lib.Event_DISCOVERY,
		url,
		&core.DiscoveryEvent{
			Module:  px.Name(),
			URL:     url,
			ValueID: "RUN",
		},
	)
	px.dchan <- ev
	for {
		select {
		case v := <-px.mchan:
			if v.Type() != lib.Event_STATE_MUTATION {
				px.api.Log(lib.LLERROR, "got unexpected non-mutation event")
				break
			}
			m := v.Data().(*core.MutationEvent)
			go px.handleMutation(m)
			break
		}
	}
}

// Init is used to intialize an executable module prior to entrypoint
func (px *PXE) Init(api lib.APIClient) {
	px.api = api
	px.mutex = sync.RWMutex{}
	px.nodeBy = make(map[nodeQueryBy]map[string]lib.Node)
	px.nodeBy[queryByIP] = make(map[string]lib.Node)
	px.nodeBy[queryByMAC] = make(map[string]lib.Node)
	px.cfg = px.NewConfig().(*pb.PXEConfig)
}

// Stop should perform a graceful exit
func (px *PXE) Stop() {
	os.Exit(0)
}

////////////////////////
// Unexported methods /
//////////////////////

func (px *PXE) handleMutation(m *core.MutationEvent) {
	switch m.Type {
	case core.MutationEvent_MUTATE:
		switch m.Mutation[1] {
		case "NONEtoWAIT": // starting a new mutation, register the node
			if e := px.NodeCreate(m.NodeCfg); e != nil {
				px.api.Logf(lib.LLERROR, "%v", e)
				break
			}
			url := lib.NodeURLJoin(m.NodeCfg.ID().String(), PXEStateURL)
			ev := core.NewEvent(
				lib.Event_DISCOVERY,
				url,
				&core.DiscoveryEvent{
					Module:  px.Name(),
					URL:     url,
					ValueID: "WAIT",
				},
			)
			px.dchan <- ev
		case "WAITtoINIT": // we're initializing, but don't do anything (more for discovery/timeout)
		case "INITtoCOMP": // done mutating a node, deregister
			v, _ := m.NodeCfg.GetValue(px.cfg.IpUrl)
			ip := IPv4.BytesToIP(v.Bytes())
			px.NodeDelete(queryByIP, ip.String())
			url := lib.NodeURLJoin(m.NodeCfg.ID().String(), PXEStateURL)
			ev := core.NewEvent(
				lib.Event_DISCOVERY,
				url,
				&core.DiscoveryEvent{
					Module:  px.Name(),
					URL:     url,
					ValueID: "COMP",
				},
			)
			px.dchan <- ev
		}
	case core.MutationEvent_INTERRUPT: // on any interrupt, we remove the node
		v, e := m.NodeCfg.GetValue(px.cfg.IpUrl)
		if e != nil || !v.IsValid() {
			break
		}
		ip := IPv4.BytesToIP(v.Bytes())
		px.NodeDelete(queryByIP, ip.String())
	}
}

func init() {
	module := &PXE{}
	mutations := make(map[string]lib.StateMutation)
	discovers := make(map[string]map[string]reflect.Value)
	dpxe := make(map[string]reflect.Value)

	for m := range muts {
		dur, _ := time.ParseDuration(muts[m].timeout)
		mutations[m] = core.NewStateMutation(
			map[string][2]reflect.Value{
				PXEStateURL: {
					reflect.ValueOf(muts[m].f),
					reflect.ValueOf(muts[m].t),
				},
			},
			reqs,
			excs,
			lib.StateMutationContext_CHILD,
			dur,
			[3]string{module.Name(), "/PhysState", "PHYS_HANG"},
		)
		dpxe[pxepb.PXE_State_name[int32(muts[m].t)]] = reflect.ValueOf(muts[m].t)
	}

	mutations["WAITtoINIT"] = core.NewStateMutation(
		map[string][2]reflect.Value{
			PXEStateURL: {
				reflect.ValueOf(pxepb.PXE_WAIT),
				reflect.ValueOf(pxepb.PXE_INIT),
			},
			"/RunState": {
				reflect.ValueOf(cpb.Node_UNKNOWN),
				reflect.ValueOf(cpb.Node_INIT),
			},
		},
		reqs,
		excs,
		lib.StateMutationContext_CHILD,
		time.Second*30,
		[3]string{module.Name(), "/PhysState", "PHYS_HANG"},
	)
	dpxe["INIT"] = reflect.ValueOf(pxepb.PXE_INIT)

	discovers[PXEStateURL] = dpxe
	discovers["/RunState"] = map[string]reflect.Value{
		"NODE_INIT": reflect.ValueOf(cpb.Node_INIT),
	}
	discovers["/PhysState"] = map[string]reflect.Value{
		"PHYS_HANG": reflect.ValueOf(cpb.Node_PHYS_HANG),
	}
	discovers[SrvStateURL] = map[string]reflect.Value{
		"RUN": reflect.ValueOf(cpb.ServiceInstance_RUN)}
	si := core.NewServiceInstance("pxe", module.Name(), module.Entry, nil)

	// Register it all
	core.Registry.RegisterModule(module)
	core.Registry.RegisterServiceInstance(module, map[string]lib.ServiceInstance{si.ID(): si})
	core.Registry.RegisterDiscoverable(module, discovers)
	core.Registry.RegisterMutations(module, mutations)
}
