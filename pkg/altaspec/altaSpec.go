/***
Copyright 2014 Cisco Systems Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package altaspec

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

// Network endpoint definition
type AltaEndpoint struct {
	NetworkName     string // Name of the network
	IntfMacAddr     string // Mac address for the interface
	IntfIpv4Addr    string // IP address for the interface
	IntfIpv4Masklen int    // IP netmask length
	Ipv4Gateway     string // default gateway
}

type AltaSchedPolicy struct {
	SchedulerName string            // Name of the scheduler [leastUsed, binPack, random]
	RestartPolicy string            // restart policy [always, never, onFailure]
	NumRestart    int               // number of times to restart
	Filters       map[string]string // list of constraints
	Resources     []Resource        // list of resources requested
}

// Specifications for a Alta container
type AltaSpec struct {
	AltaId      string          // Unique identifier for the container
	AltaName    string          // Optional Unique name for the container
	NumCpu      uint32          // Number of CPUs cores
	CpuPerc     int64           // CPU percentage on eahc core
	Memory      int64           // Memory in MBs
	Image       string          // Image for the container
	Command     []string        // Command to run in the container
	Args        []string        // Arguments for the command
	WorkingDir  string          // Working directory for the command
	EnvList     []string        // Environment variables
	ExposePorts []string        // List of ports to expose(alternative to EXPOSE keyword in dockerfile)
	PortMapList []string        // Port mapping(for externally visible ports)
	SchedPolicy AltaSchedPolicy // Scheduler policy

	Volumes   []AltaVolumeBind // Volumes to be mounted
	Endpoints []AltaEndpoint   // Network endpoints to be created
}

// Resource available or consumed
type Resource struct {
	Type     string  // Resource type
	UnitType string  // 'descrete' or 'fluid'
	NumRsrc  float64 // number of resources provided or consumed
}

// Slave node information
type NodeSpec struct {
	HostName   string            // Name of the host
	Resources  []Resource        // List of resources
	Attributes map[string]string // List of attributes
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
	Name        string           `json:"name"`        // Optional name
	Image       string           `json:"image"`       // Image to run
	Cpu         string           `json:"cpu"`         // CPU Eg: 2, 1.5, .5, 4 etc
	Memory      string           `json:"memory"`      // Memory Eg: 1G, 16GB, 200MB, 2g etc
	Command     string           `json:"command"`     // override entry point
	Networks    []string         `json:"network"`     // List of networks to join
	Environment []string         `json:"environment"` // Optional environment variable
	Volumes     []AltaVolumeBind `json:"volumes"`     // Volumes to mount
}
