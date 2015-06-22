package etcdClient

import (
	"time"

	api "github.com/contiv/symphony/pkg/confStore/confStoreApi"

	"github.com/coreos/go-etcd/etcd"
	"github.com/golang/glog"
)

// Etcd error codes
const EtcdErrorCodeNotFound = 100
const EtcdErrorCodeKeyExists = 105

// Lock object
type Lock struct {
	name          string
	myId          string
	isAcquired    bool
	isReleased    bool
	holderId      string
	ttl           uint64
	timeout       uint64
	modifiedIndex uint64
	eventChan     chan api.LockEvent
	stopChan      chan bool
	watchCh       chan *etcd.Response
	watchStopCh   chan bool
	client        *etcd.Client
}

// Create a new lock
func (self *EtcdPlugin) NewLock(name string, myId string, ttl uint64) (api.LockInterface, error) {
	// Create a lock
	lock := new(Lock)

	// Initialize the lock
	lock.name = name
	lock.myId = myId
	lock.ttl = ttl
	lock.client = self.client

	// Create channels
	lock.eventChan = make(chan api.LockEvent, 1)
	lock.stopChan = make(chan bool, 1)

	// Setup some channels for watch
	lock.watchCh = make(chan *etcd.Response, 1)
	lock.watchStopCh = make(chan bool, 1)

	return lock, nil
}

// Acquire a lock
func (self *Lock) Acquire(timeout uint64) error {
	self.timeout = timeout

	// Acquire in background
	go self.acquireLock()

	return nil
}

// Release a lock
func (self *Lock) Release() error {
	keyName := "/contiv.io/lock/" + self.name

	// Mark this as released
	self.isReleased = true

	// Send stop signal on stop channel
	self.stopChan <- true

	// If the lock was acquired, release it
	if self.isAcquired {
		// Update TTL on the lock
		resp, err := self.client.CompareAndDelete(keyName, self.myId, self.modifiedIndex)
		if err != nil {
			glog.Errorf("Error Deleting key. Err: %v", err)
		} else {
			glog.Infof("Deleted key lock %s, Resp: %+v", keyName, resp)

			// Update modifiedIndex
			self.modifiedIndex = resp.Node.ModifiedIndex
		}
	}

	return nil
}

// Note: This is for debug/test purposes only
// Stop a lock without releasing it.
// Let the etcd TTL expiry release it
func (self *Lock) Kill() error {
	// Mark this as released
	self.isReleased = true

	// Send stop signal on stop channel
	self.stopChan <- true

	return nil
}

// Return event channel
func (self *Lock) EventChan() <-chan api.LockEvent {
	return self.eventChan
}

// Check if the lock is acquired
func (self *Lock) IsAcquired() bool {
	return self.isAcquired
}

// Get current lock holder's Id
func (self *Lock) GetHolder() string {
	return self.holderId
}

// *********************** Internal functions *************
// Try acquiring a lock.
// This assumes its called in its own go routine
func (self *Lock) acquireLock() {
	keyName := "/contiv.io/lock/" + self.name

	// Start a watch on the lock first so that we dont loose any notifications
	go self.watchLock()

	// Wait in this loop forever till lock times out or released
	for {
		glog.Infof("Getting the lock %s to see if its acquired", keyName)
		// Get the key and see if we or someone else has already acquired the lock
		resp, err := self.client.Get(keyName, false, false)
		if err != nil {
			if err.(*etcd.EtcdError).ErrorCode != EtcdErrorCodeNotFound {
				glog.Errorf("Error getting the key %s. Err: %v", keyName, err)
			} else {
				glog.Infof("Lock %s does not exist. trying to acquire it", keyName)
			}

			// Try to acquire the lock
			resp, err := self.client.Create(keyName, self.myId, self.ttl)
			if err != nil {
				if err.(*etcd.EtcdError).ErrorCode != EtcdErrorCodeKeyExists {
					glog.Errorf("Error creating key %s. Err: %v", keyName, err)
				} else {
					glog.Infof("Lock %s acquired by someone else", keyName)
				}
			} else {
				glog.Infof("Acquired lock %s. Resp: %#v, Node: %+v", keyName, resp, resp.Node)

				// Successfully acquired the lock
				self.isAcquired = true
				self.holderId = self.myId
				self.modifiedIndex = resp.Node.ModifiedIndex

				// Send acquired message to event channel
				self.eventChan <- api.LockEvent{EventType: api.LockAcquired}

				// refresh it
				self.refreshLock()

				// If lock is released, we are done, else go back and try to acquire it
				if self.isReleased {
					return
				}
			}
		} else if resp.Node.Value == self.myId {
			glog.Infof("Already Acquired key %s. Resp: %#v, Node: %+v", keyName, resp, resp.Node)

			// We have already acquired the lock. just keep refreshing it
			self.isAcquired = true
			self.holderId = self.myId
			self.modifiedIndex = resp.Node.ModifiedIndex

			// Send acquired message to event channel
			self.eventChan <- api.LockEvent{EventType: api.LockAcquired}

			// Refresh lock
			self.refreshLock()

			// If lock is released, we are done, else go back and try to acquire it
			if self.isReleased {
				return
			}
		} else if resp.Node.Value != self.myId {
			glog.Infof("Lock already acquired by someone else. Resp: %+v, Node: %+v", resp, resp.Node)

			// Set the current holder's Id
			self.holderId = resp.Node.Value

			// Wait for changes on the lock
			self.waitForLock()

			if self.isReleased {
				return
			}
		}
	}
}

