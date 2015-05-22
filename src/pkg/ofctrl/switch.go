package ofctrl

import (
    "log"
    "net"
    "time"
    // "sync"

    "pkg/ofctrl/ofp10"
    "pkg/ofctrl/ofpxx"
    "pkg/ofctrl/util"
)


type OFSwitch struct {
    stream      *MessageStream
    actors      []interface{}
    dpid        net.HardwareAddr
    ports       map[uint16]ofp10.PhyPort
}

var switchDb map[string]*OFSwitch = make(map[string]*OFSwitch)

// Builds and populates a Switch struct then starts listening
// for OpenFlow messages on conn.
func NewSwitch(stream *MessageStream, msg ofp10.SwitchFeatures) *OFSwitch {
    var s *OFSwitch

    if (switchDb[msg.DPID.String()] == nil) {
        log.Println("Openflow Connection for new switch:", msg.DPID)

        s = new(OFSwitch)
        s.stream = stream
        s.actors = *new([]interface{})
        s.dpid = msg.DPID

        // Add a default message handler for echo replies
        dfltActor := DefaultActor{}
        s.AddActor(&dfltActor)

        // Save it
        switchDb[msg.DPID.String()] = s

        // Main receive loop for the switch
        go s.receive()

    } else {
        log.Println("Openflow Connection for switch:", msg.DPID)

        s = switchDb[msg.DPID.String()]
        s.stream = stream
        s.dpid = msg.DPID
    }

    // Setup ports db
    s.ports = make(map[uint16]ofp10.PhyPort)
    for _, p := range msg.Ports {
        s.ports[p.PortNo] = p
    }

    // Send connection up callback
    for _, inst := range s.actors {
        if actor, ok := inst.(ofp10.ConnectionUpReactor); ok {
            actor.ConnectionUp(s.DPID())
        }
    }

    // Return the new switch
    return s
}

// Returns a pointer to the Switch mapped to dpid.
func Switch(dpid net.HardwareAddr) *OFSwitch {
    return switchDb[dpid.String()]
}

// Check if an actor already exists
func (sw *OFSwitch) actorExists(inst interface{}) bool {
    for _, actr := range sw.actors {
        if actr == inst {
            return true
        }
    }
    return false
}

// Add a message handler
func (sw *OFSwitch) AddActor(inst interface{}) {
    if (!sw.actorExists(inst)) {
        sw.actors = append(sw.actors, inst)
    }

    // FIXME: find a better place to send this callback
    // Send connection up callback
    if actor, ok := inst.(ofp10.ConnectionUpReactor); ok {
        actor.ConnectionUp(sw.DPID())
    }
}


// Returns the dpid of Switch s.
func (s *OFSwitch) DPID() net.HardwareAddr {
    return s.dpid
}


// Sends an OpenFlow message to this Switch.
func (s *OFSwitch) Send(req util.Message) {
    s.stream.Outbound <- req
}

// Receive loop for each Switch.
func (s *OFSwitch) receive() {
    for {
        select {
        case msg := <-s.stream.Inbound:
            // New message has been received from message
            // stream.
            s.distributeMessages(s.dpid, msg)
        case err := <-s.stream.Error:
            // Message stream has been disconnected.
            for _, app := range s.actors {
                if actor, ok := app.(ofp10.ConnectionDownReactor); ok {
                    actor.ConnectionDown(s.DPID(), err)
                }
            }
            return
        }
    }
}

