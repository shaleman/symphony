package main

import (
	"flag"
	"os"
	"time"

	"github.com/contiv/symphony/zeus/altaCtrler"
	"github.com/contiv/symphony/zeus/api"
	"github.com/contiv/symphony/zeus/netCtrler"
	"github.com/contiv/symphony/zeus/nodeCtrler"
	"github.com/contiv/symphony/zeus/rsrcMgr"
	"github.com/contiv/symphony/zeus/volumesCtrler"

	//"pkg/altaspec"
	"github.com/contiv/symphony/pkg/confStore"
	"github.com/contiv/symphony/pkg/confStore/confStoreApi"

	"github.com/golang/glog"
)

const ZEUS_MASTER_TTL = 10 // mastership TTL is 10sec

var stopMasterChan chan bool
var stopSlaveChan chan bool

var cStore confStoreApi.ConfStorePlugin

// Main run loop for Zeus master
func runLoopMaster() {
	// Start the resource mgr
	rsrcMgr.Init(cStore)

	// Start Node controller
	err := nodeCtrler.Init(cStore)
	if err != nil {
		glog.Fatalf("Failed to create node mgr")
	}

	// Start the Alta controller
	err = altaCtrler.Init(cStore)
	if err != nil {
		glog.Fatalf("Failed to create alta mgr")
	}

	// Create volume controller
	err = volumesCtrler.Init(cStore)
	if err != nil {
		glog.Fatalf("Failed to create volume ctrler")
	}

	// Restore resources from conf store
	err = rsrcMgr.Restore()
	if err != nil {
		glog.Errorf("Failed to restore resources from conf store")
	}

	// Initialize network controller
	netCtrler.Init()

	// Restore state
	err = volumesCtrler.RestoreVolumes()
	if err != nil {
		glog.Errorf("Error restoring volumes. Err: %v", err)
	}

	// Restore alta container state
	err = altaCtrler.RestoreAltaActors()
	if err != nil {
		glog.Errorf("Error restoring volumes. Err: %v", err)
	}

	// Start the HTTP server
	go api.CreateServer(8000)

	glog.Infof("Master service is running")

	cnt := 0
	for {
		select {
		case <-stopMasterChan:
			glog.Infof("Exiting master loop")
			return
		case <-time.After(time.Second * 30):
			if cnt == 0 {
				/*
				   glog.Infof("Creating alta..")
				   altaCtrler.CreateAlta(&altaspec.AltaConfig{
				       Name: "first",
				       Image: "ubuntu:14.04",
				       Cpu: "1",
				       Memory: "500MB",
				       Command: "/bin/sh",
				       Environment: []string{ "TEST_ENV=test" },
				   })
				*/
			}
			cnt++
		}
	}
}

// Start the master run loop
func becomeMaster() {
	glog.Infof("Becoming master")

	// Drain the stop channel if there are old commands
	close(stopMasterChan)
	stopMasterChan = make(chan bool, 1)

	go runLoopMaster()
}

// Become a slave by simply stopping the master run loop
func becomeSlave() {
	glog.Infof("Stopping all master jobs")
	stopMasterChan <- true
}

// Main function
func main() {
	// Enable glog logging
	flag.Lookup("logtostderr").Value.Set("true")

	// Determine what role we should run on by trying to acquire master lock
	// If we acquire the lock, we run as master. If we fail we run as slave

	// Create the conf store client
	cStore = confStore.NewConfStore()

	// Create channels for run loop
	stopMasterChan = make(chan bool, 1)
	stopSlaveChan = make(chan bool, 1)

	myId, _ := os.Hostname()

	// Create the lock
	masterLock, err := cStore.NewLock("zeus/master", myId, ZEUS_MASTER_TTL)
	if err != nil {
		glog.Fatalf("Could not create master lock. Err: %v", err)
	}

	// Try to acquire the lock
	err = masterLock.Acquire(0)
	if err != nil {
		// We dont expect any error during acquire.
		glog.Fatalf("Error while acquiring lock. Err: %v", err)
	}

	glog.Infof("Waiting to become master")

	isMaster := false

	// Wait for lock events
	select {
	case event := <-masterLock.EventChan():
		if event.EventType == confStoreApi.LockAcquired {
			glog.Infof("Master lock acquired")

			isMaster = true
		}
	case <-time.After(time.Second * 5):
		glog.Infof("Could not acquire master lock in 5sec, becoming a slave")
	}

	// Different run loops based on mastership
	if isMaster {
		becomeMaster()
	} else {
		becomeSlave()
	}

	// Main run loop waiting on master lock
	for {
		// Wait for lock events
		select {
		case event := <-masterLock.EventChan():
			if event.EventType == confStoreApi.LockAcquired {
				glog.Infof("Master lock acquired")

				becomeMaster()
			} else if event.EventType == confStoreApi.LockLost {
				glog.Infof("Master lock lost. Becoming slave")

				becomeSlave()
			}
		}
	}
}
