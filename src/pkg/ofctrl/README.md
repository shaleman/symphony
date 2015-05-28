# Ofctrl

This library implements a simple Openflow1.3 controller

# Usage

var app OfApp

// Create a controller
ctrler := ofctrl.NewController(&app)

This create a new controller and registers the app for event callbacks. The app needs to implement following interface to get callbacks when an openflow switch connects to the controller.


type AppInterface interface {
    // A Switch connected to the controller
    SwitchConnected(sw *OFSwitch)

    // Switch disconnected from the controller
    SwitchDisconnected(sw *OFSwitch)

    // Controller received a packet from the switch
    PacketRcvd(sw *OFSwitch, pkt *openflow13.PacketIn)
}

# Example
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
