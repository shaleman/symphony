package ofnet

// This file implements ofnet agent API which runs on each host alongside OVS.
// This assumes:
//      - ofnet agent is running on each host
//      - There is single OVS switch instance(aka bridge instance)
//      - OVS switch's forwarding is fully controller by ofnet agent
//
// It also assumes OVS is configured for openflow1.3 version and configured
// to connect to controller on specified port

import (
    //"fmt"
    "net"
    "time"
    "errors"

    "pkg/ofctrl"
    "github.com/shaleman/libOpenflow/openflow13"
    "github.com/shaleman/libOpenflow/protocol"
    "pkg/rpcHub"

    log "github.com/Sirupsen/logrus"
)

// OfnetAgent state
type OfnetAgent struct {
    ctrler      *ofctrl.Controller      // Controller instance
    ofSwitch    *ofctrl.OFSwitch        // Switch instance. Assumes single switch per agent
    localIp     net.IP                  // Local IP to be used for tunnel end points
    masterDb    map[string]*net.IP      // list of Master's IP address

    // Fgraph tables
    inputTable  *ofctrl.Table           // Packet lookup starts here
    vlanTable   *ofctrl.Table           // Vlan Table. map port or VNI to vlan
    ipTable     *ofctrl.Table           // IP lookup table

    // Port and VNI to vlan mapping table
    portVlanMap map[uint32]*uint16       // Map port number to vlan
    vniVlanMap  map[uint32]*uint16       // Map VNI to vlan
    vlanVniMap  map[uint16]*uint32       // Map vlan to VNI

    // VTEP database
    vtepTable   map[string]*uint32      // Map vtep IP to OVS port number

    // Routing table
    routeTable  map[string]*OfnetRoute  // routes indexed by ip addr

    // Router Mac to be used
    myRouterMac net.HardwareAddr
}

// IP Route information
type OfnetRoute struct {
    IpAddr          net.IP      // IP address of the end point
    VrfId           uint16      // IP address namespace
    OriginatorIp    net.IP      // Originating switch
    PortNo          uint32      // Port number on originating switch
    Timestamp       time.Time   // Timestamp of the last event
}

const FLOW_MATCH_PRIORITY = 100     // Priority for all match flows
const FLOW_MISS_PRIORITY = 1        // priority for table miss flow


// Create a new Ofnet agent and initialize it
func NewOfnetAgent(bridge string, localIp net.IP) (*OfnetAgent, error) {
    agent := new(OfnetAgent)

    // Init params
    agent.localIp = localIp
    agent.masterDb = make(map[string]*net.IP)
    agent.portVlanMap = make(map[uint32]*uint16)
    agent.vniVlanMap = make(map[uint32]*uint16)
    agent.vlanVniMap = make(map[uint16]*uint32)
    agent.routeTable = make(map[string]*OfnetRoute)
    agent.vtepTable = make(map[string]*uint32)

    agent.myRouterMac, _ = net.ParseMAC("00:00:11:11:11:11")

    // Create an openflow controller
    agent.ctrler = ofctrl.NewController(bridge, agent)


    // Create rpc server
    rpcServ := rpcHub.NewRpcServer(9002)
    rpcServ.Register(agent)

    // Return it
    return agent, nil
}

// Handle switch connected event
func (self *OfnetAgent) SwitchConnected(sw *ofctrl.OFSwitch) {
    log.Infof("Switch %v connected", sw.DPID())

    // store it for future use.
    self.ofSwitch = sw

    // Init the Fgraph
    self.initFgraph()
}

// Handle switch disconnect event
func (self *OfnetAgent) SwitchDisconnected(sw *ofctrl.OFSwitch) {
    log.Infof("Switch %v disconnected", sw.DPID())
}

// Receive a packet from the switch.
func (self *OfnetAgent) PacketRcvd(sw *ofctrl.OFSwitch, pkt *openflow13.PacketIn) {
    log.Infof("Packet received from switch %v. Packet: %+v", sw.DPID(), pkt)
    log.Infof("Input Port: %+v", pkt.Match.Fields[0].Value)
    switch(pkt.Data.Ethertype) {
    case 0x0806:
        if ((pkt.Match.Type == openflow13.MatchType_OXM) &&
            (pkt.Match.Fields[0].Class == openflow13.OXM_CLASS_OPENFLOW_BASIC) &&
            (pkt.Match.Fields[0].Field == openflow13.OXM_FIELD_IN_PORT)) {
            // Get the input port number
            switch t := pkt.Match.Fields[0].Value.(type) {
            case *openflow13.InPortField:
                var inPortFld openflow13.InPortField
                inPortFld = *t

                self.processArp(pkt.Data, inPortFld.InPort)
            }

        }

    case 0x0800:
    default:
        log.Errorf("Received unknown ethertype: %x", pkt.Data.Ethertype)
    }
}

