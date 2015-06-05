package ofctrl

import (
    "log"
    "testing"

    "github.com/shaleman/libOpenflow/openflow13"
)

type OfActor struct {
    Switch *OFSwitch
}

func (o *OfActor) PacketRcvd(sw *OFSwitch, packet *openflow13.PacketIn) {
    log.Printf("App: Received packet: %+v", packet)
}

func (o *OfActor) SwitchConnected(sw *OFSwitch) {
    log.Printf("App: Switch connected: %v", sw.DPID())

    // Store switch for later use
    o.Switch = sw
}

func (o *OfActor) SwitchDisconnected(sw *OFSwitch) {
    log.Printf("App: Switch connected: %v", sw.DPID())
}


var ofActor OfActor


func TestOfctrlInit(t *testing.T) {
    // Create a controller
    ctrler := NewController("ovsbr0", &ofActor)

    // start listening
    ctrler.Listen(":6633")
}

/* This was just an experiment
// Test connecting over unix socket
func TestUnixSocket(t *testing.T) {
    // Create a controller
    ctrler := NewController("ovsbr0", &ofActor)

    // Connect to unix socket
    conn, err := net.Dial("unix", "/var/run/openvswitch/ovsbr0.mgmt")
    if (err != nil) {
        log.Printf("Failed to connect to unix socket. Err: %v", err)
        t.Errorf("Failed to connect to unix socket. Err: %v", err)
        return
    }

    // Handle connection
    ctrler.handleConnection(conn)

    time.After(100 * time.Second)
}
*/
