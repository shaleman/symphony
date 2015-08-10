/***
Copyright 2014 Cisco Systems Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package common

// Common defenitions to be used across zeus modules

import (
	"github.com/contiv/symphony/pkg/altaspec"
)

type AltaCtrlInterface interface {
	CreateAlta(altaConfig *altaspec.AltaConfig) error
	RestoreAltaActors() error
	ListAlta() []*AltaState
	ReconcileNode(nodeAddr string, altaList []altaspec.AltaContext) error
	NodeDownEvent(nodeAddr string) error
}

// State of alta container
type AltaState struct {
	Spec        altaspec.AltaSpec // Spec for the container
	CurrNode    string            // Node where this container is placed
	ContainerId string            // ContainerId on current node
	FsmState    string            // FSM for the container
}

type ZeusCtrlers struct {
	AltaCtrler AltaCtrlInterface
}
