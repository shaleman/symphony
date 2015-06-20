package main

import (
	//"fmt"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/contiv/ofnet"
	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/netutils"
	"github.com/contiv/symphony/pkg/ovsdriver"

	log "github.com/Sirupsen/logrus"
	"github.com/jainvipin/bitset"
)

const USABLE_VLAN_START = 2
const USABLE_VLAN_END = 4094

type NetAgent struct {
	ovsDriver  *ovsdriver.OvsDriver
	ofnetAgent *ofnet.OfnetAgent

	networkDb  map[string]*altaspec.AltaNetSpec
	vlanBitset *bitset.BitSet     // Allocated Vlan Ids
	peerHostDb map[string]*string // Remote host IP addresses

	currPortNum int // Current OVS port number
	currVtepNum int // Current VTEP port number
}

// Create a new network agent
func NewNetAgent() *NetAgent {
	netAgent := new(NetAgent)

	// Create an OVS client
	netAgent.ovsDriver = ovsdriver.NewOvsDriver()
	bridge := netAgent.ovsDriver.OvsBridgeName

	// Add the local controller
	err := netAgent.ovsDriver.AddController("127.0.0.1", 6633)
	if err != nil {
		log.Fatalf("Failed to add local controller to OVS. Err: %v", err)
	}

	// Get our local IP address
	localIpAddr, err := cStore.GetLocalAddr()
	if err != nil {
		log.Fatalf("Could not find a local address to bind to. Err %v", err)
	}

	// Create an ofnet agent
	netAgent.ofnetAgent, _ = ofnet.NewOfnetAgent(bridge, "vxlan", net.ParseIP(localIpAddr))

	// Initialize vlan bitset
	netAgent.vlanBitset = bitset.New(4095) // usable vlans are from 1-4094
	netAgent.vlanBitset.Set(0).Set(1)      // Cant use vlan-0, vlan-1 is for default network
	netAgent.currPortNum = 1
	netAgent.currVtepNum = 1

	// Initialise the DB
	netAgent.peerHostDb = make(map[string]*string)
	netAgent.networkDb = make(map[string]*altaspec.AltaNetSpec)

	// Create default network
	netAgent.networkDb["default"] = &altaspec.AltaNetSpec{
		NetworkName: "default",
		VlanId:      1,
		Vni:         1,
	}

	return netAgent
}

// Create a network
func (self *NetAgent) CreateNetwork(netSpec altaspec.AltaNetSpec) error {
	// Default network is already created
	if netSpec.NetworkName == "default" {
		return nil
	}

	// Add it to the DB
	self.networkDb[netSpec.NetworkName] = &netSpec

	// Add vlan mapping
	err := netAgent.ofnetAgent.AddVlan(netSpec.VlanId, netSpec.Vni)
	if err != nil {
		log.Errorf("Error adding vlan for net %+v. Err: %v", netSpec, err)
		return err
	}

	return nil
}

// Delete a network
func (self *NetAgent) DeleteNetwork(name string) error {
	// Check if the network exists
	if self.networkDb[name] == nil {
		log.Errorf("Network %s not found", name)
		return errors.New("Network not found")
	}

	// Remove the VLAN mapping
	network := self.networkDb[name]
	err := netAgent.ofnetAgent.RemoveVlan(network.VlanId, network.Vni)
	if err != nil {
		log.Errorf("Error removing vlan for net %+v. Err: %v", network, err)
	}

	// Remove it from the DB
	delete(self.networkDb, name)

	return nil
}

// Return info about a network
func (self *NetAgent) GetNetwork(name string) (*altaspec.AltaNetSpec, error) {
	// Check if the network exists
	if self.networkDb[name] == nil {
		log.Errorf("Network %s not found", name)
		return nil, errors.New("Network not found")
	}

	// Return the network state
	return self.networkDb[name], nil
}

// Create a network interface and return port name
func (self *NetAgent) createNetIntf(NetworkName string) (string, error) {
	// First get the vlanTag for the network
	netState, err := self.GetNetwork(NetworkName)
	if err != nil {
		log.Errorf("Network %s does not exist", NetworkName)
		return "", err
	}

	// Derive a port name
	// FIXME: We need to do better job of recycling port numbers
	portName := "ovsport" + strconv.Itoa(self.currPortNum)
	for {
		self.currPortNum++
		if !self.ovsDriver.IsPortNamePresent(portName) {
			break
		}
		portName = "ovsport" + strconv.Itoa(self.currPortNum)
	}

	// Create the OVS port
	err = self.ovsDriver.CreatePort(portName, "internal", uint(netState.VlanId))
	if err != nil {
		log.Errorf("Error creating a port. Err %v", err)
		return "", err
	}

	// we are done
	return portName, nil
}

