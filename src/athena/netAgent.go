package main

import (
    //"fmt"
    "errors"
    "strconv"
    "time"

    "pkg/ovsdriver"
    "pkg/altaspec"
    "pkg/netutils"

    "github.com/jainvipin/bitset"
    "github.com/golang/glog"
)

const USABLE_VLAN_START = 2
const USABLE_VLAN_END = 4094


type NetState struct {
    Name        string
    VlanTag     uint
}

type NetAgent struct {
    ovsDriver       *ovsdriver.OvsDriver

    networkDb       map[string]*NetState
    vlanBitset      *bitset.BitSet  // Allocated Vlan Ids

    currPortNum     int     // Current OVS port number
}

// Create a new network agent
func NewNetAgent() *NetAgent {
    netAgent := new(NetAgent)

    // Create an OVS client
    netAgent.ovsDriver = ovsdriver.NewOvsDriver()

    // Initialize vlan bitset
    netAgent.vlanBitset = bitset.New(4095) // usable vlans are from 1-4094
    netAgent.vlanBitset.Set(0).Set(1)   // Cant use vlan-0, vlan-1 is for default network
    netAgent.currPortNum = 1

    // Initialise the DB
    netAgent.networkDb = make(map[string]*NetState)

    // Create default network
    netAgent.networkDb["default"] = &NetState{
        Name:       "default",
        VlanTag:    1,
    }

    return netAgent
}

// Create a network
func (self *NetAgent) CreateNetwork(name string) error {
    // find the next available vlan
    // FIXME: Move Vlan allocation to Zeus since we dont want any state here
    vlanTag, found := self.vlanBitset.NextClear(USABLE_VLAN_START)
    if (!found) {
        glog.Errorf("No available vlan Id for network %s", name)
        return errors.New("Vlan range full")
    }

    // Add it to the DB
    self.networkDb[name] = &NetState{
        Name:       name,
        VlanTag:    vlanTag,
    }

    return nil
}

// Delete a network
func (self *NetAgent) DeleteNetwork(name string) error {
    // Check if the network exists
    if (self.networkDb[name] == nil) {
        glog.Errorf("Network %s not found", name)
        return errors.New("Network not found")
    }

    // Free the vlanTag
    netState := self.networkDb[name]
    self.vlanBitset.Clear(netState.VlanTag)

    // Remove it from the DB
    delete(self.networkDb, name)

    return nil
}

// Return info about a network
func (self *NetAgent) GetNetwork(name string) (*NetState, error) {
    // Check if the network exists
    if (self.networkDb[name] == nil) {
        glog.Errorf("Network %s not found", name)
        return nil, errors.New("Network not found")
    }

    // Return the network state
    return self.networkDb[name], nil
}

// Create a network interface and return port name
func (self *NetAgent) createNetIntf(NetworkName string) (string, error) {
    // First get the vlanTag for the network
    netState, err := self.GetNetwork(NetworkName)
    if (err != nil) {
        glog.Errorf("Network %s does not exist", NetworkName)
        return "", err
    }

    // Derive a port name
    // FIXME: We need to do better job of recycling port numbers
    portName := "ovsport" + strconv.Itoa(self.currPortNum)
    for {
        self.currPortNum++
        if (!self.ovsDriver.IsPortNamePresent(portName)) {
            break
        }
        portName = "ovsport" + strconv.Itoa(self.currPortNum)
    }

    // Create the OVS port
    err = self.ovsDriver.CreatePort(portName, "internal", nil, netState.VlanTag)
    if (err != nil) {
        glog.Errorf("Error creating a port. Err %v", err)
        return "", err
    }

    // we are done
    return portName, nil
}

// Create an interface, move it to container namespace and assign mac and IP addresses
func (self *NetAgent) CreateAltaIntf(contPid int, ifNum int, ifSpec *altaspec.AltaNetIf) (string, error) {
    // Create the port
    portName, err := self.createNetIntf(ifSpec.NetworkName)
    if (err != nil) {
        glog.Errorf("Error creating network intf %+v\n. Error: %v\n", err)
        return "", err
    }

    // Hack: Wait a second for the interface to show up
    // OVS seem to take few millisecond to create the interface
    time.Sleep(100 * time.Millisecond)

    // Move it to container namespace
    err = netutils.MoveIntfToNetns(portName, contPid)
    if (err != nil) {
        glog.Errorf("Error moving network intf %s to contPid %d\n. Error: %v\n",
                    portName, contPid, err)
        return "", err
    }

    // Create interface identity
    intfIdentity := netutils.NetnsIntfIdentify{
        PortName  : "eth" + strconv.Itoa(ifNum),
        MacAddr   : ifSpec.IntfMacAddr,
        IPAddr    : ifSpec.IntfIpv4Addr,
        NetmaskLen: ifSpec.IntfIpv4Masklen,
        DefaultGw : ifSpec.Ipv4Gateway,
    }

    // Rename the intf inside the namespace and assign Mac and IP address
    err = netutils.SetNetnsIntfIdentity(contPid, portName, intfIdentity)
    if (err != nil){
        glog.Errorf("Error Setting intf %s identity: %+v\n. Error: %v\n",
                    portName, intfIdentity, err)
        return "", err
    }

    return portName, nil
}

// Delete the interface
func (self *NetAgent) DeleteAltaIntf(portName string) error {
    err := self.ovsDriver.DeletePort(portName)
    if (err != nil) {
        glog.Errorf("Error deleting port %s. Error: %v", portName, err)
    }

    return err
}