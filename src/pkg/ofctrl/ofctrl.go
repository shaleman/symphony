package ofctrl
// This library implements a simple openflow controller

import (
    "log"
    "net"
    "time"

    "pkg/ofctrl/ofpxx"
    "pkg/ofctrl/ofp10"
    "pkg/ofctrl/ofp13"
)

// Note: Command to make ovs connect to controller:
// ovs-vsctl set-controller <bridge-name> tcp:<ip-addr>:<port>
// E.g.    ovs-vsctl set-controller ovsbr0 tcp:127.0.0.1:6633

// To enable openflow1.3 support in OVS:
// ovs-vsctl set bridge <bridge-name> protocols=OpenFlow10,OpenFlow11,OpenFlow12,OpenFlow13
// E.g. sudo ovs-vsctl set bridge ovsbr0 protocols=OpenFlow10,OpenFlow11,OpenFlow12,OpenFlow13

type Controller struct{
    actor      interface{}
}

// Create a new controller
func NewController(actor interface{}) *Controller {
    c := new(Controller)

    // Save the handler
    c.actor = actor

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

    // Send ofp 1.3 Hello
    h, err := ofpxx.NewHello(4)
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
            case *ofpxx.Header:
                if m.Version == ofp10.VERSION {
                    // Version negotiation is
                    // considered complete. Create
                    // new Switch and notifiy listening
                    // applications.
                    stream.Version = m.Version
                    stream.Outbound <- ofp10.NewFeaturesRequest()
                } else if m.Version == ofp13.VERSION {
                    stream.Version = m.Version
                    stream.Outbound <- ofp13.NewFeaturesRequest()
                } else {
                    // Connection should be severed if controller
                    // doesn't support switch version.
                    log.Println("Received unsupported ofp version", m.Version)
                    stream.Shutdown <- true
                }
            // After a vaild FeaturesReply has been received we
            // have all the information we need. Create a new
            // switch object and notify applications.
            case *ofp10.SwitchFeatures:
                log.Printf("Received ofp1.0 Switch feature response: %+v", *m)

                // Create a new switch and handover the stream
                sw := NewSwitch(stream, *m)

                // Register the actors
                sw.AddActor(c.actor)

                // Let switch instance all future messages..
                return
            case *ofp13.SwitchFeatures:
                log.Printf("Received ofp1.3 Switch feature response: %+v", *m)

            // An error message may indicate a version mismatch. We
            // disconnect if an error occurs this early.
            case *ofp10.ErrorMsg:
                log.Println(m)
                stream.Version = m.Header.Version
                stream.Shutdown <- true
            case *ofp13.ErrorMsg:
                log.Printf("Received ofp1.3 error msg: %+v", *m)
            }
        case err := <-stream.Error:
            // The connection has been shutdown.
            log.Println(err)
            return
        case <-time.After(time.Second * 3):
            // This shouldn't happen. If it does, both the controller
            // and switch are no longer communicating. The TCPConn is
            // still established though.
            log.Println("Connection timed out.")
            return
        }
    }
}
