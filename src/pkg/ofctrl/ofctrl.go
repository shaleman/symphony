package ofctrl
// This library implements a simple openflow 1.3 controller

import (
    "net"
    "time"

    "github.com/shaleman/libOpenflow/common"
    "github.com/shaleman/libOpenflow/openflow13"
    "github.com/shaleman/libOpenflow/util"

    log "github.com/Sirupsen/logrus"
)

// Note: Command to make ovs connect to controller:
// ovs-vsctl set-controller <bridge-name> tcp:<ip-addr>:<port>
// E.g.    sudo ovs-vsctl set-controller ovsbr0 tcp:127.0.0.1:6633

// To enable openflow1.3 support in OVS:
// ovs-vsctl set bridge <bridge-name> protocols=OpenFlow10,OpenFlow11,OpenFlow12,OpenFlow13
// E.g. sudo ovs-vsctl set bridge ovsbr0 protocols=OpenFlow10,OpenFlow11,OpenFlow12,OpenFlow13

type AppInterface interface {
    // A Switch connected to the controller
    SwitchConnected(sw *OFSwitch)

    // Switch disconnected from the controller
    SwitchDisconnected(sw *OFSwitch)

    // Controller received a packet from the switch
    PacketRcvd(sw *OFSwitch, pkt *openflow13.PacketIn)
}

type Controller struct{
    app      AppInterface
}

// Create a new controller
func NewController(bridge string, app AppInterface) *Controller {
    c := new(Controller)

    // for debug logs
    //log.SetLevel(log.DebugLevel)

    // Save the handler
    c.app = app

    // Connect to unix socket
    // FIXME: dont hard code bridge name to ovsbr0 here
    conn, err := net.Dial("unix", "/var/run/openvswitch/" + bridge + ".mgmt")
    if (err != nil) {
        log.Fatalf("Failed to connect to unix socket. Err: %v", err)
        return nil
    }

    // Handle the connection
    go c.handleConnection(conn)

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

// Handle TCP connection from the switch
func (c *Controller) handleConnection(conn net.Conn) {
    stream := util.NewMessageStream(conn, c)

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
                if m.Version == openflow13.VERSION {
                    log.Infoln("Received Openflow 1.3 Hello message")
                    // Version negotiation is
                    // considered complete. Create
                    // new Switch and notifiy listening
                    // applications.
                    stream.Version = m.Version
                    stream.Outbound <- openflow13.NewFeaturesRequest()
                } else {
                    // Connection should be severed if controller
                    // doesn't support switch version.
                    log.Println("Received unsupported ofp version", m.Version)
                    stream.Shutdown <- true
                }
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


// Demux based on message version
func (c *Controller) Parse(b []byte) (message util.Message, err error) {
    switch b[0] {
    case openflow13.VERSION:
        message, err = openflow13.Parse(b)
    default:
        log.Errorf("Received unsupported openflow version: %d", b[0])
    }
    return
}
