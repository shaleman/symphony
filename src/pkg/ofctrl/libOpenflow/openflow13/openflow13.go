package openflow13

// Package openflow13 provides OpenFlow 1.3 structs along with Read
// and Write methods for each.
// OpenFlow Wire Protocol 0x04
//
// Struct documentation is taken from the OpenFlow Switch
// Specification Version 1.3.3.
// https://www.opennetworking.org/images/stories/downloads/sdn-resources/onf-specifications/openflow/openflow-spec-v1.3.3.pdf

import (
    "encoding/binary"
    "errors"

    "pkg/ofctrl/libOpenflow/protocol/eth"
    "pkg/ofctrl/libOpenflow/common"
    "pkg/ofctrl/libOpenflow/util"
)

const (
    VERSION = 4
)

// Echo request/reply messages can be sent from either the
// switch or the controller, and must return an echo reply. They
// can be used to indicate the latency, bandwidth, and/or
// liveness of a controller-switch connection.
func NewEchoRequest() *common.Header {
    h := common.NewOfp13Header()
    h.Type = Type_EchoRequest
    return &h
}

// Echo request/reply messages can be sent from either the
// switch or the controller, and must return an echo reply. They
// can be used to indicate the latency, bandwidth, and/or
// liveness of a controller-switch connection.
func NewEchoReply() *common.Header {
    h := common.NewOfp13Header()
    h.Type = Type_EchoReply
    return &h
}

// ofp_type 1.3
const (
    /* Immutable messages. */
    Type_Hello          = 0
    Type_Error          = 1
    Type_EchoRequest    = 2
    Type_EchoReply      = 3
    Type_Experimenter   = 4

    /* Switch configuration messages. */
    Type_FeaturesRequest    = 5
    Type_FeaturesReply      = 6
    Type_GetConfigRequest   = 7
    Type_GetConfigReply     = 8
    Type_SetConfig          = 9

    /* Asynchronous messages. */
    Type_PacketIn       = 10
    Type_FlowRemoved    = 11
    Type_PortStatus     = 12

    /* Controller command messages. */
    Type_PacketOut      = 13
    Type_FlowMod        = 14
    Type_GroupMod       = 15
    Type_PortMod        = 16
    Type_TableMod       = 17

    /* Multipart messages. */
    Type_MultiPartRequest   = 18
    Type_MultiPartReply     = 19

    /* Barrier messages. */
    Type_BarrierRequest     = 20
    Type_BarrierReply       = 21

    /* Queue Configuration messages. */
    Type_QueueGetConfigRequest  = 22
    Type_QueueGetConfigReply    = 23

    /* Controller role change request messages. */
    Type_RoleRequest            = 24
    Type_RoleReply              = 25

    /* Asynchronous message configuration. */
    Type_GetAsyncRequest        = 26
    Type_GetAsyncReply          = 27
    Type_SetAsync               = 28

    /* Meters and rate limiters configuration messages. */
    Type_MeterMod               = 29
)

// When the controller wishes to send a packet out through the
// datapath, it uses the OFPT_PACKET_OUT message: The buffer_id
// is the same given in the ofp_packet_in message. If the
// buffer_id is -1, then the packet data is included in the data
// array. If OFPP_TABLE is specified as the output port of an
// action, the in_port in the packet_out message is used in the
// flow table lookup.
type PacketOut struct {
    common.Header
    BufferId    uint32
    InPort      uint32
    ActionsLen  uint16
    pad         []byte
    Actions     []Action
    Data        util.Message
}

func NewPacketOut() *PacketOut {
    p := new(PacketOut)
    p.Header = common.NewOfp13Header()
    p.Header.Type = Type_PacketOut
    p.BufferId = 0xffffffff
    p.InPort = P_ANY
    p.ActionsLen = 0
    p.pad = make([]byte, 6)
    p.Actions = make([]Action, 0)
    return p
}

func (p *PacketOut) AddAction(act Action) {
    p.Actions = append(p.Actions, act)
    p.ActionsLen += act.Len()
}

func (p *PacketOut) Len() (n uint16) {
    n += p.Header.Len()
    n += 16
    for _, a := range p.Actions {
        n += a.Len()
    }
    n += p.Data.Len()
    //if n < 72 { return 72 }
    return
}

