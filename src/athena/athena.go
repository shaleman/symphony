package main

import (
    "flag"
    "time"

    "pkg/confStore"
    "pkg/confStore/confStoreApi"

    "github.com/golang/glog"
)

const API_PORT   = 8100     // Athena listens for REST api on this port

// Alta manager
var altaMgr   *AltaMgr

// Network agent
var netAgent    *NetAgent

// volume agent
var volumeAgent    *VolumeAgent

// Conf store plugin
var cStore   confStoreApi.ConfStorePlugin

// Register the node with registry
func registerNode() {
    // Wait for everything to be initialized before advertizing ourselves
    time.Sleep(time.Second * 1)

    // Get the local address to bind to
    lclAddr, err := cStore.GetLocalAddr()
    if (err != nil) {
        glog.Fatalf("Could not find a local address to bind to. Err %v", err)
    }

    srvInfo := confStoreApi.ServiceInfo {
        ServiceName:    "athena",
        HostAddr:       lclAddr,
        Port:           API_PORT,
    }

    // Register the node with service registry
    err = cStore.RegisterService(srvInfo)
    if (err != nil) {
        glog.Fatalf("Error registering service. Err: %v", err)
    }

    glog.Infof("Registered athena service with registry")
}

// Main function
func main() {
    // FIXME: Temporary hack for testing
    flag.Lookup("logtostderr").Value.Set("true")

    // Create a alta manager
    altaMgr = NewAltaMgr()

    // Create a network agent
    netAgent = NewNetAgent()

    // Create a volume agent
    volumeAgent = NewVolumeAgent()

    // create conf store
    cStore = confStore.NewConfStore()

    // Add the node registry.
    go registerNode()

    // Create a HTTP server
    createServer(API_PORT)



}