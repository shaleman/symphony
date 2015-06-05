package ofctrl

// This file implements the forwarding graph API for the table

import (
    "errors"

    "github.com/shaleman/libOpenflow/openflow13"

    //log "github.com/Sirupsen/logrus"
)

// Fgraph table element
type Table struct {
    Switch      *OFSwitch
    TableId     uint8
    flowDb      map[string]*Flow    // database of flow entries
}


// Fgraph element type for table
func (self *Table) Type() string {
    return "table"
}

// instruction set for table element
func (self *Table) GetFlowInstr() openflow13.Instruction {
    return openflow13.NewInstrGotoTable(self.TableId)
}

// FIXME: global unique flow cookie
var globalFlowId uint64 = 0

// Create a new flow on the table
func (self *Table) NewFlow(match FlowMatch) (*Flow, error) {
    flow := new(Flow)
    flow.Table = self
    flow.Match = match
    flow.isInstalled = false
    flow.flowId = globalFlowId // FIXME: need a better id allocation
    globalFlowId += 1
    flow.flowActions = make([]*FlowAction, 0)

    // See if the flow already exists
    flowKey := flow.flowKey()
    if (self.flowDb[flowKey] != nil) {
        return nil, errors.New("Flow already exists")
    }

    // Save it in DB. We dont install the flow till its next graph elem is set
    self.flowDb[flowKey] = flow

    return flow, nil
}

// Delete a flow from the table
func (self *Table) DeleteFlow(match FlowMatch) error {
    // FIXME: to be implemented
    return nil
}
