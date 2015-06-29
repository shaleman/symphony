package main

import (
	"os/user"

	"github.com/contiv/symphony/pkg/confStore"
	"github.com/contiv/symphony/pkg/confStore/confStoreApi"

	log "github.com/Sirupsen/logrus"
)

const API_PORT = 8100 // Athena listens for REST api on this port

// Alta manager
var altaMgr *AltaMgr

// Network agent
var netAgent *NetAgent

// volume agent
var volumeAgent *VolumeAgent

// Conf store plugin
var cStore confStoreApi.ConfStorePlugin

// cluster agent
var clusterAgent *ClusterAgent

// Main function
func main() {
	// Make sure we are running as root
	usr, err := user.Current()
	if (err != nil) || (usr.Username != "root") {
		log.Fatalf("This process can only be run as root")
	}

	// create conf store
	cStore = confStore.NewConfStore()

	// Create a alta manager
	altaMgr = NewAltaMgr()

	// Create a network agent
	netAgent = NewNetAgent()

	// Create a volume agent
	volumeAgent = NewVolumeAgent()

	// Get the local address to bind to
	ipAddr, err := cStore.GetLocalAddr()
	if err != nil {
		log.Fatalf("Could not find a local address to bind to. Err %v", err)
	}

	// Create the cluster agent
	clusterAgent = NewClusterAgent(ipAddr, API_PORT)

	// Create a HTTP server
	createServer(API_PORT)

}
