package ovsdriver

import (
    "fmt"
    "log"
    "os"
    "reflect"
    "time"
    // "strconv"
    // "strings"
    "errors"

    "github.com/golang/glog"
    "github.com/contiv/libovsdb"
)

// OVS driver state
type OvsDriver struct {
    // OVS client
    ovsClient *libovsdb.OvsdbClient

    // Name of the OVS bridge
    ovsBridgeName   string

    // OVSDB cache
    ovsdbCache map[string]map[string]libovsdb.Row
}



// Create a new OVS driver
func NewOvsDriver() *OvsDriver {
    ovsDriver := new(OvsDriver)

    // Hack:
    log.SetOutput(os.Stdout)

    // connect to OVS
    ovs, err := libovsdb.Connect("localhost", 6640)
    if err != nil {
        glog.Fatal("Failed to connect to ovsdb")
    }

    // Setup state
    ovsDriver.ovsClient  = ovs
    ovsDriver.ovsBridgeName = "ovsbr11"
    ovsDriver.ovsdbCache = make(map[string]map[string]libovsdb.Row)

    go func() {
        // Register for notifications
        ovs.Register(ovsDriver)

        // Populate initial state into cache
        initial, _ := ovs.MonitorAll("Open_vSwitch", "")
        ovsDriver.populateCache(*initial)
    }()

    // HACK: sleep the main thread so that Cache can be populated
    time.Sleep(1 * time.Second)

    // Create the default bridge instance
    err = ovsDriver.CreateBridge(ovsDriver.ovsBridgeName)
    if (err != nil) {
        glog.Errorf("Error creating the default bridge. It probably already exists")
        glog.Errorf("Error: %v", err)
    }

    // Return the new OVS driver
    return ovsDriver
}

// Populate local cache of ovs state
func (self *OvsDriver) populateCache(updates libovsdb.TableUpdates) {
    for table, tableUpdate := range updates.Updates {
        if _, ok := self.ovsdbCache[table]; !ok {
            self.ovsdbCache[table] = make(map[string]libovsdb.Row)

        }
        for uuid, row := range tableUpdate.Rows {
            empty := libovsdb.Row{}
            if !reflect.DeepEqual(row.New, empty) {
                self.ovsdbCache[table][uuid] = row.New
            } else {
                delete(self.ovsdbCache[table], uuid)
            }
        }
    }
}

// Dump the contents of the cache into stdout
func (self *OvsDriver) PrintCache() {
    fmt.Printf("OvsDB Cache: \n")
    for tName, table := range self.ovsdbCache {
        fmt.Printf("Table: %s\n", tName)
        for uuid, row := range table {
            fmt.Printf("  Row: UUID: %s\n", uuid)
            for fieldName, value := range row.Fields {
                fmt.Printf("    Field: %s, Value: %+v\n", fieldName, value)
            }
        }
    }
}
// Get the UUID for root
func (self *OvsDriver) getRootUuid() libovsdb.UUID {
    for uuid, _ := range self.ovsdbCache["Open_vSwitch"] {
        return libovsdb.UUID{uuid}
    }
    return libovsdb.UUID{}
}

// Wrapper for ovsDB transaction
func (self *OvsDriver) ovsdbTransact(ops []libovsdb.Operation) error {
    // Print out what we are sending
    fmt.Printf("Transaction: %+v\n", ops)

    // Perform OVSDB transaction
    reply, _ := self.ovsClient.Transact("Open_vSwitch", ops...)

    if len(reply) < len(ops) {
        glog.Errorf("Unexpected number of replies. Expected: %d, Recvd: %d", len(ops), len(reply))
        return errors.New("OVS transaction failed. Unexpected number of replies")
    }

    // Parse reply and look for errors
    for i, o := range reply {
        if o.Error != "" && i < len(ops) {
            return errors.New("OVS Transaction failed err " + o.Error + "Details: " + o.Details)
        } else if o.Error != "" {
            return errors.New("OVS Transaction failed err " + o.Error + "Details: " + o.Details)
        }
    }

    // Return success
    return nil
}

