package altaCtrler

import (
	"time"

	"github.com/contiv/symphony/zeus/nodeCtrler"
	"github.com/contiv/symphony/zeus/rsrcMgr"
	"github.com/contiv/symphony/zeus/volumesCtrler"

	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/libfsm"

	log "github.com/Sirupsen/logrus"
)

// Model to be persisted
type AltaModel struct {
	Spec     	altaspec.AltaSpec // Spec for the container
	CurrNode 	string            // Node where this container is placed
	ContainerId string			  // ContainerId on current node
	Fsm      	*libfsm.Fsm       // FSM for the container
}

// State of Alta container
type AltaActor struct {
	AltaId    string            // Unique Id for the container
	Model     AltaModel         // State of the alta container
	EventChan chan libfsm.Event // Event queue
	ticker    *time.Ticker      // DEBUG: ticker to print state
}

// Create a new Alta container
func NewAlta(altaSpec *altaspec.AltaSpec) (*AltaActor, error) {
	alta := new(AltaActor)

	// initialize
	alta.AltaId = altaSpec.AltaId
	alta.Model.Spec = *altaSpec

	// Start alta FSM in its own go routine
	// FSM for the node
	alta.Model.Fsm = libfsm.NewFsm(&libfsm.FsmTable{
		// currentState,  event,      newState,   callback
		{"created", "schedule", "scheduled", func(e libfsm.Event) error { return alta.scheduleAlta() }},
		{"scheduled", "createNet", "waitNet", func(e libfsm.Event) error { return alta.createNetwork() }},
		{"waitNet", "createVol", "waitVol", func(e libfsm.Event) error { return alta.mountVolume() }},
		{"waitVol", "pullImg", "waitImg", func(e libfsm.Event) error { return alta.pullImg() }},
		{"waitImg", "imgReady", "starting", func(e libfsm.Event) error { return alta.createAltaCntr() }},
		{"starting", "start", "running", func(e libfsm.Event) error { return alta.startAltaCntr() }},
		{"running", "failure", "failed", func(e libfsm.Event) error { return nil }},
		{"running", "stop", "stopped", func(e libfsm.Event) error { return alta.stopAltaCntr() }},
		{"failed", "restart", "running", func(e libfsm.Event) error { return alta.startAltaCntr() }},
		{"stopped", "start", "running", func(e libfsm.Event) error { return alta.startAltaCntr() }},
	}, "created")

	// create the channel
	alta.EventChan = make(chan libfsm.Event, 200)

	// Kick off the alta runloop
	go alta.runLoop()

	// Debug: timer to print state periodically
	alta.ticker = time.NewTicker(time.Second * 15)

	log.Infof("Created Alta: %#v", alta)

	return alta, nil
}

// Main run loop for the alta container.
// Wait in the event loop for an event
func (self *AltaActor) runLoop() {
	for {
		select {
		case event := <-self.EventChan:
			self.Model.Fsm.FsmEvent(event)

			// Save state after each transition
			self.saveModel()
		case <-self.ticker.C:
			// FIXME: Use this timer to perform retries when things fail
			log.Debugf("Alta: %s, FSM state: %s, state: %#v", self.Model.Spec.AltaName,
				self.Model.Fsm.FsmState, self)
		}
	}
}

// Queue an event to the alta container
func (self *AltaActor) AltaEvent(eventName string) {
	self.EventChan <- libfsm.Event{eventName, nil}
}

// ****************** FSM event handlers ***************
// Schedule the container to one of the nodes
func (self *AltaActor) scheduleAlta() error {
	// Ask the scheduler to assign a node
	nodeAddr, err := rsrcMgr.Scheduler("default").GetNodeForAlta(&self.Model.Spec)
	if err != nil {
		log.Errorf("Failed to schedule node. Error: %v", err)
		return err
	}

	// Save the current node
	self.Model.CurrNode = nodeAddr

	// Move forward
	self.AltaEvent("createNet")

	return nil
}

