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

package main

// This file has all clustering related stuff..

import (
	"fmt"
	"os"
	"time"

	"github.com/contiv/objmodel/objdb"
	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/psutil"

	log "github.com/Sirupsen/logrus"
)

type ClusterAgent struct {
	localIp   string // Local IP address
	apiPortNo int    // port number where we are listening

	// List of masters
	masterDb map[string]*objdb.ServiceInfo
}

// Create a new cluster agent
func NewClusterAgent(localIp string, portNo int) *ClusterAgent {
	cAgent := new(ClusterAgent)
	cAgent.localIp = localIp
	cAgent.apiPortNo = portNo
	cAgent.masterDb = make(map[string]*objdb.ServiceInfo)

	// Register the node and refresh it periodically
	go cAgent.registerNode(localIp, portNo)

	// start a backgroud poll to monitor running containers
	go cAgent.monitorContainers()

	return cAgent
}

// Register the node with registry
func (self *ClusterAgent) registerNode(localIp string, portNo int) {
	// service info
	srvInfo := objdb.ServiceInfo{
		ServiceName: "athena",
		HostAddr:    localIp,
		Port:        portNo,
	}

	// Register the node with service registry
	err := cdb.RegisterService(srvInfo)
	if err != nil {
		log.Fatalf("Error registering service. Err: %v", err)
	}

	log.Infof("Registered athena service with registry")
}

// Save master info.
// FIXME: Send periodic update about all running containers to master
func (self *ClusterAgent) addMaster(masterInfo objdb.ServiceInfo) error {
	// build master key
	masterKey := fmt.Sprintf("%s:%d", masterInfo.HostAddr, masterInfo.Port)

	log.Infof("Adding master: %s", masterKey)

	// Save it in the DB
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
	cpuMhz := cpuInfo[0].Mhz

	// Get the total memory
	memInfo, _ := psutil.VirtualMemory()
	memTotal := memInfo.Total

	// Get the host name
	hostName, _ := os.Hostname()

	// Create response
	nodeSpec := altaspec.NodeSpec{
		HostName: hostName,
		Resources: []altaspec.Resource{
			{
				Type:     "cpu",
				UnitType: "fluid",
				NumRsrc:  float64(numCpu),
			},
			{
				Type:     "memory",
				UnitType: "fluid",
				NumRsrc:  float64(memTotal),
			},
		},
		Attributes: map[string]string{
			"hostname": hostName,
			"cpu-mhz":  fmt.Sprintf("%f", cpuMhz),
		},
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
