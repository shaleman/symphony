package ofctrl

// This file implements the forwarding graph API for the switch

import (
    "errors"

    "pkg/ofctrl/libOpenflow/openflow13"

    // log "github.com/Sirupsen/logrus"
)

// Initialize the fgraph elements on the switch
func (self *OFSwitch) initFgraph() error {
    // Create the table DB
    self.tableDb = make(map[uint8]*Table)

    // Create the table 0
    table := new(Table)
    table.Switch = self
    table.TableId = 0
    table.flowDb = make(map[string]*Flow)
    self.tableDb[0] = table

    // Create drop action
    dropAction := new(Output)
    dropAction.outputType = "drop"
    dropAction.portNo = openflow13.P_ANY
    self.dropAction = dropAction

    // create send to controller action
    sendToCtrler := new(Output)
    sendToCtrler.outputType = "toController"
    sendToCtrler.portNo = openflow13.P_CONTROLLER
    self.sendToCtrler = sendToCtrler

    return nil
}

// Create a new table. return an error if it already exists
func (self *OFSwitch) NewTable(tableId uint8) (*Table, error) {
    // Check the parameters
    if (tableId == 0) {
        return nil, errors.New("Table 0 already exists")
    }

    // check if the table already exists
    if (self.tableDb[tableId] != nil) {
        return nil, errors.New("Table already exists")
    }

    // Create a new table
    table := new(Table)
    table.Switch = self
    table.TableId = tableId
    table.flowDb = make(map[string]*Flow)

    // Save it in the DB
    self.tableDb[tableId] = table

    return table, nil
}

// Delete a table.
// Return an error if there are fgraph nodes pointing at it
func (self *OFSwitch) DeleteTable(tableId uint8) error {
    // FIXME: to be implemented
    return nil
}

// Return table 0 which is the starting table for all packets
func (self *OFSwitch) DefaultTable() *Table {
    return self.tableDb[0]
}

// Create a new output graph element
func (self *OFSwitch) NewOutputPort(portNo uint32) (*Output, error) {
    output := new(Output)
    output.outputType = "port"
    output.portNo = portNo

    // FIXME: store all outputs in a DB

    return output, nil
}

// Return the drop graph element
func (self *OFSwitch) DropAction() *Output {
    return self.dropAction
}

// Return send to controller graph element
func (self *OFSwitch) SendToController() *Output {
    return self.sendToCtrler
}