// We couldnt acquire lock, Wait for changes on the lock
func (self *Lock) waitForLock() {
	// If timeout is not specified, set it to high value
	timeoutIntvl := time.Second * time.Duration(20000)
	if self.timeout != 0 {
		timeoutIntvl = time.Second * time.Duration(self.timeout)
	}

	glog.Infof("Waiting to acquire lock (%s/%s)", self.name, self.myId)

	// Create a timer
	timer := time.NewTimer(timeoutIntvl)
	defer timer.Stop()

	// Wait for changes
	for {
		// wait on watch channel for holder to release the lock
		select {
		case <-timer.C:
			if self.timeout != 0 {
				glog.Infof("Lock timeout on lock %s/%s", self.name, self.myId)

				self.eventChan <- api.LockEvent{EventType: api.LockAcquireTimeout}

				glog.Infof("Lock acquire timed out. Stopping lock")

				self.watchStopCh <- true

				// Release the lock
				self.Release()

				return
			}
		case watchResp := <-self.watchCh:
			if watchResp != nil {
				glog.V(2).Infof("Received watch notification(%s/%s): %+v", self.name, self.myId, watchResp)

				if watchResp.Action == "expire" || watchResp.Action == "delete" ||
					watchResp.Action == "compareAndDelete" {
					glog.Infof("Retrying to acquire lock")
					return
				}
			}
		case <-self.stopChan:
			glog.Infof("Stopping lock")
			self.watchStopCh <- true

			return
		}
	}
}

// Refresh lock
func (self *Lock) refreshLock() {
	// Refresh interval is 40% of TTL
	refreshIntvl := time.Second * time.Duration(self.ttl*3/10)
	keyName := "/contiv.io/lock/" + self.name

	// Create a timer
	// refTimer := time.NewTimer(refreshIntvl)
	// defer refTimer.Stop()

	// Loop forever
	for {
		select {
		case <-time.After(refreshIntvl):
			// Update TTL on the lock
			resp, err := self.client.CompareAndSwap(keyName, self.myId, self.ttl,
				self.myId, self.modifiedIndex)
			if err != nil {
				glog.Errorf("Error updating TTl. Err: %v", err)

				// We are not master anymore
				self.isAcquired = false

				// Send lock lost event
				self.eventChan <- api.LockEvent{EventType: api.LockLost}

				// FIXME: trigger a lock lost event
				return
			} else {
				glog.V(2).Infof("Refreshed TTL on lock %s, Resp: %+v", keyName, resp)

				// Update modifiedIndex
				self.modifiedIndex = resp.Node.ModifiedIndex
			}
		case watchResp := <-self.watchCh:
			// Since we already acquired the lock, nothing to do here
			// FIXME: see if we lost the lock
			if watchResp != nil {
				glog.V(2).Infof("Received watch notification for(%s/%s): %+v",
					self.name, self.myId, watchResp)
			}
		case <-self.stopChan:
			glog.Infof("Stopping lock")
			self.watchStopCh <- true
			return
		}
	}
}

// Watch for changes on the lock
func (self *Lock) watchLock() {
	keyName := "/contiv.io/lock/" + self.name

	for {
		resp, err := self.client.Watch(keyName, 0, false, self.watchCh, self.watchStopCh)
		if err != nil {
			if err != etcd.ErrWatchStoppedByUser {
				glog.Errorf("Error watching the key %s, Err %v", keyName, err)
			} else {
				glog.Infof("Watch stopped for lock %s", keyName)
			}
		} else {
			glog.Infof("Got Watch Resp: %+v", resp)
		}

		// If the lock is released, we are done
		if self.isReleased {
			return
		}

		// Wait for a second and go back to watching
		time.Sleep(1 * time.Second)
	}
}