// Add a master
// ofnet agent tries to connect to the master and download routes
func (self *OfnetAgent) AddMaster(masterAddr *string, ret *bool) error {
    myAddr := self.localIp.String()
    masterIp := net.ParseIP(*masterAddr)
    var resp bool

    log.Infof("Adding master: %s", *masterAddr)

    // Save it in DB
    self.masterDb[*masterAddr] = &masterIp

    // Register the agent with the master
    err := rpcHub.Client(*masterAddr, 9001).Call("OfnetMaster.RegisterNode", &myAddr, &resp)
    if (err != nil) {
        log.Fatalf("Failed to register with the master %s. Err: %v", masterAddr, err)
        return err
    }

    return nil
}

// Add a local port.
// This takes ofp port number, mac address, vlan and IP address of the port.
func (self *OfnetAgent) AddLocalPort(portNo uint32, macAddr net.HardwareAddr,
                                        vlan uint16, ipAddr net.IP) error {
    // Add port vlan mapping
    self.portVlanMap[portNo] = &vlan

    // Install a flow entry for vlan mapping and point it to IP table
    portVlanFlow, _ := self.vlanTable.NewFlow(ofctrl.FlowMatch{
                            Priority: FLOW_MATCH_PRIORITY,
                            InputPort: portNo,
                        })
    portVlanFlow.SetVlan(vlan)
    portVlanFlow.Next(self.ipTable)

    // build the route to add
    route := OfnetRoute{
                IpAddr: ipAddr,
                VrfId: 0,       // FIXME: get a VRF
                OriginatorIp: self.localIp,
                PortNo: portNo,
                Timestamp:  time.Now(),
            }

    // Add the route to local and master's routing table
    self.localRouteAdd(&route)

    // Create the output port
    outPort, _ := self.ofSwitch.NewOutputPort(portNo)

    // Install the IP address
    ipFlow, _ := self.ipTable.NewFlow(ofctrl.FlowMatch{
                            Priority: FLOW_MATCH_PRIORITY,
                            Ethertype: 0x0800,
                            IpDa: &ipAddr,
                        })
    ipFlow.SetMacDa(macAddr)
    ipFlow.SetMacSa(self.myRouterMac)
    ipFlow.Next(outPort)

    return nil
}

// Remove local port
func (self *OfnetAgent) RemoveLocalPort(portNo uint32) error {
    // FIXME:
    return nil
}

// Add virtual tunnel end point. This is mainly used for mapping remote vtep IP
// to ofp port number.
func (self *OfnetAgent) AddVtepPort(portNo uint32, remoteIp net.IP) error {
    log.Infof("Adding VTEP port(%d), Remote IP: %v", portNo, remoteIp)

    // Store the vtep IP to port number mapping
    self.vtepTable[remoteIp.String()] = &portNo

    // Install a flow entry for default VNI/vlan and point it to IP table
    // FIXME: Need to match on tunnelId and set good vlan id
    portVlanFlow, _ := self.vlanTable.NewFlow(ofctrl.FlowMatch{
                            Priority: FLOW_MATCH_PRIORITY,
                            InputPort: portNo,
                        })
    portVlanFlow.SetVlan(1)
    portVlanFlow.Next(self.ipTable)

    return nil
}

// Remove a VTEP port
func (self *OfnetAgent) RemoveVtepPort(portNo uint32) error {
    // FIXME:
    return nil
}
// Add a vlan.
// This is mainly used for mapping vlan id to Vxlan VNI
func (self *OfnetAgent) AddVlan(vlanId uint16, vni uint32) error {
    return nil
}

// Add remote route RPC call from master
func (self *OfnetAgent) RouteAdd(route *OfnetRoute, ret *bool) error {
    log.Infof("RouteAdd rpc call for route: %+v", route)

    // If this is a local route we are done
    if (route.OriginatorIp.String() == self.localIp.String()) {
        return nil
    }

    // First, add the route to local routing table
    self.routeTable[route.IpAddr.String()] = route

    // Lookup the VTEP for the route
    vtepPort := self.vtepTable[route.OriginatorIp.String()]
    if (vtepPort == nil) {
        log.Errorf("Could not find the VTEP for route: %+v", route)

        return errors.New("VTEP not found")
    }

    // Install the route in OVS

    // Create an output port for the vtep
    outPort, _ := self.ofSwitch.NewOutputPort(*vtepPort)

    // Install the IP address
    ipFlow, _ := self.ipTable.NewFlow(ofctrl.FlowMatch{
                            Priority: FLOW_MATCH_PRIORITY,
                            Ethertype: 0x0800,
                            IpDa: &route.IpAddr,
                        })
    ipFlow.SetMacDa(self.myRouterMac)
    // FIXME: set VNI
    // This is strictly not required at the source OVS. Source mac will be
    // overwritten by the dest OVS anyway. We keep the source mac for debugging purposes..
    // ipFlow.SetMacSa(self.myRouterMac)
    ipFlow.SetTunnelId(1)   // FIXME: hardcode VNI for now
    ipFlow.Next(outPort)

    return nil
}

