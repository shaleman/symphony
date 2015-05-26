package openflow13

import (
    "encoding/binary"

    "pkg/ofctrl/libOpenflow/common"
    "pkg/ofctrl/libOpenflow/util"
)

// BEGIN: ofp13 - 7.4.4
// ofp_error_msg 1.3
type ErrorMsg struct {
    common.Header
    Type    uint16
    Code    uint16
    Data    util.Buffer
}

func NewErrorMsg() *ErrorMsg {
    e := new(ErrorMsg)
    e.Data = *util.NewBuffer(make([]byte, 0))
    return e
}

func (e *ErrorMsg) Len() (n uint16) {
    n = e.Header.Len()
    n += 2
    n += 2
    n += e.Data.Len()
    return
}

func (e *ErrorMsg) MarshalBinary() (data []byte, err error) {
    data = make([]byte, int(e.Len()))
    next := 0

    bytes, err := e.Header.MarshalBinary()
    copy(data[next:], bytes)
    next += len(bytes)
    binary.BigEndian.PutUint16(data[next:], e.Type)
    next += 2
    binary.BigEndian.PutUint16(data[next:], e.Code)
    next += 2
    bytes, err = e.Data.MarshalBinary()
    copy(data[next:], bytes)
    next += len(bytes)
    return
}

func (e *ErrorMsg) UnmarshalBinary(data []byte) error {
    next := 0
    e.Header.UnmarshalBinary(data[next:])
    next += int(e.Header.Len())
    e.Type = binary.BigEndian.Uint16(data[next:])
    next += 2
    e.Code = binary.BigEndian.Uint16(data[next:])
    next += 2
    e.Data.UnmarshalBinary(data[next:])
    next += int(e.Data.Len())
    return nil
}

// ofp_error_type 1.3
const (
    ET_HELLO_FAILED     = 0     /* Hello protocol failed. */
    ET_BAD_REQUEST      = 1     /* Request was not understood. */
    ET_BAD_ACTION       = 2     /* Error in action description. */
    ET_BAD_INSTRUCTION  = 3     /* Error in instruction list. */
    PET_BAD_MATCH       = 4     /* Error in match. */
    ET_FLOW_MOD_FAILED  = 5      /* Problem modifying flow entry. */
    ET_GROUP_MOD_FAILED = 6     /* Problem modifying group entry. */
    ET_PORT_MOD_FAILED  = 7     /* Port mod request failed. */
    ET_TABLE_MOD_FAILED = 8      /* Table mod request failed. */
    ET_QUEUE_OP_FAILED  = 9      /* Queue operation failed. */
    ET_ROLE_REQUEST_FAILED   = 11 /* Controller Role request failed. */
    ET_METER_MOD_FAILED      = 12 /* Error in meter. */
    ET_TABLE_FEATURES_FAILED = 13 /* Setting table features failed. */
    ET_EXPERIMENTER          = 0xffff /* Experimenter error messages. */
)

// ofp_hello_failed_code 1.3
const (
    HFC_INCOMPATIBLE = iota
    HFC_EPERM
)

// ofp_bad_request_code 1.3
const (
    BRC_BAD_VERSION = iota
    BRC_BAD_TYPE
    BRC_BAD_MULTIPART
    BRC_BAD_EXPERIMENTER

    BRC_BAD_EXP_TYPE
    BRC_EPERM
    BRC_BAD_LEN
    BRC_BUFFER_EMPTY
    BRC_BUFFER_UNKNOWN
    BRC_BAD_TABLE_ID
    BRC_IS_SLAVE
    BRC_BAD_PORT
    BRC_BAD_PACKET
    BRC_MULTIPART_BUFFER_OVERFLOW
)

// ofp_bad_action_code 1.3
const (
    BAC_BAD_TYPE = iota
    BAC_BAD_LEN
    BAC_BAD_EXPERIMENTER
    BAC_BAD_EXP_TYPE
    BAC_BAD_OUT_PORT
    BAC_BAD_ARGUMENT
    BAC_EPERM
    BAC_TOO_MANY
    BAC_BAD_QUEUE
    BAC_BAD_OUT_GROUP
    BAC_MATCH_INCONSISTENT
    BAC_UNSUPPORTED_ORDER
    BAC_BAD_TAG
    BAC_BAD_SET_TYPE
    BAC_BAD_SET_LEN
    BAC_BAD_SET_ARGUMENT
)

// ofp_bad_instruction_code 1.3
const (
    BIC_UNKNOWN_INST    = 0     /* Unknown instruction. */
    BIC_UNSUP_INST      = 1     /* Switch or table does not support the instruction. */
    BIC_BAD_TABLE_ID    = 2     /* Invalid Table-ID specified. */
    BIC_UNSUP_METADATA  = 3     /* Metadata value unsupported by datapath. */
    BIC_UNSUP_METADATA_MASK = 4     /* Metadata mask value unsupported by datapath. */
    BIC_BAD_EXPERIMENTER    = 5     /* Unknown experimenter id specified. */
    BIC_BAD_EXP_TYPE    = 6     /* Unknown instruction for experimenter id. */
    BIC_BAD_LEN         = 7     /* Length problem in instructions. */
    BIC_EPERM           = 8     /* Permissions error. */
)

