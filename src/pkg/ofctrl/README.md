# Ofctrl

This library implements a simple Openflow1.3 controller

# Usage

var app OfApp

// Create a controller
ctrler := ofctrl.NewController(&app)

This creates a new controller and registers the app for event callbacks. The app needs to implement following interface to get callbacks when an openflow switch connects to the controller.


type AppInterface interface {
    // A Switch connected to the controller
    SwitchConnected(sw *OFSwitch)

    // Switch disconnected from the controller
    SwitchDisconnected(sw *OFSwitch)

    // Controller received a packet from the switch
    PacketRcvd(sw *OFSwitch, pkt *openflow13.PacketIn)
}

# Example app

type OfApp struct {
    Switch *ofctrl.OFSwitch
}

func (o *OfApp) PacketRcvd(sw *ofctrl.OFSwitch, packet *openflow13.PacketIn) {
    log.Printf("App: Received packet: %+v", packet)
}

func (o *OfApp) SwitchConnected(sw *ofctrl.OFSwitch) {
    log.Printf("App: Switch connected: %v", sw.DPID())

    // Store switch for later use
    o.Switch = sw
}

func (o *OfApp) SwitchDisconnected(sw *ofctrl.OFSwitch) {
    log.Printf("App: Switch connected: %v", sw.DPID())
}

# Forwarding Graph API
An app can install flow table entries into the Openflow switch by using forwarding graph API.


 Forwarding graph is local to each switch. It is roughly structured as follows

         +------------+
         | Controller |
         +------------+
                |
      +---------+---------+
      |                   |
 +----------+        +----------+
 | Switch 1 |        | Switch 2 |
 +----------+        +----------+
       |
       +--------------+---------------+
       |              |               |
       V              V
 +---------+      +---------+     +---------+
 | Table 1 |  +-->| Table 2 |  +->| Table 3 |
 +---------+  |   +---------+  |  +---------+
      |       |        |       |      |
 +---------+  |   +---------+  |  +--------+     +------+
 | Flow 1  +--+   | Flow 1  +--+  | Flow 1 +---->| Drop |
 +---------+      +---------+     +--------+     +------+
      |
 +---------+            +----------+
 | Flow 2  +----------->+ OutPut 1 |
 +---------+            +----------+
      |
 +---------+                 +----------+
 | Flow 3  +---------------->| Output 2 |
 +---------+                 +----------+
      |                            ^
 +---------+       +---------+     |      +----------+
 | Flow 4  +------>| Flood 1 +-----+----->| Output 3 |
 +---------+       +---------+     |      +----------+
      |                            |
 +---------+     +-----------+     |      +----------+
 | Flow 5  +---->| Multipath |     +----->| Output 4 |
 +---------+     +-----+-----+            +----------+
                       |
          +------------+-------------+
          |            |             |
    +----------+  +----------+  +----------+
    | Output 5 |  | Output 6 |  | Output 7 |
    +----------+  +----------+  +----------+


 Forwarding graph is made up of Fgraph elements. Currently there are three
 kinds of elements (i) Table (ii) Flow (iii) Output. In future we will support
 Two additional types (iv) Flood and (v) Multipath.
 - Each Switch has a set of Tables. Switch has a special DefaultTable where
   All packet lookups start.
 - Each Table contains list of Flows. Each Flow has a Match which determines
   which packets match the flow and a NextElem which it points to
 - A Flow can point to following elements
      (a) Table - This moves the forwarding lookup to specified table
      (b) Output - This causes the packet to be sent out
      (c) Flood  - This causes the packet to be flooded to list of ports
      (d) Multipath - This causes packet to be load balanced across set of
                      ports. This can be used for link aggregation and ECMP
 - There are three kinds of outputs
      (i) drop - which causes the packet to be dropped
      (ii) toController - sends the packet to controller
      (iii) port - sends the packet out of specified port
 - A flow can have additional actions like (i) Set Vlan tag (ii) Set metadata
   Which is used for setting VRF for a packet (iii) Set VNI/tunnel header etc

 ----------------------------------------------------------------
 Example usage:

 // Find the switch we want to operate on
 switch := app.Switch
 
 // Create all tables
 rxVlanTbl := switch.NewTable(1)
 macSaTable := switch.NewTable(2)
 macDaTable := switch.NewTable(3)
 ipTable := switch.NewTable(4)
 inpTable := switch.DefaultTable() // table 0. i.e starting table

 // Discard mcast source mac
 dscrdMcastSrc := inpTable.NewFlow(FlowMatch{
                                  &McastSrc: { 0x01, 0, 0, 0, 0, 0 }
                                  &McastSrcMask: { 0x01, 0, 0, 0, 0, 0 }
                                  }, 100)
 dscrdMcastSrc.Next(switch.DropAction())

 // All valid packets go to vlan table
 validInputPkt := inpTable.NewFlow(FlowMatch{}, 1)
 validInputPkt.Next(rxVlanTbl)

 // Set access vlan for port 1 and go to mac lookup
 tagPort := rxVlanTbl.NewFlow(FlowMatch{
                              InputPort: Port(1)
                              }, 100)
 tagPort.SetVlan(10)
 tagPort.Next(macSaTable)

 // Match on IP dest addr and forward to a port
 ipFlow := ipTable.NewFlow(FlowParams{
                           IpDa: &net.IPv4("10.10.10.10")
                          }, 100)

 outPort := switch.NewOutputPort(OutParams{
                              OutPort: Port(10)
                              }, 100)
 ipFlow.Next(outPort)
