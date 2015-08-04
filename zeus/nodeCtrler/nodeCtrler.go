package nodeCtrler

import (
	"errors"
	"time"

	"github.com/contiv/objmodel/objdb"
	"github.com/contiv/symphony/pkg/altaspec"

	"github.com/contiv/symphony/zeus/common"

	log "github.com/Sirupsen/logrus"
)

// Node manager
type NodeCtrler struct {
	cdb         objdb.ObjdbApi      // conf store client
	ctrlers     *common.ZeusCtrlers // All the controllers we can talk to
	nodeDb      map[string]*Node    // DB of known nodes
	nodeEventCh chan objdb.WatchServiceEvent
	watchStopCh chan bool
}

// local state
var nodeCtrl *NodeCtrler

// Create a new Node mgr
func Init(cdb objdb.ObjdbApi, ctrlers *common.ZeusCtrlers) error {
	nodeCtrl = new(NodeCtrler)

	nodeCtrl.cdb = cdb
	nodeCtrl.ctrlers = ctrlers

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
	self.nodeEventCh = make(chan objdb.WatchServiceEvent, 1)
	self.watchStopCh = make(chan bool, 1)

	// Start slow. give time for other services to restore state and be ready..
	time.Sleep(3 * time.Second)

	// Start a watch on athena service so that we dont miss any
	err := self.cdb.WatchService("athena", self.nodeEventCh, self.watchStopCh)
	if err != nil {
		log.Fatalf("Could not start a watch on athena service. Err: %v", err)
	}

	// Get a list of all existing athena nodes
	nodeList, err := self.cdb.GetService("athena")
	if err != nil {
		log.Errorf("Error getting node list from cdb. Err: %v", err)
	}

	log.Infof("Got service list: %+v", nodeList)

	// Create a node for each entry
	for _, nodeInfo := range nodeList {
		nodeKey := nodeInfo.HostAddr

		node, err := NewNode(nodeInfo.HostAddr, nodeInfo.Port)
		if err != nil {
			log.Errorf("Error creating new node %s. Err: %v", nodeKey, err)
		}

		// Save it in the DB
		self.nodeDb[nodeKey] = node
	}

	// Go in a loop listening to events
	for {
		select {
		case srvEvent := <-self.nodeEventCh:
			log.Infof("Received athena service watch event: %+v", srvEvent)

			// collect the info about the node
			nodeInfo := srvEvent.ServiceInfo
			nodeKey := nodeInfo.HostAddr

			// Handle based on event type
			if srvEvent.EventType == objdb.WatchServiceEventAdd {
				// Add the node if we dont know about it
				if self.nodeDb[nodeKey] == nil {
					log.Infof("Received Node add event for: %s", nodeKey)

					// Add the node
					node, err := NewNode(nodeInfo.HostAddr, nodeInfo.Port)
					if err != nil {
						log.Errorf("Error creating new node %s. Err: %v", nodeKey, err)
					}

					// Save it in the DB
					self.nodeDb[nodeKey] = node
				} else {
					node := self.nodeDb[nodeKey]

					// Queue the up event
					node.NodeEvent("up")
				}
			} else if srvEvent.EventType == objdb.WatchServiceEventDel {

				if self.nodeDb[nodeKey] != nil {
					log.Infof("Received Node delete event for: %s", nodeKey)

					node := self.nodeDb[nodeKey]

					// Queue the up event
					node.NodeEvent("down")
				} else {
					log.Errorf("Received delete on an unknown node: %s", nodeKey)
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
				log.Errorf("Error informing %s about node %s. Err: %v",
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
				log.Errorf("Error informing %s about node %s. Err: %v",
					nodeAddr, node.HostAddr, err)
			}
		}
	}

	return nil
}

// FIXME: This is not needed anymore
// Send network spec to all nodes
func NetSpecBcast(netSpec altaspec.AltaNetSpec) error {
	// Inform all nodes
	for _, node := range nodeCtrl.nodeDb {
		err := node.PushNetwork(netSpec)
		if err != nil {
			log.Errorf("Error sending network info to node %s. Err: %v",
				node.HostAddr, err)
		}
	}

	return nil
}

// Perform Get request on a node
func NodeGetReq(nodeAddr string, path string, data interface{}) error {
	// Make sure noe exists
	if nodeCtrl.nodeDb[nodeAddr] == nil {
		log.Errorf("Node %s not found", nodeAddr)
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
		log.Errorf("Node %s not found", nodeAddr)
		return errors.New("Node not found")
	}

	node := nodeCtrl.nodeDb[nodeAddr]

	// Perform POST operation
	return node.NodePostReq(path, req, resp)
}