// ofp_flow_mod_failed_code 1.3
const (
    FMFC_UNKNOWN         = 0   /* Unspecified error. */
    FMFC_TABLE_FULL      = 1   /* Flow not added because table was full. */
    FMFC_BAD_TABLE_ID    = 2   /* Table does not exist */
    FMFC_OVERLAP         = 3   /* Attempted to add overlapping flow with CHECK_OVERLAP flag set. */
    FMFC_EPERM           = 4   /* Permissions error. */
    FMFC_BAD_TIMEOUT     = 5   /* Flow not added because of unsupported idle/hard timeout. */
    FMFC_BAD_COMMAND     = 6   /* Unsupported or unknown command. */
    FMFC_BAD_FLAGS       = 7   /* Unsupported or unknown flags. */
)

// ofp_bad_match_code 1.3
const (
    BMC_BAD_TYPE    = 0     /* Unsupported match type specified by the match */
    BMC_BAD_LEN     = 1     /* Length problem in match. */
    BMC_BAD_TAG     = 2     /* Match uses an unsupported tag/encap. */
    BMC_BAD_DL_ADDR_MASK    = 3     /* Unsupported datalink addr mask - switch does not support arbitrary datalink address mask. */
    BMC_BAD_NW_ADDR_MASK    = 4     /* Unsupported network addr mask - switch does not support arbitrary network address mask. */
    BMC_BAD_WILDCARDS       = 5     /* Unsupported combination of fields masked or omitted in the match. */
    BMC_BAD_FIELD   = 6     /* Unsupported field type in the match. */
    BMC_BAD_VALUE   = 7     /* Unsupported value in a match field. */
    BMC_BAD_MASK    = 8     /* Unsupported mask specified in the match, field is not dl-address or nw-address. */
    BMC_BAD_PREREQ  = 9     /* A prerequisite was not met. */
    BMC_DUP_FIELD   = 10    /* A field type was duplicated. */
    BMC_EPERM       = 11    /* Permissions error. */
)

// ofp_group_mod_failed_code 1.3
const (
    GMFC_GROUP_EXISTS       = 0     /* Group not added because a group ADD attempted to replace an already-present group. */
    GMFC_INVALID_GROUP      = 1     /* Group not added because Group specified is invalid. */
    GMFC_WEIGHT_UNSUPPORTED = 2     /* Switch does not support unequal load 105 ➞ 2013; The Open Networking Foundation OpenFlow Switch Specification Version 1.3.3 sharing with select groups. */
    GMFC_OUT_OF_GROUPS      = 3     /* The group table is full. */
    GMFC_OUT_OF_BUCKETS     = 4     /* The maximum number of action buckets for a group has been exceeded. */
    GMFC_CHAINING_UNSUPPORTED = 5   /* Switch does not support groups that forward to groups. */
    GMFC_WATCH_UNSUPPORTED  = 6     /* This group cannot watch the watch_port or watch_group specified. */
    GMFC_LOOP               = 7     /* Group entry would cause a loop. */
    GMFC_UNKNOWN_GROUP      = 8     /* Group not modified because a group MODIFY attempted to modify a non-existent group. */
    GMFC_CHAINED_GROUP      = 9     /* Group not deleted because another group is forwarding to it. */
    GMFC_BAD_TYPE           = 10    /* Unsupported or unknown group type. */
    GMFC_BAD_COMMAND        = 11    /* Unsupported or unknown command. */
    GMFC_BAD_BUCKET         = 12    /* Error in bucket. */
    GMFC_BAD_WATCH          = 13    /* Error in watch port/group. */
    GMFC_EPERM              = 14    /* Permissions error. */
)

// ofp_port_mod_failed_code 1.0
const (
    PMFC_BAD_PORT = iota
    PMFC_BAD_HW_ADDR
    PMFC_BAD_CONFIG
    PMFC_BAD_ADVERTISE
    PMFC_EPERM
)

// ofp_table_mod_failed_code
const (
    TMFC_BAD_TABLE = 0      /* Specified table does not exist. */
    TMFC_BAD_CONFIG = 1     /* Specified config is invalid. */
    TMFC_EPERM = 2          /* Permissions error. */
)

// ofp_queue_op_failed_code 1.0
const (
    QOFC_BAD_PORT = iota
    QOFC_BAD_QUEUE
    QOFC_EPERM
)

// END: ofp13 - 7.4.4
// END: ofp13 - 7.4
