package common

// Common defenitions to be used across zeus modules

import (
	"github.com/contiv/symphony/pkg/altaspec"
)
type AltaCtrlInterface interface {
	CreateAlta(altaConfig *altaspec.AltaConfig) error
	RestoreAltaActors() error
	ListAlta() []*AltaState
	DiffNodeAltaLList(nodeAddr string, altaList []altaspec.AltaContext) error
}

// State of alta container
type AltaState struct {
	Spec     	altaspec.AltaSpec // Spec for the container
	CurrNode 	string            // Node where this container is placed
	ContainerId string			  // ContainerId on current node
	FsmState    string       	  // FSM for the container
}

type ZeusCtrlers struct {
	AltaCtrler  AltaCtrlInterface
}
