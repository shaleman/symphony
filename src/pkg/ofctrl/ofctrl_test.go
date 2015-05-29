package ofctrl

import (
    "net"
    "log"
    "testing"
    //"flag"

    "pkg/ofctrl/libOpenflow/openflow13"
)

type OfActor struct {}

func (o *OfActor) PacketRcvd(dpid net.HardwareAddr, packet *openflow13.PacketIn) {
    log.Println("Received packet: ", packet)
}


var ofActor OfActor

func TestOfctrlInit(t *testing.T) {
    // Hack to log output
    // flag.Lookup("logtostderr").Value.Set("true")


    // Create a controller
    ctrler := NewController(&ofActor)

    // start listening
    ctrler.Listen(":6633")
}
