package ofnet
// This package implements openflow network manager

import (
    "net"
    "pkg/ofnet/ofctrl"
)

// Interface implemented by each datapath
type OfnetDatapath interface {
    // Switch connected notification
    SwitchConnected(sw *ofctrl.OFSwitch)

    // Switch disconnected notification
    SwitchDisconnected(sw *ofctrl.OFSwitch)

    // Process Incoming packet
    PacketRcvd(sw *ofctrl.OFSwitch, pkt *ofctrl.PacketIn)

    // Add a local endpoint to forwarding DB
    AddLocalEndpoint(portNo uint32, macAddr net.HardwareAddr, vlan uint16, ipAddr net.IP) error

    // Remove a local endpoint from forwarding DB
    RemoveLocalEndpoint(portNo uint32) error

    // Add an remote VTEP
    AddVtepPort(portNo uint32, remoteIp net.IP) error

    // Remove remote VTEP
    RemoveVtepPort(portNo uint32, remoteIp net.IP) error

    // Add a vlan
    AddVlan(vlanId uint16, vni uint32) error

    // Remove a vlan
    RemoveVlan(vlanId uint16, vni uint32) error
}