// **************** OVS driver API ********************
func (self *OvsDriver) CreateBridge(bridgeName string) error {
    namedUuidStr := "dummy"

    // simple insert/delete operation
    brOp := libovsdb.Operation{}
    bridge := make(map[string]interface{})
    bridge["name"] = bridgeName
    brOp = libovsdb.Operation{
        Op:       "insert",
        Table:    "Bridge",
        Row:      bridge,
        UUIDName: namedUuidStr,
    }


    // Inserting/Deleting a Bridge row in Bridge table requires mutating
    // the open_vswitch table.
    brUuid := []libovsdb.UUID{libovsdb.UUID{namedUuidStr}}
    mutateUuid := brUuid
    mutateSet, _ := libovsdb.NewOvsSet(mutateUuid)
    mutation := libovsdb.NewMutation("bridges", "insert", mutateSet)
    condition := libovsdb.NewCondition("_uuid", "==", self.getRootUuid())

    // simple mutate operation
    mutateOp := libovsdb.Operation{
        Op:        "mutate",
        Table:     "Open_vSwitch",
        Mutations: []interface{}{mutation},
        Where:     []interface{}{condition},
    }

    operations := []libovsdb.Operation{brOp, mutateOp}

    // operations := []libovsdb.Operation{brOp}
    return self.ovsdbTransact(operations)
}

// Delete a bridge from ov
func (self *OvsDriver) DeleteBridge(bridgeName string) error {
    namedUuidStr := "dummy"
    brUuid := []libovsdb.UUID{libovsdb.UUID{namedUuidStr}}


    // simple insert/delete operation
    brOp := libovsdb.Operation{}
    condition := libovsdb.NewCondition("name", "==", bridgeName)
    brOp = libovsdb.Operation{
        Op:    "delete",
        Table: "Bridge",
        Where: []interface{}{condition},
    }
    // also fetch the br-uuid from cache
    for uuid, row := range self.ovsdbCache["Bridge"] {
        name := row.Fields["name"].(string)
        if name == bridgeName {
            brUuid = []libovsdb.UUID{libovsdb.UUID{uuid}}
            break
        }
    }

    // Inserting/Deleting a Bridge row in Bridge table requires mutating
    // the open_vswitch table.
    mutateUuid := brUuid
    mutateSet, _ := libovsdb.NewOvsSet(mutateUuid)
    mutation := libovsdb.NewMutation("bridges", "delete", mutateSet)
    condition = libovsdb.NewCondition("_uuid", "==", self.getRootUuid())

    // simple mutate operation
    mutateOp := libovsdb.Operation{
        Op:        "mutate",
        Table:     "Open_vSwitch",
        Mutations: []interface{}{mutation},
        Where:     []interface{}{condition},
    }

    operations := []libovsdb.Operation{brOp, mutateOp}
    return self.ovsdbTransact(operations)
}

