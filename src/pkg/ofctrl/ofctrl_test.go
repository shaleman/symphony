package ofctrl

import (
    "net"
    "log"
    "testing"
    //"flag"

    "pkg/ofctrl/ofp10"
)

type OfActor struct {}

func (o *OfActor) PacketIn(dpid net.HardwareAddr, packet *ofp10.PacketIn) {
    log.Println("Received packet: ", packet)
}

func (o *OfActor) EchoRequest(dpid net.HardwareAddr) {
    log.Println("Received echo request")
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
