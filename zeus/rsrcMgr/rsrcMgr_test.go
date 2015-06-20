package rsrcMgr

import (
	"flag"
	"testing"

	"github.com/golang/glog"
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

	glog.Infof("Added provider: %+v", provider)
	glog.Infof("Provider State: %#v", rsrcMgr.rsrcDb["cpu"].Providers["host1"])
	glog.Infof("Provider State: %#v", rsrcMgr.rsrcDb["memory"].Providers["host1"])

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

	glog.Infof("Added provider: %+v", provider)
	glog.Infof("Resource State: %#v", rsrcMgr.rsrcDb["vlan"])
	glog.Infof("Provider State: %#v", rsrcMgr.rsrcDb["vlan"].Providers["global"])

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

	glog.Infof("Got alloc Resp: %+v", respRsrsList)
	glog.Infof("Provider State: %#v", rsrcMgr.rsrcDb["cpu"].Providers["host1"])
	glog.Infof("Provider State: %#v", rsrcMgr.rsrcDb["memory"].Providers["host1"])

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

	glog.Infof("Got alloc Resp: %+v", respRsrsList)
	glog.Infof("Provider State: %#v", rsrcMgr.rsrcDb["vlan"].Providers["global"])
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
		glog.Errorf("Got unexpected alloc Resp: %+v", respRsrsList)
		t.Errorf("No Error allocating cpu/mem resource")
	}

	glog.Infof("Provider State: %#v", rsrcMgr.rsrcDb["cpu"].Providers["host1"])
	glog.Infof("Provider State: %#v", rsrcMgr.rsrcDb["memory"].Providers["host1"])

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
		glog.Infof("Got unexpected alloc Resp: %+v", respRsrsList)
		t.Errorf("No Error allocating vlan resource")
	}

	glog.Infof("Provider State: %#v", rsrcMgr.rsrcDb["vlan"].Providers["global"])
}
