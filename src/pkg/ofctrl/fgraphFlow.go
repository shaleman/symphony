package ofctrl

// This file implements the forwarding graph API for the flow

import (
    "net"
    "encoding/json"

    "pkg/ofctrl/libOpenflow/openflow13"

    log "github.com/Sirupsen/logrus"
)

// Small subset of openflow fields we currently support
// FIXME: we need to start supporting Masks on each field
type FlowMatch struct {
    InputPort       uint32
    MacDa           *net.HardwareAddr
    MacSa           *net.HardwareAddr
    Ethertype       uint16
    VlanId          uint16
    IpSa            *net.IP
    IpDa            *net.IP
}

// additional actions in flow's instruction set
type FlowAction struct {
    actionType      string      // Type of action "setVlan", "setMetadata"
    vlanId          uint16      // Vlan Id in case of "setVlan"
    metadata        uint64      // Metadata in case of "setMetadata"
}

// State of a flow entry
type Flow struct {
    Table           *Table          // Table where this flow resides
    Priority        uint16          // Priority of the flow entry
    Match           FlowMatch       // Fields to be matched
    NextElem        FgraphElem      // Next fw graph element
    isInstalled     bool            // Is the flow installed in the switch
    flowId          uint64          // Unique ID for the flow
    flowActions     []*FlowAction   // List of flow actions
}

// string key for the flow
// FIXME: simple json conversion for now. This needs to be smarter
func (self *Flow) flowKey() string {
    jsonVal, err := json.Marshal(self)
    if (err != nil) {
        return ""
    }

    return string(jsonVal)
}


// Fgraph element type for the flow
func (self *Flow) Type() string {
    return "flow"
}

// instruction set for flow element
func (self *Flow) GetFlowInstr() openflow13.Instruction {
    log.Fatalf("Unexpected call to get flow's instruction set")
    return nil
}

// Translate our match fields into openflow 1.3 match fields
func (self *Flow) xlateMatch() openflow13.Match {
    ofMatch := openflow13.NewMatch()

    if (self.Match.InputPort != 0) {
        inportField := openflow13.NewInPortField(self.Match.InputPort)
        ofMatch.AddField(*inportField)
    }

    if (self.Match.MacDa != nil) {
        macDaField := openflow13.NewEthDstField(*self.Match.MacDa)
        ofMatch.AddField(*macDaField)
    }

    if (self.Match.MacSa != nil) {
        macSaField := openflow13.NewEthSrcField(*self.Match.MacSa)
        ofMatch.AddField(*macSaField)
    }

    if (self.Match.Ethertype != 0) {
        etypeField := openflow13.NewEthTypeField(self.Match.Ethertype)
        ofMatch.AddField(*etypeField)
    }

    if (self.Match.VlanId != 0) {
        vidField := openflow13.NewVlanIdField(self.Match.VlanId)
        ofMatch.AddField(*vidField)
    }

    if (self.Match.IpDa != nil) {
        ipDaField := openflow13.NewIpv4DstField(*self.Match.IpDa)
        ofMatch.AddField(*ipDaField)
    }

    if (self.Match.IpSa != nil) {
        ipSaField := openflow13.NewIpv4SrcField(*self.Match.IpSa)
        ofMatch.AddField(*ipSaField)
    }

    return *ofMatch
}

// Install a flow entry
func (self *Flow) install() error {
    // Create a flowmode entry
    flowMod := openflow13.NewFlowMod()
    flowMod.TableId = self.Table.TableId
    flowMod.Priority = self.Priority
    flowMod.Cookie = self.flowId

    // Add or modify
    if (!self.isInstalled) {
        flowMod.Command = openflow13.FC_ADD
    } else {
        flowMod.Command = openflow13.FC_MODIFY
    }

    // convert match fields to openflow 1.3 format
    flowMod.Match = self.xlateMatch()
    log.Printf("flow install: Match: %+v", flowMod.Match)

    // Check if there are any flow actions to perform
    for _, flowAction := range self.flowActions {
        switch(flowAction.actionType) {
        case "setVlan":
            // Push Vlan Tag action
            pushVlanAction := openflow13.NewActionPushVlan(0x8100)

            // Set Outer vlan tag field
            vlanField := openflow13.NewVlanIdField(flowAction.vlanId)
            setVlanAction := openflow13.NewActionSetField(*vlanField)

            // Apply actions instruction that contains push vlan and set vlan actions
            pushVlanInstr := openflow13.NewInstrApplyActions()
            pushVlanInstr.AddAction(pushVlanAction)
            pushVlanInstr.AddAction(setVlanAction)

            // Add the instrction to flowmod
            flowMod.AddInstruction(pushVlanInstr)

            log.Debugf("flow install. Added setVlan instr: %+v", pushVlanInstr)

        case "setMetadata":
            // Set Metadata instruction
            metadataInstr := openflow13.NewInstrWriteMetadata(flowAction.metadata, 0)

            // Add the instruction to flowmod
            flowMod.AddInstruction(metadataInstr)

        default:
            log.Fatalf("Unknown action type %s", flowAction.actionType)
        }
    }

    // Based on the next elem, decide what to install
    switch (self.NextElem.Type()) {
    case "table":
        // Get the instruction set from the element
        instr := self.NextElem.GetFlowInstr()

        // Add the instruction to flowmod
        flowMod.AddInstruction(instr)

        log.Debugf("flow install: added goto table instr: %+v", instr)

    case "output":
        // Get the instruction set from the element
        instr := self.NextElem.GetFlowInstr()

        // Add the instruction to flowmod if its not nil
        // a nil instruction means drop action
        if (instr != nil) {
            flowMod.AddInstruction(instr)

            log.Debugf("flow install: added output port instr: %+v", instr)
        }
    default:
        log.Fatalf("Unknown Fgraph element type %s", self.NextElem.Type())
    }

    log.Debugf("Sending flowmod: %+v", flowMod)

    // Send the message
    self.Table.Switch.Send(flowMod)

    // Mark it as installed
    self.isInstalled = true

    return nil
}

// Set Next element in the Fgraph. This determines what actions will be
// part of the flow's instruction set
func (self *Flow) Next(elem FgraphElem) error {
    // Set the next element in the graph
    self.NextElem = elem

    // Install the flow entry
    return self.install()
}

// Special actions on the flow to set vlan id
func (self *Flow) SetVlan(vlanId uint16) error {
    action := new(FlowAction)
    action.actionType = "setVlan"
    action.vlanId   = vlanId

    // Add to the action list
    // FIXME: detect duplicates
    self.flowActions = append(self.flowActions, action)

    // If the flow entry was already installed, re-install it
    if (self.isInstalled) {
        self.install()
    }

    return nil
}

// Special actions on the flow to set metadata
func (self *Flow) SetMetadata(metadata uint64) error {
    action := new(FlowAction)
    action.actionType = "setMetadata"
    action.metadata   = metadata

    // Add to the action list
    // FIXME: detect duplicates
    self.flowActions = append(self.flowActions, action)

    // If the flow entry was already installed, re-install it
    if (self.isInstalled) {
        self.install()
    }

    return nil
}
