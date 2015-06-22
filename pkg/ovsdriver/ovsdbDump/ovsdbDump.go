package main

import (
	"time"

	"github.com/contiv/symphony/pkg/ovsdriver"
)

func main() {
	// Connect to OVS
	ovsDriver := ovsdriver.NewOvsDriver()

	// Wait a little for cache to be populated
	time.Sleep(time.Second * 1)

	// Dump the cache contents
	ovsDriver.PrintCache()
}
