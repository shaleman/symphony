package ofctrl

// This file defines the forwarding graph API

import (
    "pkg/ofctrl/libOpenflow/openflow13"
)

// Example usage
// rxVlanTbl := switch.NewTable(1)
// macSaTable := switch.NewTable(2)
// macDaTable := switch.NewTable(3)
// ipTable := switch.NewTable(4)
//
// inpTable := switch.DefaultTable() // table 0. i.e starting table
// validInputPkt := inpTable.NewFlow(FlowMatch{}, 1)
//
// dscrdMcastSrc := inpTable.NewFlow(FlowMatch{
//                                  &McastSrc: { 0x01, 0, 0, 0, 0, 0 }
//                                  &McastSrcMask: { 0x01, 0, 0, 0, 0, 0 }
//                                  }, 100)
// dscrdMcastSrc.Next(switch.DropAction())
// validInputPkt.Next(rxVlanTbl)
//
// tagPort := rxVlanTbl.NewFlow(FlowMatch{
//                              InputPort: Port(0)
//                              }, 100)
// tagPort.SetVlan(10)
// tagPort.Next(macSaTable)
//
// ipFlow := ipTable.NewFlow(FlowParams{
//                           IpDa: &net.IPv4("10.10.10.10")
//                          }, 100)
//
// outPort := switch.NewOutput(OutParams{
//                              OutPort: Port(10)
//                              }, 100)
// ipFlow.Next(outPort)
//

type FgraphElem interface {
    Type() string       // Returns the type of fw graph element
    GetFlowInstr() openflow13.Instruction   // Returns the formatted instruction set
}