func (self *OvsDriver) CreatePort(intfName, intfType string, intfOptions map[string]interface{}, vlanTag uint) error {
    portUuidStr := intfName
    intfUuidStr := fmt.Sprintf("Intf%s", intfName)
    portUuid := []libovsdb.UUID{libovsdb.UUID{portUuidStr}}
    intfUuid := []libovsdb.UUID{libovsdb.UUID{intfUuidStr}}
    opStr := "insert"
    var err error = nil

    // insert/delete a row in Interface table
    intf := make(map[string]interface{})
    intf["name"] = intfName
    intf["type"] = intfType

    //idMap := make(map[string]string)
    // idMap["endpoint-id"] = id
    // intf["external_ids"], err = libovsdb.NewOvsMap(idMap)
    // if err != nil {
    //    return err
    // }

    // Handle special options for Vxlan vTEPs
    if intfOptions != nil {
        intf["options"], err = libovsdb.NewOvsMap(intfOptions)
        if err != nil {
            glog.Infof("error '%s' creating options from %v \n", err, intfOptions)
            return err
        }
    }

    // Add an entry in Interface table
    intfOp := libovsdb.Operation{
        Op:       opStr,
        Table:    "Interface",
        Row:      intf,
        UUIDName: intfUuidStr,
    }


    // insert/delete a row in Port table
    port := make(map[string]interface{})
    port["name"] = intfName
    if vlanTag != 0 {
        port["vlan_mode"] = "access"
        port["tag"] = vlanTag
    } else {
        port["vlan_mode"] = "trunk"
    }

    port["interfaces"], err = libovsdb.NewOvsSet(intfUuid)
    if err != nil {
        return err
    }

    // port["external_ids"], err = libovsdb.NewOvsMap(idMap)
    // if err != nil {
    //    return err
    // }

    // Add an entry in Port table
    portOp := libovsdb.Operation{
        Op:       opStr,
        Table:    "Port",
        Row:      port,
        UUIDName: portUuidStr,
    }

    // mutate the Ports column of the row in the Bridge table
    mutateSet, _ := libovsdb.NewOvsSet(portUuid)
    mutation := libovsdb.NewMutation("ports", opStr, mutateSet)
    condition := libovsdb.NewCondition("name", "==", self.ovsBridgeName)
    mutateOp := libovsdb.Operation{
        Op:        "mutate",
        Table:     "Bridge",
        Mutations: []interface{}{mutation},
        Where:     []interface{}{condition},
    }

    // Perform OVS transaction
    operations := []libovsdb.Operation{intfOp, portOp, mutateOp}
    return self.ovsdbTransact(operations)
}

func (self *OvsDriver) DeletePort(intfName string) error {
    portUuidStr := intfName
    portUuid := []libovsdb.UUID{libovsdb.UUID{portUuidStr}}
    opStr := "delete"

    // insert/delete a row in Interface table
    condition := libovsdb.NewCondition("name", "==", intfName)
    intfOp := libovsdb.Operation{
        Op:    opStr,
        Table: "Interface",
        Where: []interface{}{condition},
    }

    // insert/delete a row in Port table
    condition = libovsdb.NewCondition("name", "==", intfName)
    portOp := libovsdb.Operation{
        Op:    opStr,
        Table: "Port",
        Where: []interface{}{condition},
    }

    // also fetch the port-uuid from cache
    for uuid, row := range self.ovsdbCache["Port"] {
        name := row.Fields["name"].(string)
        if name == intfName {
            portUuid = []libovsdb.UUID{libovsdb.UUID{uuid}}
            break
        }
    }

    // mutate the Ports column of the row in the Bridge table
    mutateSet, _ := libovsdb.NewOvsSet(portUuid)
    mutation := libovsdb.NewMutation("ports", opStr, mutateSet)
    condition = libovsdb.NewCondition("name", "==", self.ovsBridgeName)
    mutateOp := libovsdb.Operation{
        Op:        "mutate",
        Table:     "Bridge",
        Mutations: []interface{}{mutation},
        Where:     []interface{}{condition},
    }

    // Perform OVS transaction
    operations := []libovsdb.Operation{intfOp, portOp, mutateOp}
    return self.ovsdbTransact(operations)
}

// Check the local cache and see if the portname is taken already
// HACK alert: This is used to pick next port number instead of managing
//    port number space actively across agent restarts
func (self *OvsDriver) IsPortNamePresent(intfName string) bool {
    for tName, table := range self.ovsdbCache {
        if tName == "Port" {
            for _, row := range table {
                for fieldName, value := range row.Fields {
                    if fieldName == "name" {
                        if value == intfName {
                            // Interface name exists.
                            return true
                        }
                    }
                }
            }
        }
    }

    // We could not find the interface name
    return false
}

// ************************ Notification handler for OVS DB changes ****************
func (self *OvsDriver) Update(context interface{}, tableUpdates libovsdb.TableUpdates) {
    // fmt.Printf("Received OVS update: %+v\n\n", tableUpdates)
    self.populateCache(tableUpdates)
}
func (self *OvsDriver) Disconnected(ovsClient *libovsdb.OvsdbClient) {
    glog.Errorf("OVS BD client disconnected")
}
func (self *OvsDriver) Locked([]interface{}) {
}
func (self *OvsDriver) Stolen([]interface{}) {
}
func (self *OvsDriver) Echo([]interface{}) {
}