// Create an interface, move it to container namespace and assign mac and IP addresses
func (self *NetAgent) CreateAltaIntf(contPid int, ifNum int, ifSpec *altaspec.AltaNetIf) (string, error) {
	// Create the port
	portName, err := self.createNetIntf(ifSpec.NetworkName)
	if err != nil {
		log.Errorf("Error creating network intf %+v\n. Error: %v\n", err)
		return "", err
	}

	// Hack: Wait a second for the interface to show up
	// OVS seem to take few millisecond to create the interface
	time.Sleep(1000 * time.Millisecond)

	// Move it to container namespace
	err = netutils.MoveIntfToNetns(portName, contPid)
	if err != nil {
		log.Errorf("Error moving network intf %s to contPid %d\n. Error: %v\n",
			portName, contPid, err)
		return "", err
	}

	// Create interface identity
	intfIdentity := netutils.NetnsIntfIdentify{
		PortName:   "eth" + strconv.Itoa(ifNum),
		MacAddr:    ifSpec.IntfMacAddr,
		IPAddr:     ifSpec.IntfIpv4Addr,
		NetmaskLen: ifSpec.IntfIpv4Masklen,
		DefaultGw:  ifSpec.Ipv4Gateway,
	}

	// Rename the intf inside the namespace and assign Mac and IP address
	err = netutils.SetNetnsIntfIdentity(contPid, portName, intfIdentity)
	if err != nil {
		log.Errorf("Error Setting intf %s identity: %+v\n. Error: %v\n",
			portName, intfIdentity, err)
		return "", err
	}

	// Get OFP port number
	ofpPort, err := self.ovsDriver.GetOfpPortNo(portName)
	if err != nil {
		log.Errorf("Error getting OFP port number from OVS. Err: %v", err)
		return "", err
	}

	// First get the vlanTag for the network
	netState, err := self.GetNetwork(ifSpec.NetworkName)
	if err != nil {
		log.Errorf("Network %s does not exist", ifSpec.NetworkName)
		return "", err
	}

	intfMac, _ := net.ParseMAC(ifSpec.IntfMacAddr)
	endpoint := ofnet.EndpointInfo{
		PortNo:  ofpPort,
		MacAddr: intfMac,
		Vlan:    uint16(netState.VlanId),
		IpAddr:  net.ParseIP(ifSpec.IntfIpv4Addr),
	}

	// Add local port to ofnet
	err = self.ofnetAgent.AddLocalEndpoint(endpoint)
	if err != nil {
		log.Errorf("Error adding local port %s to ofnetAgent. Err: %v", portName, err)
		return "", err
	}

	return portName, nil
}

// Delete the interface
func (self *NetAgent) DeleteAltaIntf(portName string) error {
	// Get OFP port number
	ofpPort, err := self.ovsDriver.GetOfpPortNo(portName)
	if err != nil {
		log.Errorf("Error getting OFP port number from OVS. Err: %v", err)
		return err
	}

	// Remove the endpoint from ofnet agent
	err = self.ofnetAgent.RemoveLocalEndpoint(ofpPort)
	if err != nil {
		log.Errorf("Failed to remove ofnet port: %s", portName)
		return err
	}

	// Finally delete the port in OVS
	err = self.ovsDriver.DeletePort(portName)
	if err != nil {
		log.Errorf("Error deleting port %s. Error: %v", portName, err)
	}

	return err
}

// Add a peer host. Create VTEPs associated with the peer
func (self *NetAgent) AddPeerHost(peerAddr string) error {
	// Check if the peer already exists
	if self.peerHostDb[peerAddr] != nil {
		return errors.New("Peer exists")
	}

	// Check if the VTEP already exists
	isPresent, vtepName := self.ovsDriver.IsVtepPresent(peerAddr)
	if !isPresent {
		// Derive a Vtep port name
		// FIXME: We need to do better job of recycling port numbers
		vtepName = "vtep" + strconv.Itoa(self.currVtepNum)
		for {
			self.currVtepNum++
			if !self.ovsDriver.IsPortNamePresent(vtepName) {
				break
			}
			vtepName = "vtep" + strconv.Itoa(self.currVtepNum)
		}

		// Create the OVS VTEP port
		err := self.ovsDriver.CreateVtep(vtepName, peerAddr)
		if err != nil {
			log.Errorf("Error creating a VTEP. Err %v", err)
			return err
		}

		// Hack: Wait a second for the interface to show up
		// OVS seem to take few millisecond to create the interface
		time.Sleep(1000 * time.Millisecond)
	}

	// Get OFP port number for the VTEP
	ofpPort, err := self.ovsDriver.GetOfpPortNo(vtepName)
	if err != nil {
		log.Errorf("Error getting OFP port number from OVS. Err: %v", err)
		return err
	}

	// Inform Ofnet about the VTEP
	err = self.ofnetAgent.AddVtepPort(ofpPort, net.ParseIP(peerAddr))
	if err != nil {
		log.Errorf("Error adding VTEP port to ofnet. Err: ", err)
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
	if vtepName == nil {
		return errors.New("Peer does not exist")
	}

	// Get OFP port number for the VTEP
	ofpPort, err := self.ovsDriver.GetOfpPortNo(*vtepName)
	if err != nil {
		log.Errorf("Error getting OFP port number from OVS. Err: %v", err)
		return err
	}

	// remove the VTEP from ofnet
	err = self.ofnetAgent.RemoveVtepPort(ofpPort, net.ParseIP(peerAddr))
	if err != nil {
		log.Errorf("Error removing vtep port from ofnet. Err: %v", err)
		return err
	}

	// Ask OVS driver to delete the vtep
	err = self.ovsDriver.DeleteVtep(*vtepName)
	if err != nil {
		log.Errorf("Error deleting vtep port %s. Err: %v", vtepName, err)
		return err
	}

	return nil
}
