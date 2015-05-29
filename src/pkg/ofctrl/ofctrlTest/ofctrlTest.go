package main

import (
    "net"
    "time"
    //"flag"

    "pkg/ofctrl/libOpenflow/openflow13"
    "pkg/ofctrl"

    log "github.com/Sirupsen/logrus"
)

type OfActor struct {
    Switch *ofctrl.OFSwitch
}

func (o *OfActor) PacketRcvd(sw *ofctrl.OFSwitch, packet *openflow13.PacketIn) {
    log.Printf("App: Received packet: %+v", packet)
}

func (o *OfActor) SwitchConnected(sw *ofctrl.OFSwitch) {
    log.Printf("App: Switch connected: %v", sw.DPID())

    // Store switch for later use
    o.Switch = sw
}

func (o *OfActor) SwitchDisconnected(sw *ofctrl.OFSwitch) {
    log.Printf("App: Switch connected: %v", sw.DPID())
}


var ofActor OfActor

// Send requests from the switch
func sendRequest() {
    // wait for 10sec to make sure we are connected
    <-time.After(time.Second * 10)
    // Send flowmod request
    dropMod := openflow13.NewFlowMod()
    dropMod.Priority = 1

    log.Printf("Sending DropMod: %+v", dropMod)
    ofActor.Switch.Send(dropMod)

    arpMod := openflow13.NewFlowMod()
    arpMod.Priority = 2
    etypeField := openflow13.NewEthTypeField(0x806)
    arpMatch := openflow13.NewMatch()
    arpMatch.AddField(*etypeField)
    arpMod.Match = *arpMatch
    ctrlAct := openflow13.NewActionOutput(openflow13.P_CONTROLLER)
    ctrlInstr := openflow13.NewInstrApplyActions()
    ctrlInstr.AddAction(ctrlAct)
    arpMod.AddInstruction(ctrlInstr)

    log.Printf("Sending ArpMod: %+v, instr: %+v", arpMod, arpMod.Instructions[0])
    ofActor.Switch.Send(arpMod)

    <-time.After(time.Second * 3)

    // Build flow stats req
    stats :=new(openflow13.MultipartRequest)
    stats.Header = openflow13.NewOfp13Header()
    stats.Header.Type = openflow13.Type_MultiPartRequest
    stats.Type = openflow13.MultipartType_Flow
    flowReq := openflow13.NewFlowStatsRequest()
    flowReq.TableId = openflow13.OFPTT_ALL
    flowReq.OutPort = openflow13.P_ANY
    flowReq.OutGroup = openflow13.OFPG_ANY
    stats.Body = flowReq
    stats.Header.Length = stats.Len()

    log.Printf("Sending stats req: %+v, body: %+v", stats, stats.Body)

    ofActor.Switch.Send(stats)

    <- time.After(time.Second * 5)

    cfgReq := openflow13.NewConfigRequest()

    log.Printf("Sending config req: %+v", cfgReq)
    ofActor.Switch.Send(cfgReq)

}

func fgraphTest() {
    // wait for 10sec to make sure we are connected
    <-time.After(time.Second * 10)

    log.Printf("Adding flow table...")

    sw := ofActor.Switch

    vlanTbl, _ := sw.NewTable(1)
    initTbl := sw.DefaultTable()
    ipTbl, _ := sw.NewTable(2)

    log.Printf("Adding bcast mac SA")
    // Drop bcast source mac
    bcastMac, _ := net.ParseMAC("ff:ff:ff:ff:ff:ff")
    bcastSrc, _ := initTbl.NewFlow(ofctrl.FlowMatch{
                                Priority: 100,
                                MacSa: &bcastMac,
                            })
    bcastSrc.Next(sw.DropAction())
    <-time.After(time.Second * 2)

    log.Printf("Adding valid pkt flow")
    // valid pkts go to next table
    validPkt, _ := initTbl.NewFlow(ofctrl.FlowMatch{ Priority: 1 })
    validPkt.Next(vlanTbl)
    <-time.After(time.Second * 2)

    log.Printf("Adding set vlan")

    // Add vlan id based on port num
    port2Vlan, _ := vlanTbl.NewFlow(ofctrl.FlowMatch{
                                Priority: 100,
                                InputPort: 2,
                                })
    port2Vlan.SetVlan(9)
    port2Vlan.Next(ipTbl)
    <-time.After(time.Second * 2)

    log.Printf("Adding IP lookup")

    // IP DA lookup
    ipDa := net.ParseIP("10.10.10.10")
    ipFlow, _ := ipTbl.NewFlow(ofctrl.FlowMatch{
                        Priority: 100,
                        Ethertype: 0x0800,
                        IpDa: &ipDa,
                    })
    port3, _ := sw.NewOutputPort(3)
    ipFlow.Next(port3)

    log.Printf("Finished installing flow tables")
}
func main() {
    // for debug logs
    log.SetLevel(log.DebugLevel)

    // Create a controller
    ctrler := ofctrl.NewController(&ofActor)

    // go sendRequest()
    go fgraphTest()

    // start listening
    ctrler.Listen(":6633")
}