func (sw *OFSwitch) distributeMessages(dpid net.HardwareAddr, msg util.Message) {
    log.Printf("Received message: %+v, on switch: %s", msg, dpid.String())

    for _, app := range sw.actors {
        switch t := msg.(type) {
        case *ofpxx.Header:
            switch t.Header().Type {
            case ofp10.Type_Hello:
                if actor, ok := app.(ofp10.HelloReactor); ok {
                    actor.Hello(t)
                }
            case ofp10.Type_EchoRequest:
                if actor, ok := app.(ofp10.EchoRequestReactor); ok {
                    actor.EchoRequest(sw.DPID())
                }
            case ofp10.Type_EchoReply:
                if actor, ok := app.(ofp10.EchoReplyReactor); ok {
                    actor.EchoReply(sw.DPID())
                }
            case ofp10.Type_FeaturesRequest:
                if actor, ok := app.(ofp10.FeaturesRequestReactor); ok {
                    actor.FeaturesRequest(t)
                }
            case ofp10.Type_GetConfigRequest:
                if actor, ok := app.(ofp10.GetConfigRequestReactor); ok {
                    actor.GetConfigRequest(t)
                }
            case ofp10.Type_BarrierRequest:
                if actor, ok := app.(ofp10.BarrierRequestReactor); ok {
                    actor.BarrierRequest(t)
                }
            case ofp10.Type_BarrierReply:
                if actor, ok := app.(ofp10.BarrierReplyReactor); ok {
                    actor.BarrierReply(sw.DPID(), t)
                }
            }
        case *ofp10.ErrorMsg:
            if actor, ok := app.(ofp10.ErrorReactor); ok {
                actor.Error(sw.DPID(), t)
            }
        case *ofp10.VendorHeader:
            if actor, ok := app.(ofp10.VendorReactor); ok {
                actor.VendorHeader(sw.DPID(), t)
            }
        case *ofp10.SwitchFeatures:
            if actor, ok := app.(ofp10.FeaturesReplyReactor); ok {
                actor.FeaturesReply(sw.DPID(), t)
            }
        case *ofp10.SwitchConfig:
            switch t.Header.Type {
            case ofp10.Type_GetConfigReply:
                if actor, ok := app.(ofp10.GetConfigReplyReactor); ok {
                    actor.GetConfigReply(sw.DPID(), t)
                }
            case ofp10.Type_SetConfig:
                if actor, ok := app.(ofp10.SetConfigReactor); ok {
                    actor.SetConfig(t)
                }
            }
        case *ofp10.PacketIn:
            if actor, ok := app.(ofp10.PacketInReactor); ok {
                actor.PacketIn(sw.DPID(), t)
            }
        case *ofp10.FlowRemoved:
            if actor, ok := app.(ofp10.FlowRemovedReactor); ok {
                actor.FlowRemoved(sw.DPID(), t)
            }
        case *ofp10.PortStatus:
            if actor, ok := app.(ofp10.PortStatusReactor); ok {
                actor.PortStatus(sw.DPID(), t)
            }
        case *ofp10.PacketOut:
            if actor, ok := app.(ofp10.PacketOutReactor); ok {
                actor.PacketOut(t)
            }
        case *ofp10.FlowMod:
            if actor, ok := app.(ofp10.FlowModReactor); ok {
                actor.FlowMod(t)
            }
        case *ofp10.PortMod:
            if actor, ok := app.(ofp10.PortModReactor); ok {
                actor.PortMod(t)
            }
        case *ofp10.StatsRequest:
            if actor, ok := app.(ofp10.StatsRequestReactor); ok {
                actor.StatsRequest(t)
            }
        case *ofp10.StatsReply:
            if actor, ok := app.(ofp10.StatsReplyReactor); ok {
                actor.StatsReply(sw.DPID(), t)
            }
        }
    }
}

// Default openflow message handler
type DefaultActor struct {}

func (o *DefaultActor) ConnectionUp(dpid net.HardwareAddr) {
    log.Println("Switch connected:", dpid)

    dropMod := ofp10.NewFlowMod()
    dropMod.Priority = 1

    arpFmod := ofp10.NewFlowMod()
    arpFmod.Priority = 2
    arpFmod.Match.DLType = 0x0806 // ARP Messages
    arpFmod.AddAction(ofp10.NewActionOutput(ofp10.P_CONTROLLER))

    sw := Switch(dpid)
    if (sw != nil)  {
        sw.Send(ofp10.NewFeaturesRequest())
        sw.Send(dropMod)
        sw.Send(arpFmod)
        sw.Send(ofp10.NewEchoRequest())
    }
}

func (o *DefaultActor) ConnectionDown(dpid net.HardwareAddr) {
    log.Println("Switch Disconnected:", dpid)
}

func (o *DefaultActor) EchoRequest(dpid net.HardwareAddr) {
    // Wait three seconds then send an echo_reply message.
    go func() {
        <-time.After(time.Second * 3)

        // Send echo reply
        res := ofp10.NewEchoReply()
        Switch(dpid).Send(res)
    }()
}

func (o *DefaultActor) EchoReply(dpid net.HardwareAddr) {
    // Wait three seconds then send an echo_request message.
    go func() {
        <-time.After(time.Second * 3)

        // Send echo request
        res := ofp10.NewEchoRequest()
        Switch(dpid).Send(res)
    }()
}
