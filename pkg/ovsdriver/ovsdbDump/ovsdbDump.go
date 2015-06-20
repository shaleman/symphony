package main

import (
	"flag"
	"time"

	"github.com/contiv/symphony/pkg/ovsdriver"
)

func main() {
	// FIXME: Temporary hack for testing
	flag.Lookup("logtostderr").Value.Set("true")

	// Connect to OVS
	ovsDriver := ovsdriver.NewOvsDriver()

	// Wait a little for cache to be populated
	time.Sleep(time.Second * 1)

	// Dump the cache contents
	ovsDriver.PrintCache()

}
