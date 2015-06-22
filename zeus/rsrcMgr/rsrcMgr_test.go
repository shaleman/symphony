package rsrcMgr

import (
	"flag"
	"testing"

	log "github.com/Sirupsen/logrus"
)

func TestAddProvider(t *testing.T) {
	// Hack to log output
	flag.Lookup("logtostderr").Value.Set("true")

	// Initialize the resource mgr
	Init(nil)

	// provider to add
	provider := []ResourceProvide{
		{
			Type:     "cpu",
			Provider: "host1",
			UnitType: "fluid",
			NumRsrc:  4,
		},
		{
			Type:     "memory",
			Provider: "host1",
			UnitType: "fluid",
			NumRsrc:  5 * 1024,
		},
	}

	// Add a provider
	err := AddResourceProvider(provider)
	if err != nil {
		t.Errorf("Error adding provider %+v. Err: %v", provider, err)
	}

	log.Infof("Added provider: %+v", provider)
	log.Infof("Provider State: %#v", rsrcMgr.rsrcDb["cpu"].Providers["host1"])
	log.Infof("Provider State: %#v", rsrcMgr.rsrcDb["memory"].Providers["host1"])

	// provider to add
	provider = []ResourceProvide{
		{
			Type:     "vlan",
			Provider: "global",
			UnitType: "descrete",
			NumRsrc:  4094,
		},
	}

	// Add a provider
	err = AddResourceProvider(provider)
	if err != nil {
		t.Errorf("Error adding provider %+v. Err: %v", provider, err)
	}

	log.Infof("Added provider: %+v", provider)
	log.Infof("Resource State: %#v", rsrcMgr.rsrcDb["vlan"])
	log.Infof("Provider State: %#v", rsrcMgr.rsrcDb["vlan"].Providers["global"])

}

func TestAllocResource(t *testing.T) {
	// resource list
	rsrcList := []ResourceUse{
		{
			Type:     "cpu",
			Provider: "host1",
			UserKey:  "alta1234",
			NumRsrc:  2,
		},
		{
			Type:     "memory",
			Provider: "host1",
			UserKey:  "alta1234",
			NumRsrc:  1 * 1024,
		},
	}

	// Allocate the resource
	respRsrsList, err := AllocResources(rsrcList)
	if err != nil {
		t.Errorf("Error allocating cpu/mem resource. Err: %v", err)
	}

	log.Infof("Got alloc Resp: %+v", respRsrsList)
	log.Infof("Provider State: %#v", rsrcMgr.rsrcDb["cpu"].Providers["host1"])
	log.Infof("Provider State: %#v", rsrcMgr.rsrcDb["memory"].Providers["host1"])

	// Vlan resource to allocate
	rsrcList = []ResourceUse{
		{
			Type:     "vlan",
			Provider: "global",
			UserKey:  "net1234",
			NumRsrc:  5,
		},
	}

	// Allocate the resource
	respRsrsList, err = AllocResources(rsrcList)
	if err != nil {
		t.Errorf("Error allocating vlan resource. Err: %v", err)
	}

	log.Infof("Got alloc Resp: %+v", respRsrsList)
	log.Infof("Provider State: %#v", rsrcMgr.rsrcDb["vlan"].Providers["global"])
}

func TestAllocResourceFail(t *testing.T) {
	// resource list
	rsrcList := []ResourceUse{
		{
			Type:     "cpu",
			Provider: "host1",
			UserKey:  "alta1234",
			NumRsrc:  2,
		},
		{
			Type:     "memory",
			Provider: "host1",
			UserKey:  "alta1234",
			NumRsrc:  4.5 * 1024,
		},
	}

	// Allocate the resource
	respRsrsList, err := AllocResources(rsrcList)
	if err == nil {
		log.Errorf("Got unexpected alloc Resp: %+v", respRsrsList)
		t.Errorf("No Error allocating cpu/mem resource")
	}

	log.Infof("Provider State: %#v", rsrcMgr.rsrcDb["cpu"].Providers["host1"])
	log.Infof("Provider State: %#v", rsrcMgr.rsrcDb["memory"].Providers["host1"])

	// Vlan resource to allocate
	rsrcList = []ResourceUse{
		{
			Type:     "vlan",
			Provider: "global",
			UserKey:  "net1234",
			NumRsrc:  5000,
		},
	}

	// Allocate the resource
	respRsrsList, err = AllocResources(rsrcList)
	if err == nil {
		log.Infof("Got unexpected alloc Resp: %+v", respRsrsList)
		t.Errorf("No Error allocating vlan resource")
	}

	log.Infof("Provider State: %#v", rsrcMgr.rsrcDb["vlan"].Providers["global"])
}
