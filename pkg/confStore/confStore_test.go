package confStore

import (
	"flag"
	"fmt"
	"testing"
	"time"

	api "github.com/contiv/symphony/pkg/confStore/confStoreApi"

	"github.com/golang/glog"
)

type JsonObj struct {
	Value string
}

// New confstore
var cStore = NewConfStore()

// Perform Set/Get operation on default conf store
func TestSetGet(t *testing.T) {
	// Hack to log output
	flag.Lookup("logtostderr").Value.Set("true")
	// flag.Lookup("v").Value.Set("2")

	// Set
	setVal := JsonObj{
		Value: "test1",
	}
	err := cStore.SetObj("/contiv.io/test", setVal)
	if err != nil {
		fmt.Printf("Error setting key. Err: %v\n", err)
		t.Errorf("Error setting key")
	}

	var retVal JsonObj
	err = cStore.GetObj("/contiv.io/test", &retVal)
	if err != nil {
		fmt.Printf("Error getting key. Err: %v\n", err)
		t.Errorf("Error getting key")
	}

	if retVal.Value != "test1" {
		fmt.Printf("Got invalid response: %+v\n", retVal)
		t.Errorf("Got invalid response")
	}
}

func TestLockAcquireRelease(t *testing.T) {
	// Create a lock
	lock1, err := cStore.NewLock("master", "hostname1", 10)
	lock2, err := cStore.NewLock("master", "hostname2", 10)

	// Acquire the lock
	err = lock1.Acquire(0)
	if err != nil {
		t.Errorf("Error acquiring lock1")
	}
	err = lock2.Acquire(0)
	if err != nil {
		t.Errorf("Error acquiring lock2")
	}

	cnt := 1
	for {
		select {
		case event := <-lock1.EventChan():
			fmt.Printf("Event on Lock1: %+v\n\n", event)
			if event.EventType == api.LockAcquired {
				fmt.Printf("Master lock acquired by Lock1\n")
			}
		case event := <-lock2.EventChan():
			fmt.Printf("Event on Lock2: %+v\n\n", event)
			if event.EventType == api.LockAcquired {
				fmt.Printf("Master lock acquired by Lock2\n")
			}
		case <-time.After(time.Second * time.Duration(30)):
			if cnt == 1 {
				fmt.Printf("10sec timer. releasing Lock1\n\n")
				// At this point, lock1 should be holding the lock
				if !lock1.IsAcquired() {
					t.Errorf("Lock1 failed to acquire lock\n\n")
				}
				lock1.Release()
				cnt++
			} else {
				fmt.Printf("20sec timer. releasing Lock2\n\n")

				// At this point, lock1 should be holding the lock
				if !lock2.IsAcquired() {
					t.Errorf("Lock2 failed to acquire lock\n\n")
				}

				// we are done with the test
				lock2.Release()

				return
			}
		}
	}
}

func TestLockAcquireTimeout(t *testing.T) {
	fmt.Printf("\n\n\n\n\n\n =========================================================== \n\n\n\n\n")
	// Create a lock
	lock1, err := cStore.NewLock("master", "hostname1", 10)
	lock2, err := cStore.NewLock("master", "hostname2", 10)

	// Acquire the lock
	err = lock1.Acquire(0)
	if err != nil {
		t.Errorf("Error acquiring lock1")
	}
	err = lock2.Acquire(20)
	if err != nil {
		t.Errorf("Error acquiring lock2")
	}

	for {
		select {
		case event := <-lock1.EventChan():
			fmt.Printf("Event on Lock1: %+v\n\n", event)
			if event.EventType == api.LockAcquired {
				fmt.Printf("Master lock acquired by Lock1\n")
			}
		case event := <-lock2.EventChan():
			fmt.Printf("Event on Lock2: %+v\n\n", event)
			if event.EventType != api.LockAcquireTimeout {
				fmt.Printf("Invalid event on Lock2\n")
			} else {
				fmt.Printf("Lock2 timeout as expected")
			}
		case <-time.After(time.Second * time.Duration(40)):
			fmt.Printf("40sec timer. releasing Lock1\n\n")
			// At this point, lock1 should be holding the lock
			if !lock1.IsAcquired() {
				t.Errorf("Lock1 failed to acquire lock\n\n")
			}
			lock1.Release()

			time.Sleep(time.Second * 3)

			return
		}
	}
}

func TestServiceRegister(t *testing.T) {
	// Service info
	service1Info := api.ServiceInfo{"athena", "10.10.10.10", 4567}
	service2Info := api.ServiceInfo{"athena", "10.10.10.10", 4568}

	// register it
	err := cStore.RegisterService(service1Info)
	if err != nil {
		t.Errorf("Error registering service. Err: %+v\n", err)
	}
	glog.Infof("Registered service: %+v", service1Info)

	err = cStore.RegisterService(service2Info)
	if err != nil {
		t.Errorf("Error registering service. Err: %+v\n", err)
	}
	glog.Infof("Registered service: %+v", service2Info)

	resp, err := cStore.GetService("athena")
	if err != nil {
		t.Errorf("Error getting service. Err: %+v\n", err)
	}

	glog.Infof("Got service list: %+v\n", resp)

	if (len(resp) < 2) || (resp[0] != service1Info) || (resp[1] != service2Info) {
		t.Errorf("Resp service list did not match input")
	}

	time.Sleep(time.Second * 90)
}

func TestServiceDeregister(t *testing.T) {
	// Service info
	service1Info := api.ServiceInfo{"athena", "10.10.10.10", 4567}
	service2Info := api.ServiceInfo{"athena", "10.10.10.10", 4568}

	// register it
	err := cStore.DeregisterService(service1Info)
	if err != nil {
		t.Errorf("Error deregistering service. Err: %+v\n", err)
	}
	err = cStore.DeregisterService(service2Info)
	if err != nil {
		t.Errorf("Error deregistering service. Err: %+v\n", err)
	}

	time.Sleep(time.Second * 10)
}

func TestServiceWatch(t *testing.T) {
	service1Info := api.ServiceInfo{"athena", "10.10.10.10", 4567}

	// register it
	err := cStore.RegisterService(service1Info)
	if err != nil {
		t.Errorf("Error registering service. Err: %+v\n", err)
	}
	glog.Infof("Registered service: %+v", service1Info)

	// Create event channel
	eventChan := make(chan api.WatchServiceEvent, 1)
	stopChan := make(chan bool, 1)

	// Start watching for service
	err = cStore.WatchService("athena", eventChan, stopChan)
	if err != nil {
		t.Errorf("Error watching service. Err %v", err)
	}

	cnt := 1
	for {
		select {
		case srvEvent := <-eventChan:
			glog.Infof("\n----\nReceived event: %+v\n----", srvEvent)
		case <-time.After(time.Second * time.Duration(10)):
			service2Info := api.ServiceInfo{"athena", "10.10.10.11", 4567}
			if cnt == 1 {
				// register it
				err := cStore.RegisterService(service2Info)
				if err != nil {
					t.Errorf("Error registering service. Err: %+v\n", err)
				}
				glog.Infof("Registered service: %+v", service2Info)
			} else if cnt == 5 {
				// deregister it
				err := cStore.DeregisterService(service2Info)
				if err != nil {
					t.Errorf("Error deregistering service. Err: %+v\n", err)
				}
				glog.Infof("Dregistered service: %+v", service2Info)
			} else if cnt == 7 {
				// Stop the watch
				stopChan <- true

				// wait a little and exit
				time.Sleep(time.Second)

				return
			}
			cnt++
		}
	}
}
