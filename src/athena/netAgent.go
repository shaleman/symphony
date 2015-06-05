package main

import (
    //"fmt"
    "net"
    "errors"
    "strconv"
    "time"

    "pkg/ovsdriver"
    "pkg/altaspec"
    "pkg/netutils"
    "pkg/ofnet"

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
    ofnetAgent      *ofnet.OfnetAgent

    networkDb       map[string]*NetState
    vlanBitset      *bitset.BitSet  // Allocated Vlan Ids
    peerHostDb      map[string]*string  // Remote host IP addresses

    currPortNum     int     // Current OVS port number
    currVtepNum     int     // Current VTEP port number
}

// Create a new network agent
func NewNetAgent() *NetAgent {
    netAgent := new(NetAgent)

    // Create an OVS client
    netAgent.ovsDriver = ovsdriver.NewOvsDriver()
    bridge := netAgent.ovsDriver.OvsBridgeName

    /* FIXME: This is not required since we use unix domain sockets
     *        Uncomment it to user server mode where we listen to socket
     *        for OVS to connect to
    // Add the local controller
    err := netAgent.ovsDriver.AddController("127.0.0.1", 6633)
    if (err != nil) {
        glog.Fatalf("Failed to add local controller to OVS. Err: %v", err)
    }
    */

    // Get our local IP address
    localIpAddr, err := cStore.GetLocalAddr()
    if (err != nil) {
        glog.Fatalf("Could not find a local address to bind to. Err %v", err)
    }

    // Create an ofnet agent
    netAgent.ofnetAgent, _ = ofnet.NewOfnetAgent(bridge, net.ParseIP(localIpAddr))

    // Initialize vlan bitset
    netAgent.vlanBitset = bitset.New(4095) // usable vlans are from 1-4094
    netAgent.vlanBitset.Set(0).Set(1)   // Cant use vlan-0, vlan-1 is for default network
    netAgent.currPortNum = 1
    netAgent.currVtepNum = 1

    // Initialise the DB
    netAgent.peerHostDb = make(map[string]*string)
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
    err = self.ovsDriver.CreatePort(portName, "internal", netState.VlanTag)
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
    time.Sleep(1000 * time.Millisecond)

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

    // Get OFP port number
    ofpPort, err := self.ovsDriver.GetOfpPortNo(portName)
    if (err != nil) {
        glog.Errorf("Error getting OFP port number from OVS. Err: %v", err)
        return "", err
    }

    // First get the vlanTag for the network
    netState, err := self.GetNetwork(ifSpec.NetworkName)
    if (err != nil) {
        glog.Errorf("Network %s does not exist", ifSpec.NetworkName)
        return "", err
    }

    // Add local port to ofnet
    intfMac, _ := net.ParseMAC(ifSpec.IntfMacAddr)
    err = self.ofnetAgent.AddLocalPort(ofpPort, intfMac, uint16(netState.VlanTag),
                                    net.ParseIP(ifSpec.IntfIpv4Addr))
    if (err != nil) {
        glog.Errorf("Error adding local port %s to ofnetAgent. Err: %v", portName, err)
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

// Add a peer host. Create VTEPs associated with the peer
func (self *NetAgent) AddPeerHost(peerAddr string) error {
    // Check if the peer already exists
    if (self.peerHostDb[peerAddr] != nil) {
        return errors.New("Peer exists")
    }

    // Check if the VTEP already exists
    isPresent, vtepName := self.ovsDriver.IsVtepPresent(peerAddr)
    if (!isPresent) {
        // Derive a Vtep port name
        // FIXME: We need to do better job of recycling port numbers
        vtepName = "vtep" + strconv.Itoa(self.currVtepNum)
        for {
            self.currVtepNum++
            if (!self.ovsDriver.IsPortNamePresent(vtepName)) {
                break
            }
            vtepName = "vtep" + strconv.Itoa(self.currVtepNum)
        }

        // Create the OVS VTEP port
        err := self.ovsDriver.CreateVtep(vtepName, peerAddr)
        if (err != nil) {
            glog.Errorf("Error creating a VTEP. Err %v", err)
            return err
        }

        // Hack: Wait a second for the interface to show up
        // OVS seem to take few millisecond to create the interface
        time.Sleep(1000 * time.Millisecond)
    }

    // Get OFP port number for the VTEP
    ofpPort, err := self.ovsDriver.GetOfpPortNo(vtepName)
    if (err != nil) {
        glog.Errorf("Error getting OFP port number from OVS. Err: %v", err)
        return err
    }

    // Inform Ofnet about the VTEP
    err = self.ofnetAgent.AddVtepPort(ofpPort, net.ParseIP(peerAddr))
    if (err != nil) {
        glog.Errorf("Error adding VTEP port to ofnet. Err: ", err)
        return err
    }

    // Add it to DB
    self.peerHostDb[peerAddr] = &vtepName

    return nil
}

// Remove peer host and remove associated VTEPs
func (self *NetAgent) RemovePeerHost(peerAddr string) error {
    // find the remote host in DB
    vtepName := self.peerHostDb[peerAddr]
    if (vtepName == nil) {
        return errors.New("Peer does not exist")
    }

    // Get OFP port number for the VTEP
    ofpPort, err := self.ovsDriver.GetOfpPortNo(*vtepName)
    if (err != nil) {
        glog.Errorf("Error getting OFP port number from OVS. Err: %v", err)
        return err
    }

    // remove the VTEP from ofnet
    err = self.ofnetAgent.RemoveVtepPort(ofpPort)
    if (err != nil) {
        glog.Errorf("Error removing vtep port from ofnet. Err: %v", err)
        return err
    }

    // Ask OVS driver to delete the vtep
    err = self.ovsDriver.DeleteVtep(*vtepName)
    if (err != nil) {
        glog.Errorf("Error deleting vtep port %s. Err: %v", vtepName, err)
        return err
    }


    return nil
}
