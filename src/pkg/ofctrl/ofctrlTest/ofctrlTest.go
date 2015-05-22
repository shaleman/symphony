package main

import (
    "net"
    "log"
    "time"
    //"flag"

    "pkg/ofctrl/ofpxx"
    "pkg/ofctrl/ofp13"
    "pkg/ofctrl"
)

type OfActor struct {
    dpid    net.HardwareAddr
}

func (o *OfActor) PacketIn(dpid net.HardwareAddr, packet *ofp13.PacketIn) {
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

func (o *OfActor) MultipartReply(dpid net.HardwareAddr, rep *ofp13.MultipartReply) {
    log.Printf("Received Stats Reply: %+v", rep)
    for _, sts := range rep.Body {
        log.Printf("Stats body: %+v", sts)
    }
}

func (o *OfActor) GetConfigReply(dpid net.HardwareAddr, config *ofp13.SwitchConfig) {
        log.Printf("Received config reply: %+v", config)
}


var ofActor OfActor

// Send requests from the switch
func sendRequest() {
    // wait for 10sec to make sure we are connected
    <-time.After(time.Second * 10)
    // Send flowmod request
    dropMod := ofp13.NewFlowMod()
    dropMod.Priority = 1

    log.Printf("Sending DropMod: %+v", dropMod)
    ofctrl.Switch(ofActor.dpid).Send(dropMod)

    arpMod := ofp13.NewFlowMod()
    arpMod.Priority = 2
    etypeField := ofp13.NewEthTypeField(0x806)
    arpMatch := ofp13.NewMatch()
    arpMatch.AddField(*etypeField)
    arpMod.Match = *arpMatch
    ctrlAct := ofp13.NewActionOutput(ofp13.P_CONTROLLER)
    ctrlInstr := ofp13.NewInstrApplyActions()
    ctrlInstr.AddAction(ctrlAct)
    arpMod.AddInstruction(ctrlInstr)

    log.Printf("Sending ArpMod: %+v, instr: %+v", arpMod, arpMod.Instructions[0])
    ofctrl.Switch(ofActor.dpid).Send(arpMod)

    <-time.After(time.Second * 3)

    // Build flow stats req
    stats :=new(ofp13.MultipartRequest)
    stats.Header = ofpxx.NewOfp13Header()
    stats.Header.Type = ofp13.Type_MultiPartRequest
    stats.Type = ofp13.MultipartType_Flow
    flowReq := ofp13.NewFlowStatsRequest()
    flowReq.TableId = ofp13.OFPTT_ALL
    flowReq.OutPort = ofp13.P_ANY
    flowReq.OutGroup = ofp13.OFPG_ANY
    stats.Body = flowReq
    stats.Header.Length = stats.Len()

    log.Printf("Sending stats req: %+v, body: %+v", stats, stats.Body)

    ofctrl.Switch(ofActor.dpid).Send(stats)

    <- time.After(time.Second * 5)

    cfgReq := ofp13.NewConfigRequest()

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
