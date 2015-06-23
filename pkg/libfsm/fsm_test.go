package libfsm

import (
	"testing"

	log "github.com/Sirupsen/logrus"
)

// Desired API

type TestFsm struct {
	Fsm       *Fsm   // FSM for this object
	TestState string // State associated with this FSM instance
}

// Constructor for test FSM
func NewTestFsm(testState string) *TestFsm {
	testFsm := new(TestFsm)

	testFsm.TestState = testState

	// Initialize the FSM
	testFsm.Fsm = NewFsm(&FsmTable{
		// currentState,  event,      newState,   callback
		{"created", "start", "started",
			func(e Event) error { return testFsm.startTestFsm(e) }},

		{"started", "stop", "stopped",
			func(e Event) error { return testFsm.stopTestFsm(e) }},

		{"stopped", "start", "started",
			func(e Event) error { return testFsm.startTestFsm(e) }},
	}, "created")

	return testFsm
}

func (self *TestFsm) startTestFsm(event Event) error {
	self.TestState = "started state"

	return nil
}

func (self *TestFsm) stopTestFsm(event Event) error {
	self.TestState = "stopped state"

	return nil
}

// Test a simple FSM transition
func TestFsmTransition(t *testing.T) {
	// Create fsm
	testFsm := NewTestFsm("created state")

	// Queue an event
	testFsm.Fsm.FsmEvent(Event{"start", nil})

	log.Infof("TestFsm: %#v", testFsm)

	if testFsm.TestState != "started state" {
		t.Errorf("FSM event failed")
	}

	// Queue an event
	testFsm.Fsm.FsmEvent(Event{"stop", nil})

	log.Infof("TestFsm: %#v", testFsm)

	if testFsm.TestState != "stopped state" {
		t.Errorf("FSM event failed")
	}
}
