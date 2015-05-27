package ofctrl

import (
    "net"
    "time"

    "pkg/ofctrl/libOpenflow/openflow13"
    "pkg/ofctrl/libOpenflow/common"
    "pkg/ofctrl/libOpenflow/util"

    log "github.com/Sirupsen/logrus"
)


type OFSwitch struct {
    stream      *MessageStream
    actors      []interface{}
    dpid        net.HardwareAddr
    app         AppInterface
    // Following are fgraph state for the switch
    tableDb         map[uint8]*Table
    dropAction      *Output
    sendToCtrler    *Output
}

var switchDb map[string]*OFSwitch = make(map[string]*OFSwitch)

// Builds and populates a Switch struct then starts listening
// for OpenFlow messages on conn.
func NewSwitch(stream *MessageStream, dpid net.HardwareAddr, app AppInterface) *OFSwitch {
    var s *OFSwitch

    if (switchDb[dpid.String()] == nil) {
        log.Infoln("Openflow Connection for new switch:", dpid)

        s = new(OFSwitch)
        s.app = app
        s.stream = stream
        s.actors = *new([]interface{})
        s.dpid = dpid

        // Initialize the fgraph elements
        s.initFgraph()

        // Add a default message handler for echo replies
        dfltActor := DefaultActor{}
        s.AddActor(&dfltActor)

        // Save it
        switchDb[dpid.String()] = s

        // Main receive loop for the switch
        go s.receive()

    } else {
        log.Infoln("Openflow Connection for switch:", dpid)

        s = switchDb[dpid.String()]
        s.stream = stream
        s.dpid = dpid
    }

    // send Switch connected callback
    app.SwitchConnected(s)

    // Send connection up callback to registered actors
    for _, inst := range s.actors {
        if actor, ok := inst.(openflow13.ConnectionUpReactor); ok {
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
func (self *OFSwitch) actorExists(inst interface{}) bool {
    for _, actr := range self.actors {
        if actr == inst {
            return true
        }
    }
    return false
}

// Add a message handler
func (self *OFSwitch) AddActor(inst interface{}) {
    if (!self.actorExists(inst)) {
        self.actors = append(self.actors, inst)
    }
}


// Returns the dpid of Switch s.
func (self *OFSwitch) DPID() net.HardwareAddr {
    return self.dpid
}


// Sends an OpenFlow message to this Switch.
func (self *OFSwitch) Send(req util.Message) {
    self.stream.Outbound <- req
}

// Receive loop for each Switch.
func (self *OFSwitch) receive() {
    for {
        select {
        case msg := <-self.stream.Inbound:
            // New message has been received from message
            // stream.
            self.distributeMessages(self.dpid, msg)
        case err := <-self.stream.Error:
            // send Switch disconnected callback
            self.app.SwitchDisconnected(self)

            // Message stream has been disconnected.
            for _, app := range self.actors {
                if actor, ok := app.(openflow13.ConnectionDownReactor); ok {
                    actor.ConnectionDown(self.DPID(), err)
                }
            }
            return
        }
    }
}

func (self *OFSwitch) distributeMessages(dpid net.HardwareAddr, msg util.Message) {
    log.Debugf("Received message: %+v, on switch: %s", msg, dpid.String())

    for _, app := range self.actors {
        switch t := msg.(type) {
        case *common.Header:
            switch t.Header().Type {
            case openflow13.Type_Hello:
                if actor, ok := app.(openflow13.HelloReactor); ok {
                    actor.Hello(t)
                }
            case openflow13.Type_EchoRequest:
                if actor, ok := app.(openflow13.EchoRequestReactor); ok {
                    actor.EchoRequest(self.DPID())
                }
            case openflow13.Type_EchoReply:
                if actor, ok := app.(openflow13.EchoReplyReactor); ok {
                    actor.EchoReply(self.DPID())
                }
            case openflow13.Type_FeaturesRequest:
                if actor, ok := app.(openflow13.FeaturesRequestReactor); ok {
                    actor.FeaturesRequest(t)
                }
            case openflow13.Type_GetConfigRequest:
                if actor, ok := app.(openflow13.GetConfigRequestReactor); ok {
                    actor.GetConfigRequest(t)
                }
            case openflow13.Type_BarrierRequest:
                if actor, ok := app.(openflow13.BarrierRequestReactor); ok {
                    actor.BarrierRequest(t)
                }
            case openflow13.Type_BarrierReply:
                if actor, ok := app.(openflow13.BarrierReplyReactor); ok {
                    actor.BarrierReply(self.DPID(), t)
                }
            }
        case *openflow13.ErrorMsg:
            if actor, ok := app.(openflow13.ErrorReactor); ok {
                actor.Error(self.DPID(), t)
            }
        case *openflow13.VendorHeader:
            if actor, ok := app.(openflow13.VendorReactor); ok {
                actor.VendorHeader(self.DPID(), t)
            }
        case *openflow13.SwitchFeatures:
            if actor, ok := app.(openflow13.FeaturesReplyReactor); ok {
                actor.FeaturesReply(self.DPID(), t)
            }
        case *openflow13.SwitchConfig:
            switch t.Header.Type {
            case openflow13.Type_GetConfigReply:
                if actor, ok := app.(openflow13.GetConfigReplyReactor); ok {
                    actor.GetConfigReply(self.DPID(), t)
                }
            case openflow13.Type_SetConfig:
                if actor, ok := app.(openflow13.SetConfigReactor); ok {
                    actor.SetConfig(t)
                }
            }
        case *openflow13.PacketIn:
            // send packet rcvd callback
            self.app.PacketRcvd(self, t)

            // Send it to all registered actors too
            if actor, ok := app.(openflow13.PacketInReactor); ok {
                actor.PacketIn(self.DPID(), t)
            }
        case *openflow13.FlowRemoved:
            if actor, ok := app.(openflow13.FlowRemovedReactor); ok {
                actor.FlowRemoved(self.DPID(), t)
            }
        case *openflow13.PortStatus:
            if actor, ok := app.(openflow13.PortStatusReactor); ok {
                actor.PortStatus(self.DPID(), t)
            }
        case *openflow13.PacketOut:
            if actor, ok := app.(openflow13.PacketOutReactor); ok {
                actor.PacketOut(t)
            }
        case *openflow13.FlowMod:
            if actor, ok := app.(openflow13.FlowModReactor); ok {
                actor.FlowMod(t)
            }
        case *openflow13.PortMod:
            if actor, ok := app.(openflow13.PortModReactor); ok {
                actor.PortMod(t)
            }
        case *openflow13.MultipartRequest:
            if actor, ok := app.(openflow13.MultipartRequestReactor); ok {
                actor.MultipartRequest(t)
            }
        case *openflow13.MultipartReply:
            if actor, ok := app.(openflow13.MultipartReplyReactor); ok {
                actor.MultipartReply(self.DPID(), t)
            }
        }
    }
}

// Default openflow message handler
type DefaultActor struct {}

func (o *DefaultActor) ConnectionUp(dpid net.HardwareAddr) {
    log.Println("DefaultActor: Switch connected:", dpid)

    sw := Switch(dpid)
    if (sw != nil)  {
        sw.Send(openflow13.NewFeaturesRequest())
        sw.Send(openflow13.NewEchoRequest())
    }
}

func (o *DefaultActor) ConnectionDown(dpid net.HardwareAddr) {
    log.Println("DefaultActor: Switch Disconnected:", dpid)
}

func (o *DefaultActor) EchoRequest(dpid net.HardwareAddr) {
    // Wait three seconds then send an echo_reply message.
    go func() {
        <-time.After(time.Second * 3)

        // Send echo reply
        res := openflow13.NewEchoReply()
        Switch(dpid).Send(res)
    }()
}

func (o *DefaultActor) EchoReply(dpid net.HardwareAddr) {
    // Wait three seconds then send an echo_request message.
    go func() {
        <-time.After(time.Second * 3)

        // Send echo request
        res := openflow13.NewEchoRequest()
        Switch(dpid).Send(res)
    }()
}
