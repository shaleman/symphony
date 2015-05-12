package libfsm

import (
    "github.com/golang/glog"
    )
// Finite state machines
// Library to implement FSMs.
// - Each FSM object runs in its own goroutine
// - State of the FSM and any subclass inherited from fsm class are written
//   automatically to a distributed datastore like etcd/consul
// - Events are queued to FSM using the event channel and processed in order

// Main FSM structure
type Fsm struct {
    transitions     *FsmTable   // FSM transition table
    FsmState        string      // FSM's current state
}

// FSM event
type Event struct {
    EventName   string          // Name of the event
    EventData   interface{}     // Event specific data
}

// Callback function type
type CallbackFunc func(Event) error

// FSM Transition entry
type Transition struct {
    CurrState       string
    EventName       string
    NewState        string
    Callback        CallbackFunc
}

type FsmTable []Transition


// Create a new Fsm
func NewFsm(fsmTable *FsmTable, initState string) *Fsm {
    fsm := new(Fsm)

    fsm.transitions = fsmTable
    fsm.FsmState = initState

    return fsm
}

// Handle a new event for the fsm
func (self *Fsm) FsmEvent(event Event) {
    glog.Infof("Processing event %s in state %s", event.EventName, self.FsmState)

    // find the <currState,event> pair in the transition table
    for _, trans := range *self.transitions {
        if ((trans.CurrState == self.FsmState) && (trans.EventName == event.EventName)) {
            err := trans.Callback(event)
            if (err != nil) {
                glog.Errorf("Processing event %s failed in state %s", event.EventName, self.FsmState)

                return
            } else {
                if (self.FsmState != trans.NewState) {
                    glog.Infof("Transitioning to state %s", trans.NewState)
                    self.FsmState = trans.NewState
                }

                return
            }
        }
    }

    // If we reached here, we did not find a valid transition
    glog.Errorf("Invalid event %s in state %s", event.EventName, self.FsmState)

    return
}
