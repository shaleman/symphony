/***
Copyright 2014 Cisco Systems Inc. All rights reserved.

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
package nodeCtrler

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/contiv/symphony/pkg/rsrcMgr"

	"github.com/contiv/objmodel/objdb"
	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/libfsm"

	log "github.com/Sirupsen/logrus"
)

// Per node state
type Node struct {
	HostName   string              // Host name
	HostAddr   string              // Host ip addr
	Port       int                 // Port where athena is running
	Resources  []altaspec.Resource // Schedulable resource
	Attributes map[string]string   // node attributes
	Fsm        *libfsm.Fsm         // FSM for the node
	eventChan  chan libfsm.Event   // Event channel
	ticker     *time.Ticker        // Periodic ticker for the node
}

// Create a new node
func NewNode(hostAddr string, port int) (*Node, error) {
	node := new(Node)

	log.Infof("Adding node: %s:%d", hostAddr, port)

	// Initialize the values
	node.HostAddr = hostAddr
	node.Port = port

	// FSM for the node
	node.Fsm = libfsm.NewFsm(&libfsm.FsmTable{
		// currentState,  event,      newState,   callback
		{"created", "up", "alive", func(e libfsm.Event) error { return node.nodeUpEvent() }},
		{"created", "ticker", "created", func(e libfsm.Event) error { return nil }},
		{"alive", "up", "alive", func(e libfsm.Event) error { return node.nodeUpEvent() }},
		{"alive", "ticker", "alive", func(e libfsm.Event) error { return node.nodeAliveTicker() }},
		{"alive", "timeout", "unreachable", func(e libfsm.Event) error { return nil }},
		{"alive", "down", "down", func(e libfsm.Event) error { return node.nodeDownEvent() }},
		{"unreachable", "ticker", "unreachable", func(e libfsm.Event) error { return nil }},
		{"unreachable", "up", "alive", func(e libfsm.Event) error { return nil }},
		{"down", "up", "reachable", func(e libfsm.Event) error { return node.nodeReachableEvent() }},
		{"reachable", "register", "alive", func(e libfsm.Event) error { return node.nodeUpEvent() }},
		{"down", "ticker", "down", func(e libfsm.Event) error { return node.nodeAliveTicker() }},
	}, "created")

	// create the channel
	node.eventChan = make(chan libfsm.Event, 200)

	// Create a timer for periodic poll
	node.ticker = time.NewTicker(time.Second * 5)

	// Kick off the node runloop
	go node.nodeRunLoop()

	// post Get info event
	node.eventChan <- libfsm.Event{"up", nil}

	return node, nil
}

// Main run loop for the node
// FIXME: after each event we need to persist the object
func (self *Node) nodeRunLoop() {
	for {
		select {
		case event := <-self.eventChan:
			self.Fsm.FsmEvent(event)
		case <-self.ticker.C:
			// Use this ticker to perform keepalive and retries
			self.Fsm.FsmEvent(libfsm.Event{"ticker", nil})
		}
	}
}

// Queue an event to the node
func (self *Node) NodeEvent(eventName string) {
	self.eventChan <- libfsm.Event{eventName, nil}
}

// Handle node up event
func (self *Node) nodeUpEvent() error {
	log.Infof("Getting node info")

	// FIXME: HACK alert
	// We currently need new node to establish VTEP tunnels to all peer nodes
	// Before it can register with ofnet master and get all routes. Ideally,
	// Ofnet should handle routes and VTEPs coming out of order.
	// Please fix this in ofnet

	// Inform all node about the new node and new node about all existing nodes
	err := nodeUpBcast(self.HostAddr)
	if err != nil {
		log.Errorf("Error sending nod up broadcast. Err: %v", err)
	}

	// Get my address
	localIpAddr, err := nodeCtrl.cdb.GetLocalAddr()
	if err != nil {
		log.Fatalf("Could not find a local address. Err %v", err)
		return err
	}

	// master info
	masterInfo := objdb.ServiceInfo{
		ServiceName: "zeus",
		HostAddr:    localIpAddr,
		Port:        8000, // FIXME: dont hardcode the port
	}

	var nodeSpec altaspec.NodeSpec
	err = self.NodePostReq("/node/register", &masterInfo, &nodeSpec)
	if err != nil {
		log.Errorf("Error getting node info. Err: %v", err)
		return err
	}

	// Save the node info
	self.HostName = nodeSpec.HostName
	self.Resources = nodeSpec.Resources
	self.Attributes = nodeSpec.Attributes

	log.Infof("Got node info: %+v\n Node: %+v", nodeSpec, self)

	// Add Node resources
	// FIXME: need better rsrc mgmt for cpu. Need to manage num cores
	//        and CPU oversubscription level etc
	var rsrcProvider []rsrcMgr.ResourceProvide
	for _, rsrc := range nodeSpec.Resources {
		provider := rsrcMgr.ResourceProvide{
			Type:     rsrc.Type,
			Provider: self.HostAddr,
			UnitType: rsrc.UnitType,
			NumRsrc:  rsrc.NumRsrc,
		}
		rsrcProvider = append(rsrcProvider, provider)
	}

	// Add the resource provider
	err = rsrcMgr.AddResourceProvider(rsrcProvider)
	if err != nil {
		log.Errorf("Error adding provider %+v. Err: %v", rsrcProvider, err)
		return err
	}

	return nil
}

func (self *Node) nodeAliveTicker() error {
	log.Debugf("Current state of the node %s is %s", self.HostAddr, self.Fsm.FsmState)

	// Ticker is useful only in alive state
	if self.Fsm.FsmState != "alive" {
		return nil
	}

	// Get list of altas running on this node
	var altaList []altaspec.AltaContext
	err := self.NodeGetReq("/alta", &altaList)
	if err != nil {
		log.Errorf("Error getting alta list from node %s. Err: %v", self.HostAddr, err)
		return err
	}

	log.Debugf("Got alta list from node %s: %+v", self.HostAddr, altaList)

	// See what to do about the containers
	err = nodeCtrl.ctrlers.AltaCtrler.ReconcileNode(self.HostAddr, altaList)
	if err != nil {
		log.Errorf("Error updating altaList %+v for node %s. Err: %v", altaList,
			self.HostAddr, err)
	}

	return nil
}

// Handle a node down event
func (self *Node) nodeDownEvent() error {
	// See what to do about the containers
	err := nodeCtrl.ctrlers.AltaCtrler.NodeDownEvent(self.HostAddr)
	if err != nil {
		log.Errorf("Error handling down event for node %s. Err: %v", self.HostAddr, err)
	}

	// Remove Node resources
	var rsrcProvider []rsrcMgr.ResourceProvide
	for _, rsrc := range self.Resources {
		provider := rsrcMgr.ResourceProvide{
			Type:     rsrc.Type,
			Provider: self.HostAddr,
			UnitType: rsrc.UnitType,
			NumRsrc:  rsrc.NumRsrc,
		}
		rsrcProvider = append(rsrcProvider, provider)
	}

	// Remove the resource provider
	err = rsrcMgr.RemoveResourceProvider(rsrcProvider)
	if err != nil {
		log.Errorf("Error removing provider %+v. Err: %v", rsrcProvider, err)
		return err
	}

	return nil
}

// Handle Node coming back up.
// we go from down->reachable->alive transition so that REST handlers can know
// node is reachable and its safe to perform http operations
func (self *Node) nodeReachableEvent() error {
	// Simply queue the register event
	self.NodeEvent("register")

	return nil
}

// Get JSON output from a http request
func (self *Node) NodeGetReq(path string, data interface{}) error {
	url := "http://" + self.HostAddr + ":" + strconv.Itoa(self.Port) + path

	// Make sure node is up and running
	if (self.Fsm.FsmState != "alive") && (self.Fsm.FsmState != "created") &&
		(self.Fsm.FsmState != "reachable") {
		log.Errorf("Node %s is down. cant make REST call %s", self.HostAddr, path)
		return errors.New("Node unreachable")
	}

	log.Debugf("Making REST request to url: %s", url)

	// perform Get request
	res, err := http.Get(url)
	if err != nil {
		log.Errorf("Error during http get. Err: %v", err)
		return err
	}

	// Check response code
	if res.StatusCode != http.StatusOK {
		log.Errorf("HTTP error response. Status: %s, StatusCode: %d", res.Status, res.StatusCode)
		return errors.New("HTTP Error response")
	}

	// Read the entire resp
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("Error during ioutil readall. Err: %v", err)
		return err
	}

	// Json to struct
	err = json.Unmarshal(body, data)
	if err != nil {
		log.Errorf("Error during json unmarshall. Err: %v", err)
		return err
	}

	log.Debugf("Results for (%s): %+v\n", url, data)

	return nil
}

// perform http POST request and return the response
func (self *Node) NodePostReq(path string, req interface{}, resp interface{}) error {
	url := "http://" + self.HostAddr + ":" + strconv.Itoa(self.Port) + path

	// Make sure node is up and running
	if (self.Fsm.FsmState != "alive") && (self.Fsm.FsmState != "created") &&
		(self.Fsm.FsmState != "reachable") {
		log.Errorf("Node %s is down. cant make REST call %s", self.HostAddr, path)
		return errors.New("Node unreachable")
	}

	log.Infof("Making REST request to url: %s", url)

	// Convert the req to json
	jsonStr, err := json.Marshal(req)
	if err != nil {
		log.Errorf("Error converting request data(%#v) to Json. Err: %v", req, err)
		return err
	}

	// Perform HTTP POST operation
	res, err := http.Post(url, "application/json", strings.NewReader(string(jsonStr)))
	if err != nil {
		log.Errorf("Error during http get. Err: %v", err)
		return err
	}

	// Check the response code
	if res.StatusCode != http.StatusOK {
		log.Errorf("HTTP error response. Status: %s, StatusCode: %d", res.Status, res.StatusCode)
		return errors.New("HTTP Error response")
	}

	// Read the entire response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("Error during ioutil readall. Err: %v", err)
		return err
	}

	// Convert response json to struct
	err = json.Unmarshal(body, resp)
	if err != nil {
		log.Errorf("Error during json unmarshall. Err: %v", err)
		return err
	}

	log.Infof("Results for (%s): %+v\n", url, resp)

	return nil
}

// FIXME: deprecated. not needed anymore
// Push network info to node
func (self *Node) PushNetwork(netSpec altaspec.AltaNetSpec) error {
	var resp altaspec.ReqSuccess
	url := "/network/create"
	err := self.NodePostReq(url, netSpec, &resp)
	if err != nil {
		log.Errorf("Error sending network %s to node %s. Err: %v",
			netSpec.NetworkName, self.HostAddr, err)
		return err
	}

	return nil
}
