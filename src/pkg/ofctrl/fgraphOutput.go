package ofctrl

// This file implements the forwarding graph API for the output element

import (
    "pkg/ofctrl/libOpenflow/openflow13"

    // log "github.com/Sirupsen/logrus"
)

type Output struct {
    outputType      string      // Output type: "drop", "toController" or "port"
    portNo          uint32      // Output port number
}


// Fgraph element type for the output
func (self *Output) Type() string {
    return "output"
}

// instruction set for output element
func (self *Output) GetFlowInstr() openflow13.Instruction {
    outputInstr := openflow13.NewInstrApplyActions()

    switch (self.outputType) {
    case "drop":
        return nil
    case "toController":
        outputAct := openflow13.NewActionOutput(openflow13.P_CONTROLLER)
        outputInstr.AddAction(outputAct)
    case "port":
        outputAct := openflow13.NewActionOutput(self.portNo)
        outputInstr.AddAction(outputAct)
    }

    return outputInstr
}