// Create networks on the host
func (self *AltaActor) createNetwork() error {

	// FIXME: just Move forward, default network already exists
	self.AltaEvent("createVol")

	return nil
}

// Create volumes for the container
func (self *AltaActor) mountVolume() error {
	// For each volume
	for _, volume := range self.Model.Spec.Volumes {
		log.Infof("Mounting volume: %+v", volume)

		// Mount the volume. create it if it doesnt exist
		err := volumesCtrler.MountVolume(volume, self.Model.CurrNode)
		if err != nil {
			log.Errorf("Error mounting volume. Err: %v", err)
			return err
		}

	}

	// Trigger next event
	self.AltaEvent("pullImg")

	return nil
}

// Pull required image
func (self *AltaActor) pullImg() error {
	log.Infof("Checking if image %s exists on host %s", self.Model.Spec.Image, self.Model.CurrNode)

	// Check if the image exists
	imgPath := "/image/" + self.Model.Spec.Image + "/ispresent"
	var resp altaspec.ReqSuccess
	err := nodeCtrler.NodeGetReq(self.Model.CurrNode, imgPath, &resp)
	if err != nil {
		log.Errorf("Error checking image presence. Err: %v", err)
		return err
	}

	if resp.Success {
		// Image exists move forward
		self.AltaEvent("imgReady")
		return nil
	}

	log.Infof("Pulling image: %s", self.Model.Spec.Image)

	imgPullPath := "/image/" + self.Model.Spec.Image + "/pull"
	dummy := struct{ dummy string }{dummy: "dummy"}
	err = nodeCtrler.NodePostReq(self.Model.CurrNode, imgPullPath, dummy, &resp)
	if err != nil {
		log.Errorf("Error pulling image. Err: %v", err)
		return err
	}

	// Image is ready move forward
	if resp.Success {
		self.AltaEvent("imgReady")
	}
	return nil
}

// Create the container
func (self *AltaActor) createAltaCntr() error {
	log.Infof("Creating container %+v on host %s", self.Model.Spec, self.Model.CurrNode)

	// Create the container
	var resp altaspec.AltaContext
	err := nodeCtrler.NodePostReq(self.Model.CurrNode, "/alta/create", self.Model.Spec, &resp)
	if err != nil {
		log.Errorf("Error creating container. Err: %v", err)
		return err
	}

	log.Infof("Got create response: %+v", resp)

	// Save the container id for later
	self.Model.ContainerId = resp.ContainerId

	self.AltaEvent("start")

	return nil
}

// Start the container
func (self *AltaActor) startAltaCntr() error {
	log.Infof("Starting container %s on host %s", self.AltaId, self.Model.CurrNode)

	startPath := "/alta/" + self.AltaId + "/start"
	dummy := struct{ dummy string }{dummy: "dummy"}
	var resp altaspec.ReqSuccess

	// Start the container
	err := nodeCtrler.NodePostReq(self.Model.CurrNode, startPath, dummy, &resp)
	if err != nil {
		log.Errorf("Error starting container. Err: %v", err)
		return err
	}

	return nil
}

// Stop the container
func (self *AltaActor) stopAltaCntr() error {
	log.Infof("Stopping container %s on host %s", self.AltaId, self.Model.CurrNode)

	stopPath := "/alta/" + self.AltaId + "/stop"
	dummy := struct{ dummy string }{dummy: "dummy"}
	var resp altaspec.ReqSuccess

	// Stop the container
	err := nodeCtrler.NodePostReq(self.Model.CurrNode, stopPath, dummy, &resp)
	if err != nil {
		log.Errorf("Error stopping container. Err: %v", err)
		return err
	}

	return nil
}

// Save alta container state to conf store
func (self *AltaActor) saveModel() error {
	storeKey := "alta/" + self.Model.Spec.AltaId

	// Save it to conf store
	err := altaCtrl.cStore.SetObj(storeKey, self.Model)
	if err != nil {
		log.Errorf("Error storing object %+v. Err: %v", self.Model, err)
		return err
	}

	return nil
}