func (p *PacketOut) MarshalBinary() (data []byte, err error) {
    data = make([]byte, int(p.Len()))
    b := make([]byte, 0)
    n := 0

    p.Header.Length = p.Len()
    b, err = p.Header.MarshalBinary()
    copy(data[n:], b)
    n += len(b)

    binary.BigEndian.PutUint32(data[n:], p.BufferId)
    n += 4
    binary.BigEndian.PutUint32(data[n:], p.InPort)
    n += 4
    binary.BigEndian.PutUint16(data[n:], p.ActionsLen)
    n += 2
    n += 6 // for pad

    for _, a := range p.Actions {
        b, err = a.MarshalBinary()
        copy(data[n:], b)
        n += len(b)
    }

    b, err = p.Data.MarshalBinary()
    copy(data[n:], b)
    n += len(b)
    return
}

func (p *PacketOut) UnmarshalBinary(data []byte) error {
    err := p.Header.UnmarshalBinary(data)
    n := p.Header.Len()

    p.BufferId = binary.BigEndian.Uint32(data[n:])
    n += 4
    p.InPort = binary.BigEndian.Uint32(data[n:])
    n += 4
    p.ActionsLen = binary.BigEndian.Uint16(data[n:])
    n += 2

    n += 6 // for pad

    for n < (n + p.ActionsLen) {
        a := DecodeAction(data[n:])
        p.Actions = append(p.Actions, a)
        n += a.Len()
    }

    err = p.Data.UnmarshalBinary(data[n:])
    return err
}

// ofp_packet_in 1.3
type PacketIn struct {
    common.Header
    BufferId uint32
    TotalLen uint16
    Reason   uint8
    TableId  uint8
    Cookie   uint64
    Match    Match
    pad      []uint8
    Data     eth.Ethernet
}

func NewPacketIn() *PacketIn {
    p := new(PacketIn)
    p.Header = common.NewOfp13Header()
    p.Header.Type = Type_PacketIn
    p.BufferId = 0xffffffff
    p.Reason = 0
    p.TableId = 0
    p.Cookie = 0
    p.Match = *NewMatch()
    return p
}

func (p *PacketIn) Len() (n uint16) {
    n += p.Header.Len()
    n += 16
    n += p.Match.Len()
    n += 2
    n += p.Data.Len()
    return
}

func (p *PacketIn) MarshalBinary() (data []byte, err error) {
    data, err = p.Header.MarshalBinary()

    b := make([]byte, 16)
    n := 0
    binary.BigEndian.PutUint32(b, p.BufferId)
    n += 4
    binary.BigEndian.PutUint16(b[n:], p.TotalLen)
    n += 2
    b[n] = p.Reason
    n += 1
    b[n] = p.TableId
    n += 1
    binary.BigEndian.PutUint64(b, p.Cookie)
    n += 8
    data = append(data, b...)

    b, err = p.Match.MarshalBinary()
    data = append(data, b...)

    b = make([]byte, 2)
    copy(b[0:], p.pad)
    data = append(data, b...)

    b, err = p.Data.MarshalBinary()
    data = append(data, b...)
    return
}

func (p *PacketIn) UnmarshalBinary(data []byte) error {
    err := p.Header.UnmarshalBinary(data)
    n := p.Header.Len()

    p.BufferId = binary.BigEndian.Uint32(data[n:])
    n += 4
    p.TotalLen = binary.BigEndian.Uint16(data[n:])
    n += 2
    p.Reason = data[n]
    n += 1
    p.TableId = data[n]
    n += 1
    p.Cookie = binary.BigEndian.Uint64(data[n:])
    n += 8

    err = p.Match.UnmarshalBinary(data[n:])
    n += p.Match.Len()

    copy(p.pad, data[n:])
    n += 2

    err = p.Data.UnmarshalBinary(data[n:])
    return err
}

// ofp_packet_in_reason 1.3
const (
    R_NO_MATCH = iota   /* No matching flow (table-miss flow entry). */
    R_ACTION            /* Action explicitly output to controller. */
    R_INVALID_TTL       /* Packet has invalid TTL */
)

// ofp_vendor 1.3
type VendorHeader struct {
    Header common.Header /*Type OFPT_VENDOR*/
    Vendor uint32
}

func (v *VendorHeader) Len() (n uint16) {
    return v.Header.Len() + 4
}

func (v *VendorHeader) MarshalBinary() (data []byte, err error) {
    data, err = v.Header.MarshalBinary()

    b := make([]byte, 4)
    binary.BigEndian.PutUint32(data[:4], v.Vendor)

    data = append(data, b...)
    return
}

func (v *VendorHeader) UnmarshalBinary(data []byte) error {
    if len(data) < int(v.Len()) {
        return errors.New("The []byte the wrong size to unmarshal an " +
            "VendorHeader message.")
    }
    v.Header.UnmarshalBinary(data)
    n := int(v.Header.Len())
    v.Vendor = binary.BigEndian.Uint32(data[n:])
    return nil
}