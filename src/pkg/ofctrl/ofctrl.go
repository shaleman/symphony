package ofctrl
// This library implements a simple openflow 1.3 controller

import (
    "net"
    "time"

    "pkg/ofctrl/libOpenflow/common"
    "pkg/ofctrl/libOpenflow/openflow10"
    "pkg/ofctrl/libOpenflow/openflow13"

    log "github.com/Sirupsen/logrus"
)

// Note: Command to make ovs connect to controller:
// ovs-vsctl set-controller <bridge-name> tcp:<ip-addr>:<port>
// E.g.    ovs-vsctl set-controller ovsbr0 tcp:127.0.0.1:6633

// To enable openflow1.3 support in OVS:
// ovs-vsctl set bridge <bridge-name> protocols=OpenFlow10,OpenFlow11,OpenFlow12,OpenFlow13
// E.g. sudo ovs-vsctl set bridge ovsbr0 protocols=OpenFlow10,OpenFlow11,OpenFlow12,OpenFlow13

type AppInterface interface {
    SwitchConnected(sw *OFSwitch)
    SwitchDisconnected(sw *OFSwitch)
    PacketRcvd(sw *OFSwitch, pkt *openflow13.PacketIn)
}

type Controller struct{
    app      AppInterface
}

// Create a new controller
func NewController(app AppInterface) *Controller {
    c := new(Controller)

    // Save the handler
    c.app = app

    return c
}

// Listen on a port
func (c *Controller) Listen(port string) {
    addr, _ := net.ResolveTCPAddr("tcp", port)

    sock, err := net.ListenTCP("tcp", addr)
    if err != nil {
        log.Fatal(err)
    }
    defer sock.Close()

    log.Println("Listening for connections on", addr)
    for {
        conn, err := sock.AcceptTCP()
        if err != nil {
            log.Fatal(err)
        }
        go c.handleConnection(conn)
    }
}

func (c *Controller) handleConnection(conn *net.TCPConn) {
    stream := NewMessageStream(conn)

    log.Println("New connection..")

    // Send ofp 1.3 Hello by default
    h, err := common.NewHello(4)
    if err != nil {
        return
    }
    stream.Outbound <- h

    for {
        select {
        // Send hello message with latest protocol version.
        case msg := <-stream.Inbound:
            switch m := msg.(type) {
            // A Hello message of the appropriate type
            // completes version negotiation. If version
            // types are incompatable, it is possible the
            // connection may be servered without error.
            case *common.Hello:
                if m.Version == openflow10.VERSION {
                    // Version negotiation is
                    // considered complete. Create
                    // new Switch and notifiy listening
                    // applications.
                    stream.Version = m.Version
                    stream.Outbound <- openflow10.NewFeaturesRequest()

                    log.Warnln("Received Openflow 1.0 Hello message")
                    log.Warnln("This controller requires openflow 1.3")

                } else if m.Version == openflow13.VERSION {
                    log.Infoln("Received Openflow 1.3 Hello message")

                    stream.Version = m.Version
                    stream.Outbound <- openflow13.NewFeaturesRequest()
                } else {
                    // Connection should be severed if controller
                    // doesn't support switch version.
                    log.Println("Received unsupported ofp version", m.Version)
                    stream.Shutdown <- true
                }
            case *openflow10.SwitchFeatures:
                log.Warnln("Received Openflow 1.3 feature response")
                log.Warnln("This controller requires openflow 1.3")

            // After a vaild FeaturesReply has been received we
            // have all the information we need. Create a new
            // switch object and notify applications.
            case *openflow13.SwitchFeatures:
                log.Printf("Received ofp1.3 Switch feature response: %+v", *m)

                // Create a new switch and handover the stream
                NewSwitch(stream, m.DPID, c.app)

                // Let switch instance handle all future messages..
                return

            // An error message may indicate a version mismatch. We
            // disconnect if an error occurs this early.
            case *openflow10.ErrorMsg:
                log.Warnln(m)
                stream.Version = m.Header.Version
                stream.Shutdown <- true
            case *openflow13.ErrorMsg:
                log.Warnf("Received ofp1.3 error msg: %+v", *m)
                stream.Shutdown <- true
            }
        case err := <-stream.Error:
            // The connection has been shutdown.
            log.Println(err)
            return
        case <-time.After(time.Second * 3):
            // This shouldn't happen. If it does, both the controller
            // and switch are no longer communicating. The TCPConn is
            // still established though.
            log.Warnln("Connection timed out.")
            return
        }
    }
}
