package main

import (
    "net"
    "log"
    "time"
    //"flag"

    "pkg/ofctrl/libOpenflow/common"
    "pkg/ofctrl/libOpenflow/openflow13"
    "pkg/ofctrl"
)

type OfActor struct {
    dpid    net.HardwareAddr
}

func (o *OfActor) PacketIn(dpid net.HardwareAddr, packet *openflow13.PacketIn) {
    log.Printf("Received packet: %+v", packet)
}

func (o *OfActor) EchoRequest(dpid net.HardwareAddr) {
    log.Println("Received echo request")
}

func (o *OfActor) EchoReply(dpid net.HardwareAddr) {
    log.Println("Received echo reply")
}

func (o *OfActor) ConnectionUp(dpid net.HardwareAddr) {
    log.Println("Actor: Switch connected:", dpid)
    o.dpid = dpid
}

func (o *OfActor) MultipartReply(dpid net.HardwareAddr, rep *openflow13.MultipartReply) {
    log.Printf("Received Stats Reply: %+v", rep)
    for _, sts := range rep.Body {
        log.Printf("Stats body: %+v", sts)
    }
}

func (o *OfActor) GetConfigReply(dpid net.HardwareAddr, config *openflow13.SwitchConfig) {
        log.Printf("Received config reply: %+v", config)
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
    ofctrl.Switch(ofActor.dpid).Send(dropMod)

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
    ofctrl.Switch(ofActor.dpid).Send(arpMod)

    <-time.After(time.Second * 3)

    // Build flow stats req
    stats :=new(openflow13.MultipartRequest)
    stats.Header = common.NewOfp13Header()
    stats.Header.Type = openflow13.Type_MultiPartRequest
    stats.Type = openflow13.MultipartType_Flow
    flowReq := openflow13.NewFlowStatsRequest()
    flowReq.TableId = openflow13.OFPTT_ALL
    flowReq.OutPort = openflow13.P_ANY
    flowReq.OutGroup = openflow13.OFPG_ANY
    stats.Body = flowReq
    stats.Header.Length = stats.Len()

    log.Printf("Sending stats req: %+v, body: %+v", stats, stats.Body)

    ofctrl.Switch(ofActor.dpid).Send(stats)

    <- time.After(time.Second * 5)

    cfgReq := openflow13.NewConfigRequest()

    log.Printf("Sending config req: %+v", cfgReq)
    ofctrl.Switch(ofActor.dpid).Send(cfgReq)

}

func main() {
    // Hack to log output
    // flag.Lookup("logtostderr").Value.Set("true")


    // Create a controller
    ctrler := ofctrl.NewController(&ofActor)

    go sendRequest()

    // start listening
    ctrler.Listen(":6633")
}
