package altaspec

import (
// "fmt"
)

// Volume parameters
type AltaVolumeSpec struct {
	DatastoreType     string // host volumes, Ceph/gluster/nfs/FC etc
	DatastoreVolumeId string // Volume identifier on the datastore device name/directory etc
	VolumeSize        uint   // Size of the volume
}

// Volume to be mounted on to the container
type AltaVolumeBind struct {
	DatastoreType     string // host volumes, Ceph/gluster/nfs/FC etc
	DatastoreVolumeId string // Volume identifier on the datastore device name/directory etc
	BindMountPoint    string // Where to mount it on the container
}

// Network parameters
type AltaNetSpec struct {
	NetworkName string // Name of the network
	VlanId      uint16 // Vlan Id
	Vni         uint32 // Virtual network id(Vxlan VNI)
}

// Network interface definition
type AltaNetIf struct {
	NetworkName     string // Name of the network
	IntfMacAddr     string // Mac address for the interface
	IntfIpv4Addr    string // IP address for the interface
	IntfIpv4Masklen int    // IP netmask length
	Ipv4Gateway     string // default gateway
}

// Specifications for a Alta container
type AltaSpec struct {
	AltaId      string   // Unique identifier for the container
	AltaName    string   // Optional Unique name for the container
	NumCpu      uint32   // Number of CPUs cores
	CpuPerc     int64    // CPU percentage on eahc core
	Memory      int64    // Memory in MBs
	Image       string   // Image for the container
	Command     []string // Command to run in the container
	Args        []string // Arguments for the command
	WorkingDir  string   // Working directory for the command
	EnvList     []string // Environment variables
	ExposePorts []string // List of ports to expose(alternative to EXPOSE keyword in dockerfile)
	PortMapList []string // Port mapping(for externally visible ports)

	Volumes    []AltaVolumeBind // Volumes to be mounted
	NetworkIfs []AltaNetIf      // Network interfaces to be created
}

// Slave node information
type NodeSpec struct {
	HostName    string // Name of the host
	NumCpuCores int    // Logical cpu cores(one per hyperthread)
	CpuMhz      uint64 // CPU megahertz
	MemTotal    uint64 // Total available memory in bytes
}

// Request was success
type ReqSuccess struct {
	Success bool
}

type AltaContext struct {
	AltaId      string
	ContainerId string
}

// Docker compose options
type DockerCompose struct {
	Image          string `json:"image"`
	Build          string `json:"build"`
	Command        string `json:"command"`
	Links          string `json:"links"`
	External_links string `json:"external_links"`
	Ports          string `json:"ports"`
	Expose         string `json:"expose"`
	Volumes        string `json:"volumes"`
	Volumes_from   string `json:"volumes_from"`
	Environment    string `json:"environment"`
	Env_file       string `json:"env_file"`
	Extends        string `json:"extends"`
	Net            string `json:"net"`
	Pid            string `json:"pid"`
	Dns            string `json:"dns"`
	Dns_search     string `json:"dns_search"`
	Cpu_shares     string `json:"cpu_shares"`
	Working_dir    string `json:"working_dir"`
	Entrypoint     string `json:"entrypoint"`
	User           string `json:"user"`
	Hostname       string `json:"hostname"`
	Domainname     string `json:"domainname"`
	Mem_limit      string `json:"mem_limit"`
	Privileged     string `json:"privileged"`
	Restart        string `json:"restart"`
	Stdin_open     string `json:"stdin_open"`
	Tty            string `json:"tty"`
}

// Alta container specification as specified by the user
// We are trying to be backward compatible with Docker Compose spec.
// But, its not clear if most of the options in docker compose are relavent to us
type AltaConfig struct {
	Name        string   `json:"name"`        // Optional name
	Image       string   `json:"image"`       // Image to run
	Cpu         string   `json:"cpu"`         // CPU Eg: 2, 1.5, .5, 4 etc
	Memory      string   `json:"memory"`      // Memory Eg: 1G, 16GB, 200MB, 2g etc
	Command     string   `json:"command"`     // override entry point
	Network     []string `json:"network"`     // List of networks to join
	Environment []string `json:"environment"` // Optional environment variable
	Volumes     []string `json:"volumes"`     // Additional volumes to mount
}