// Delete remote route RPC call from master
func (self *OfnetAgent) RouteDel(route *OfnetRoute, ret *bool) error {
    return nil
}

// Add a local route to routing table and distribute it
func (self *OfnetAgent) localRouteAdd(route *OfnetRoute) error {
    // First, add the route to local routing table
    self.routeTable[route.IpAddr.String()] = route

    // Send the route to all known masters
    for masterAddr, _ := range self.masterDb {
        var resp bool

        log.Infof("Sending route %+v to master %s", route, masterAddr)

        // Make the RPC call to add the route to master
        err := rpcHub.Client(masterAddr, 9001).Call("OfnetMaster.RouteAdd", route, &resp)
        if (err != nil) {
            log.Errorf("Failed to add route %+v to master %s. Err: %v", route, masterAddr, err)
            return err
        }
    }

    return nil
}

const VLAN_TBL_ID = 1
const IP_TBL_ID = 2

// initialize Fgraph on the switch
func (self *OfnetAgent) initFgraph() error {
    sw := self.ofSwitch

    // Create all tables
    self.inputTable = sw.DefaultTable()
    self.vlanTable, _ = sw.NewTable(VLAN_TBL_ID)
    self.ipTable, _ = sw.NewTable(IP_TBL_ID)

    //Create all drop entries
    // Drop mcast source mac
    bcastMac, _ := net.ParseMAC("01:00:00:00:00:00")
    bcastSrcFlow, _ := self.inputTable.NewFlow(ofctrl.FlowMatch{
                            Priority: FLOW_MATCH_PRIORITY,
                            MacSa: &bcastMac,
                            MacSaMask: &bcastMac,
                        })
    bcastSrcFlow.Next(sw.DropAction())

    // Redirect ARP packets to controller
    arpFlow, _ := self.inputTable.NewFlow(ofctrl.FlowMatch{
                            Priority: FLOW_MATCH_PRIORITY,
                            Ethertype: 0x0806,
                        })
    arpFlow.Next(sw.SendToController())

    // Send all valid packets to vlan table
    // This is installed at lower priority so that all packets that miss above
    // flows will match entry
    validPktFlow, _ := self.inputTable.NewFlow(ofctrl.FlowMatch{
                            Priority: FLOW_MISS_PRIORITY,
                        })
    validPktFlow.Next(self.vlanTable)

    // Drop all packets that miss Vlan lookup
    vlanMissFlow, _ := self.vlanTable.NewFlow(ofctrl.FlowMatch{
                            Priority: FLOW_MISS_PRIORITY,
                        })
    vlanMissFlow.Next(sw.DropAction())

    // Drop all packets that miss IP lookup
    ipMissFlow, _ := self.ipTable.NewFlow(ofctrl.FlowMatch{
                            Priority: FLOW_MISS_PRIORITY,
                        })
    ipMissFlow.Next(sw.DropAction())

    return nil
}

// Process incoming packet
func (self *OfnetAgent) processArp(pkt protocol.Ethernet, inPort uint32) {
    log.Debugf("processing ARP packet on port %d", inPort)
    switch t := pkt.Data.(type) {
    case *protocol.ARP:
        log.Debugf("ARP packet: %+v", *t)
        var arpHdr protocol.ARP = *t

        switch arpHdr.Operation {
        case protocol.Type_Request:
            // FIXME: Send an ARP response only we have a route

            // Form an ARP response
            arpResp, _ := protocol.NewARP(protocol.Type_Reply)
            arpResp.HWSrc = self.myRouterMac
            arpResp.IPSrc = arpHdr.IPDst
            arpResp.HWDst = arpHdr.HWSrc
            arpResp.IPDst = arpHdr.IPSrc

            log.Infof("Sending ARP response: %+v", arpResp)

            // build the ethernet packet
            ethPkt := protocol.NewEthernet()
            ethPkt.HWDst = arpResp.HWDst
            ethPkt.HWSrc = arpResp.HWSrc
            ethPkt.Ethertype = 0x0806
            ethPkt.Data = arpResp

            log.Infof("Sending ARP response Ethernet: %+v", ethPkt)

            // Packet out
            pktOut := openflow13.NewPacketOut()
            pktOut.Data = ethPkt
            pktOut.AddAction(openflow13.NewActionOutput(inPort))

            log.Infof("Sending ARP response packet: %+v", pktOut)

            // Send it out
            self.ofSwitch.Send(pktOut)
        default:
            log.Infof("Dropping ARP response packet from port %d", inPort)
        }
    }
}
