package netCtrler

import (
	"flag"
	"testing"

	"github.com/contiv/symphony/zeus/rsrcMgr"

	log "github.com/Sirupsen/logrus"
)

// Simple test to create a network and add an end point
func TestAddNetwork(t *testing.T) {
	// Hack to log output
	flag.Lookup("logtostderr").Value.Set("true")

	// initialize rsrcMgr since we use it for resource allocation
	rsrcMgr.Init(nil)

	// Initialize the ctrler
	Init()

	// Create network
	network, err := NewNetwork("default")
	if err != nil {
		t.Errorf("Error creating network default. Err: %v", err)
		return
	}

	log.Infof("Successfully Created network: %+v", network)

	// Create new endpoint
	ep, err := network.NewEndPoint("alta1234.0")
	if err != nil {
		t.Errorf("Error creating network endpoint. Err: %v", err)
		return
	}

	log.Infof("Successfully Created endpoint: %+v", ep)
}
