package nodeCtrler

import (
    "errors"

    // "pkg/confStore"
    "pkg/confStore/confStoreApi"

    "github.com/golang/glog"
)


// Node manager
type NodeCtrler struct {
    cStore          confStoreApi.ConfStorePlugin  // conf store client
    nodeDb          map[string]*Node    // DB of known nodes
    nodeEventCh     chan confStoreApi.WatchServiceEvent
    watchStopCh     chan bool
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
    if (err != nil) {
        glog.Fatalf("Could not start a watch on athena service. Err: %v", err)
    }

    // Get a list of all existing athena nodes
    nodeList, err := self.cStore.GetService("athena")
    if (err != nil) {
        glog.Errorf("Error getting node list from cstore. Err: %v", err)
    }

    glog.Infof("Got service list: %+v", nodeList)

    // Create a node for each entry
    for _, nodeInfo := range nodeList {
        nodeKey := nodeInfo.HostAddr

        node, err := NewNode(nodeInfo.HostAddr, nodeInfo.Port)
        if (err != nil) {
            glog.Errorf("Error creating new node %s. Err: %v", nodeKey, err)
        }

        // Save it in the DB
        self.nodeDb[nodeKey] = node
    }

    // Go in a loop listening to events
    for {
        select {
        case srvEvent := <- self.nodeEventCh:
            glog.Infof("Received athena service watch event: %+v", srvEvent)

            // collect the info about the node
            nodeInfo := srvEvent.ServiceInfo
            nodeKey := nodeInfo.HostAddr

            // Handle based on event type
            if (srvEvent.EventType == confStoreApi.WatchServiceEventAdd) {
                // Add the node if we dont know about it
                if (self.nodeDb[nodeKey] == nil) {
                    glog.Infof("Received Node add event for: %s", nodeKey)

                    // Add the node
                    node, err := NewNode(nodeInfo.HostAddr, nodeInfo.Port)
                    if (err != nil) {
                        glog.Errorf("Error creating new node %s. Err: %v", nodeKey, err)
                    }

                    // Save it in the DB
                    self.nodeDb[nodeKey] = node
                } else {
                    node := self.nodeDb[nodeKey]

                    // Queue the up event
                    node.NodeEvent("up")
                }
            } else if (srvEvent.EventType == confStoreApi.WatchServiceEventDel) {


                if (self.nodeDb[nodeKey] != nil) {
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

// Perform Get request on a node
func NodeGetReq(nodeAddr string, path string, data interface{}) error {
    // Make sure noe exists
    if (nodeCtrl.nodeDb[nodeAddr] == nil) {
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
    if (nodeCtrl.nodeDb[nodeAddr] == nil) {
        glog.Errorf("Node %s not found", nodeAddr)
        return errors.New("Node not found")
    }

    node := nodeCtrl.nodeDb[nodeAddr]

    // Perform POST operation
    return node.NodePostReq(path, req, resp)
}
