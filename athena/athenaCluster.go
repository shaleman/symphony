package main

// This file has all clustering related stuff..

import (
	"fmt"
	"time"
	"os"

	"github.com/contiv/symphony/pkg/confStore/confStoreApi"
	"github.com/contiv/symphony/pkg/psutil"
	"github.com/contiv/symphony/pkg/altaspec"

	log "github.com/Sirupsen/logrus"
)

type ClusterAgent struct {
	localIp		string		// Local IP address
	apiPortNo	int			// port number where we are listening

	// List of masters
	masterDb	map[string]*confStoreApi.ServiceInfo
}

// Create a new cluster agent
func NewClusterAgent(localIp string, portNo int) (*ClusterAgent) {
	cAgent := new(ClusterAgent)
	cAgent.localIp = localIp
	cAgent.apiPortNo = portNo
	cAgent.masterDb = make(map[string]*confStoreApi.ServiceInfo)

	// Register the node and refresh it periodically
	go cAgent.registerNode(localIp, portNo)

	// start a backgroud poll to monitor running containers
	go cAgent.monitorContainers()

	return cAgent
}

// Register the node with registry
func (self *ClusterAgent) registerNode(localIp string, portNo int) {
	// service info
	srvInfo := confStoreApi.ServiceInfo{
		ServiceName: "athena",
		HostAddr:    localIp,
		Port:        portNo,
	}

	// Register the node with service registry
	err := cStore.RegisterService(srvInfo)
	if err != nil {
		log.Fatalf("Error registering service. Err: %v", err)
	}

	log.Infof("Registered athena service with registry")
}

// Save master info.
// FIXME: Send periodic update about all running containers to master
func (self *ClusterAgent) addMaster(masterInfo confStoreApi.ServiceInfo) error {
	// build master key
	masterKey := fmt.Sprintf("%s:%d", masterInfo.HostAddr, masterInfo.Port)

	log.Infof("Adding master: %s", masterKey)

	// Save it itn the DB
	self.masterDb[masterKey] = &masterInfo

	// Add the master in ofnet agent
	err := netAgent.AddMaster(masterInfo.HostAddr)
	if err != nil {
		log.Errorf("Error adding master %s. Err: %v", masterKey, err)
	}

	return nil
}

// Get locally available resources..
func (self *ClusterAgent) getNodeSpec() altaspec.NodeSpec {
	// Get the number of CPU
	numCpu, _ := psutil.CPUCounts(true)

	// CPU speed
	cpuInfo, _ := psutil.CPUInfo()
	cpuMhz := uint64(cpuInfo[0].Mhz)

	// Get the total memory
	memInfo, _ := psutil.VirtualMemory()
	memTotal := memInfo.Total

	// Get the host name
	hostName, _ := os.Hostname()

	// Create response
	nodeSpec := altaspec.NodeSpec{
		HostName:    hostName,
		NumCpuCores: numCpu,
		CpuMhz:      cpuMhz,
		MemTotal:    memTotal,
	}

	return nodeSpec
}

// Periodically send container info to all masters
func (self *ClusterAgent) monitorContainers() {
	for {
		// Wakeup every second
		time.Sleep(1 * time.Second)

		// FIXME: do something here..
	}
}
