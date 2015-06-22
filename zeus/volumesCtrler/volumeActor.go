package volumesCtrler

import (
	//"time"
	"errors"
	//"strings"
	//"io/ioutil"
	//"strconv"
	//"net/http"
	//"encoding/json"

	"github.com/contiv/symphony/zeus/nodeCtrler"

	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/libfsm"

	log "github.com/Sirupsen/logrus"
)

// Volume model to be persisted
type VolumeModel struct {
	Spec     altaspec.AltaVolumeSpec // Volume specification
	CurrNode string                  // Node where its mounted
	Fsm      *libfsm.Fsm             // FSM state of this actor
}

// State of the actor
type VolumeActor struct {
	Model     VolumeModel       // Model for this actor
	EventChan chan libfsm.Event // Event channel
}

// Create a new volume actor
func NewVolumeActor(volumeSpec altaspec.AltaVolumeSpec) (*VolumeActor, error) {
	volume := new(VolumeActor)

	// Initialize
	volume.Model.Spec = volumeSpec

	// Create the FSM
	volume.Model.Fsm = libfsm.NewFsm(&libfsm.FsmTable{
		// currentState,  event,      newState,   callback
		{"created", "init", "inited", func(e libfsm.Event) error { return volume.createVolume(e) }},
		{"inited", "mount", "mounted", func(e libfsm.Event) error { return volume.mountVolume(e) }},
		{"mounted", "unmount", "inited", func(e libfsm.Event) error { return volume.unmountVolume() }},
		{"inited", "destroy", "destroyed", func(e libfsm.Event) error { return volume.deleteVolume() }},
	}, "created")

	// create the channel
	volume.EventChan = make(chan libfsm.Event, 200)

	// Kick off the runloop
	go volume.runLoop()

	return volume, nil
}

// run loop waiting for events to process
func (self *VolumeActor) runLoop() {
	for {
		select {
		case event := <-self.EventChan:
			self.Model.Fsm.FsmEvent(event)
		}

		// Persist the model after each event
		self.saveModel()
	}
}

// Queue an event to the alta container
func (self *VolumeActor) VolumeEvent(event libfsm.Event) {
	self.EventChan <- event
}

// Create and initialize a volume
// FIXME: we need a retry mechanism for volume creation when it fails
func (self *VolumeActor) createVolume(e libfsm.Event) error {
	nodeAddr := e.EventData.(string)
	var resp altaspec.ReqSuccess
	urlPath := "/volume/create"

	// Ask the node to create the volume
	err := nodeCtrler.NodePostReq(nodeAddr, urlPath, self.Model.Spec, &resp)
	if err != nil {
		log.Errorf("Error creating volume %v, Err: %v", self.Model.Spec, err)
		return err
	}

	if !resp.Success {
		log.Errorf("Failure response while creating volume: %v. Err: %v", self.Model.Spec, err)
		return errors.New("Volume create failed")
	}

	return nil
}

// Mount the volume on a node
func (self *VolumeActor) mountVolume(e libfsm.Event) error {
	nodeAddr := e.EventData.(string)
	var resp altaspec.ReqSuccess
	urlPath := "/volume/mount"

	// Ask the node to create the volume
	err := nodeCtrler.NodePostReq(nodeAddr, urlPath, self.Model.Spec, &resp)
	if err != nil {
		log.Errorf("Error mounting volume %v, Err: %v", self.Model.Spec, err)
		return err
	}

	if !resp.Success {
		log.Errorf("Failure response while mounting volume: %v. Err: %v", self.Model.Spec, err)
		return errors.New("Volume mount failed")
	}

	// Save the current node where volume is mounted
	self.Model.CurrNode = nodeAddr

	return nil
}

// Unmount the volume on a node
func (self *VolumeActor) unmountVolume() error {
	var resp altaspec.ReqSuccess
	urlPath := "/volume/unmount"

	// Ask the node to create the volume
	err := nodeCtrler.NodePostReq(self.Model.CurrNode, urlPath, self.Model.Spec, &resp)
	if err != nil {
		log.Errorf("Error unmounting volume %v, Err: %v", self.Model.Spec, err)
		return err
	}

	if !resp.Success {
		log.Errorf("Failure response while unmounting volume: %v. Err: %v", self.Model.Spec, err)
		return errors.New("Volume unmount failed")
	}

	return nil
}

// Delete a volume
func (self *VolumeActor) deleteVolume() error {
	return nil
}

// Save the actor model
func (self *VolumeActor) saveModel() error {
	volumeKey := self.Model.Spec.DatastoreType + ":" + self.Model.Spec.DatastoreVolumeId
	storeKey := "volume/" + volumeKey

	// Save it to conf store
	err := ctrler.cStore.SetObj(storeKey, self.Model)
	if err != nil {
		log.Errorf("Error storing object %+v. Err: %v", self.Model, err)
		return err
	}

	return nil
}
