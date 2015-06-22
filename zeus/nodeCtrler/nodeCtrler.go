package nodeCtrler

import (
	"errors"

	"github.com/contiv/symphony/pkg/altaspec"
	// "pkg/confStore"
	"github.com/contiv/ofnet/rpcHub"
	"github.com/contiv/symphony/pkg/confStore/confStoreApi"

	"github.com/golang/glog"
)

// Node manager
type NodeCtrler struct {
	cStore      confStoreApi.ConfStorePlugin // conf store client
	nodeDb      map[string]*Node             // DB of known nodes
	nodeEventCh chan confStoreApi.WatchServiceEvent
	watchStopCh chan bool
}

// local state
var nodeCtrl *NodeCtrler

// Create a new Node mgr
func Init(cStore confStoreApi.ConfStorePlugin) error {
	nodeCtrl = new(NodeCtrler)

	nodeCtrl.cStore = cStore

	// Initialize the node db
	nodeCtrl.nodeDb = make(map[string]*Node)

	//FIXME: we probably need a channel to stop nodeMgr loop
	// Start NodeCtrler thread
	go nodeCtrl.nodeMgrLoop()

	return nil
}

// FIXME: this is temporary till we have scheduler
func ListNodes() []*Node {
	nodeList := make([]*Node, 0)

	for _, node := range nodeCtrl.nodeDb {
		nodeList = append(nodeList, node)
	}

	return nodeList
}

// Main loop of node manager
func (self *NodeCtrler) nodeMgrLoop() {
	// Create channels for watch thread
	self.nodeEventCh = make(chan confStoreApi.WatchServiceEvent, 1)
	self.watchStopCh = make(chan bool, 1)

	// Start a watch on athena service so that we dont miss any
	err := self.cStore.WatchService("athena", self.nodeEventCh, self.watchStopCh)
	if err != nil {
		glog.Fatalf("Could not start a watch on athena service. Err: %v", err)
	}

	// Get a list of all existing athena nodes
	nodeList, err := self.cStore.GetService("athena")
	if err != nil {
		glog.Errorf("Error getting node list from cstore. Err: %v", err)
	}

	glog.Infof("Got service list: %+v", nodeList)

	// Create a node for each entry
	for _, nodeInfo := range nodeList {
		nodeKey := nodeInfo.HostAddr

		node, err := NewNode(nodeInfo.HostAddr, nodeInfo.Port)
		if err != nil {
			glog.Errorf("Error creating new node %s. Err: %v", nodeKey, err)
		}

		// Save it in the DB
		self.nodeDb[nodeKey] = node
	}

	// Go in a loop listening to events
	for {
		select {
		case srvEvent := <-self.nodeEventCh:
			glog.Infof("Received athena service watch event: %+v", srvEvent)

			// collect the info about the node
			nodeInfo := srvEvent.ServiceInfo
			nodeKey := nodeInfo.HostAddr

			// Handle based on event type
			if srvEvent.EventType == confStoreApi.WatchServiceEventAdd {
				// Add the node if we dont know about it
				if self.nodeDb[nodeKey] == nil {
					glog.Infof("Received Node add event for: %s", nodeKey)

					// Add the node
					node, err := NewNode(nodeInfo.HostAddr, nodeInfo.Port)
					if err != nil {
						glog.Errorf("Error creating new node %s. Err: %v", nodeKey, err)
					}

					// Save it in the DB
					self.nodeDb[nodeKey] = node
				} else {
					node := self.nodeDb[nodeKey]

					// Queue the up event
					node.NodeEvent("up")
				}
			} else if srvEvent.EventType == confStoreApi.WatchServiceEventDel {

				if self.nodeDb[nodeKey] != nil {
					glog.Infof("Received Node delete event for: %s", nodeKey)

					node := self.nodeDb[nodeKey]

					// Queue the up event
					node.NodeEvent("down")
				} else {
					glog.Errorf("Received delete on an unknown node: %s", nodeKey)
				}
			}
		}
	}
}

// Inform all existing nodes about a new node coming up
func nodeUpBcast(nodeAddr string) error {
	// Inform everyone except the node thats coming up
	for _, node := range nodeCtrl.nodeDb {
		if node.HostAddr != nodeAddr {
			var resp altaspec.ReqSuccess
			err := node.NodePostReq("/peer/"+nodeAddr, "", &resp)
			if err != nil {
				glog.Errorf("Error informing %s about node %s. Err: %v",
					node.HostAddr, nodeAddr, err)
			}
		}
	}

	// Inform the node about all the other nodes
	for _, node := range nodeCtrl.nodeDb {
		if node.HostAddr != nodeAddr {
			var resp altaspec.ReqSuccess
			err := NodePostReq(nodeAddr, "/peer/"+node.HostAddr, "", &resp)
			if err != nil {
				glog.Errorf("Error informing %s about node %s. Err: %v",
					nodeAddr, node.HostAddr, err)
			}
		}
	}

	// Get my address
	localIpAddr, err := nodeCtrl.cStore.GetLocalAddr()
	if err != nil {
		glog.Fatalf("Could not find a local address. Err %v", err)
		return err
	}

	// Ask the node to connect to master
	var resp bool
	client := rpcHub.Client(nodeAddr, 9002) // FIXME: get the port number from ofnet
	err = client.Call("OfnetAgent.AddMaster", &localIpAddr, &resp)
	if err != nil {
		glog.Errorf("Failed to tell node %s about the master. Err: %v", nodeAddr, err)
		return err
	}

	return nil
}

// Send network spec to all nodes
func NetSpecBcast(netSpec altaspec.AltaNetSpec) error {
	// Inform all nodes
	for _, node := range nodeCtrl.nodeDb {
		err := node.PushNetwork(netSpec)
		if err != nil {
			glog.Errorf("Error sending network info to node %s. Err: %v",
				node.HostAddr, err)
		}
	}

	return nil
}

// Perform Get request on a node
func NodeGetReq(nodeAddr string, path string, data interface{}) error {
	// Make sure noe exists
	if nodeCtrl.nodeDb[nodeAddr] == nil {
		glog.Errorf("Node %s not found", nodeAddr)
		return errors.New("Node not found")
	}

	node := nodeCtrl.nodeDb[nodeAddr]

	// Perform GET request
	return node.NodeGetReq(path, data)
}

// Perform POST request on a node
func NodePostReq(nodeAddr string, path string, req interface{}, resp interface{}) error {
	// Make sure noe exists
	if nodeCtrl.nodeDb[nodeAddr] == nil {
		glog.Errorf("Node %s not found", nodeAddr)
		return errors.New("Node not found")
	}

	node := nodeCtrl.nodeDb[nodeAddr]

	// Perform POST operation
	return node.NodePostReq(path, req, resp)
}
