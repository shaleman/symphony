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
package scheduler

import (
	"errors"
	"sort"

	log "github.com/Sirupsen/logrus"
	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/rsrcMgr"
	"github.com/contiv/symphony/zeus/nodeCtrler"
)

// ****************** Bin packing scheduler implementation *****************

type binPackSched struct {
	dummy string
}

// Return a new bin packing scheduler
func NewBinPackScheduler() SchedulerIntf {
	return new(binPackSched)
}

// This is the bin packing scheduler. This is roughly how it works
// 1. Get a list of nodes that match filter criteria
// 2. Sort them based on host address
// 3. Find the first node that has the free resources
func (self *binPackSched) ScheduleAlta(spec *altaspec.AltaSpec) (string, error) {
	reqCpu := float64(spec.NumCpu)
	reqMem := float64(spec.Memory)

	// Get the list of providers
	cpuProviders := rsrcMgr.ListProviders("cpu")
	memProviders := rsrcMgr.ListProviders("memory")

	// Check if we have any resource providers at all
	if (cpuProviders == nil) || (memProviders == nil) {
		return "", errors.New("No nodes to schedule")
	}

	// Get a list of nodes that match the filter
	nodeList := nodeCtrler.FilterNodes(spec.SchedPolicy.Filters)

	// See if any node matched the filter
	if len(nodeList) == 0 {
		return "", errors.New("No nodes that match the filter")
	}

	// Sort the nodes
	sort.Strings(nodeList)

	// Find the first provider with enough resource
	for _, nodeAddr := range nodeList {
		if (cpuProviders[nodeAddr] != nil) && (memProviders[nodeAddr] != nil) {
			cpuProvider := cpuProviders[nodeAddr]
			memProvider := memProviders[nodeAddr]

			// See if it has enough resources
			if (cpuProvider.FreeRsrc >= reqCpu) && (memProvider.FreeRsrc >= reqMem) {
				// We can use this provider. reserve the resource
				log.Infof("Picking node %s for Alta: %s", memProvider.Provider, spec.AltaId)

				// resource list
				rsrcList := []rsrcMgr.ResourceUse{
					{
						Type:     "cpu",
						Provider: cpuProvider.Provider,
						UserKey:  spec.AltaId,
						NumRsrc:  reqCpu,
					},
					{
						Type:     "memory",
						Provider: memProvider.Provider,
						UserKey:  spec.AltaId,
						NumRsrc:  reqMem,
					},
				}

				// Allocate the resource
				_, err := rsrcMgr.AllocResources(rsrcList)
				if err != nil {
					log.Errorf("Error allocating cpu/mem resource. Err: %v", err)
					return "", errors.New("Error allocating resource")
				}

				// Finally return the provider's name
				return memProvider.Provider, nil
			}
		}
	}

	return "", errors.New("No Nodes with resource")
}
