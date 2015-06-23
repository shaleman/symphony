package main

import (
	"net"
	"os/user"
	"time"

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

var localIpAddr net.IP

// Register the node with registry
func registerNode() {
	// Wait for everything to be initialized before advertizing ourselves
	// FIXME: we need OVS to be connected before we connect to zeus. OVS can take up
	// to 10 second to connect to controller
	time.Sleep(time.Second * 10)

	// Get the local address to bind to
	localIpAddr, err := cStore.GetLocalAddr()
	if err != nil {
		log.Fatalf("Could not find a local address to bind to. Err %v", err)
	}

	srvInfo := confStoreApi.ServiceInfo{
		ServiceName: "athena",
		HostAddr:    localIpAddr,
		Port:        API_PORT,
	}

	// Register the node with service registry
	err = cStore.RegisterService(srvInfo)
	if err != nil {
		log.Fatalf("Error registering service. Err: %v", err)
	}

	log.Infof("Registered athena service with registry")
}

// Main function
func main() {
	// Make sure we are running as root
	usr, _ := user.Current()
	if usr.Username != "root" {
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

	// Add the node registry.
	go registerNode()

	// Create a HTTP server
	createServer(API_PORT)

}
