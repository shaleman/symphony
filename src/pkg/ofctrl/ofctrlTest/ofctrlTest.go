package main

import (
    "net"
    "log"
    "time"
    //"flag"

    "pkg/ofctrl/ofpxx"
    "pkg/ofctrl/ofp10"
    "pkg/ofctrl"
)

type OfActor struct {
    dpid    net.HardwareAddr
}

func (o *OfActor) PacketIn(dpid net.HardwareAddr, packet *ofp10.PacketIn) {
    log.Println("Received packet: ", packet)
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

func (o *OfActor) StatsReply(dpid net.HardwareAddr, rep *ofp10.StatsReply) {
    log.Printf("Received Stats Reply: %+v", rep)
    for _, sts := range rep.Body {
        log.Printf("Stats body: %+v", sts)
    }
}

func (o *OfActor) GetConfigReply(dpid net.HardwareAddr, config *ofp10.SwitchConfig) {
        log.Printf("Received config reply: %+v", config)
}


var ofActor OfActor

// Send requests from the switch
func sendRequest() {
    // wait for 5sec
    <-time.After(time.Second * 10)

    // Build flow stats req
    stats :=new(ofp10.StatsRequest)
    stats.Header = ofpxx.NewOfp10Header()
    stats.Header.Type = ofp10.Type_StatsRequest
    stats.Type = ofp10.StatsType_Flow
    flowReq := ofp10.NewFlowStatsRequest()
    flowReq.TableId = 0
    flowReq.OutPort = ofp10.P_NONE
    stats.Body = flowReq
    stats.Header.Length = stats.Len()

    log.Printf("Sending stats req: %+v, body: %+v", stats, stats.Body)

    ofctrl.Switch(ofActor.dpid).Send(stats)

    <- time.After(time.Second * 5)

    cfgReq := ofp10.NewConfigRequest()

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
