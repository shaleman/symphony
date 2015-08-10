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

	log "github.com/Sirupsen/logrus"
	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/rsrcMgr"
	"github.com/contiv/symphony/zeus/nodeCtrler"
)

// ****************** Least Used scheduler implementation *****************

type leastUsedSched struct {
	dummy string
}

// Return a new least used scheduler
func NewLeastUsedScheduler() SchedulerIntf {
	return new(leastUsedSched)
}

// Get the least used node for the alta container
// First it applies the filter to get a short list of possible nodes
// Then it searches for nodes with least used resources in following order
//   1. Least used CPU
//   2. Least used memory
// When all are equal, it returns the first one
func (self *leastUsedSched) ScheduleAlta(spec *altaspec.AltaSpec) (string, error) {
	reqCpu := float64(spec.NumCpu)
	reqMem := float64(spec.Memory)
	var maxFreeRsrc float64 = 0

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

	// filter out cpu providers
	var cpuFilterList []*rsrcMgr.RsrcProvider
	for _, nodeAddr := range nodeList {
		if cpuProviders[nodeAddr] != nil {
			cpuFilterList = append(cpuFilterList, cpuProviders[nodeAddr])
		}
	}
	// Determine the max free cpu resource
	for _, provider := range cpuFilterList {
		if (provider.FreeRsrc >= reqCpu) && (provider.FreeRsrc > maxFreeRsrc) {
			maxFreeRsrc = provider.FreeRsrc
		}
	}

	// Create a sublist of least used cpu providers
	var cpuProviderList []*rsrcMgr.RsrcProvider
	for _, provider := range cpuFilterList {
		if (provider.FreeRsrc >= reqCpu) && (provider.FreeRsrc >= maxFreeRsrc) {
			cpuProviderList = append(cpuProviderList, provider)
		}
	}

	log.Infof("Cpu provider list: %+v", cpuProviderList)

	// Check who has the least used memory
	var maxFreeMem float64 = 0
	for _, cpuProvider := range cpuProviderList {
		memProvider := memProviders[cpuProvider.Provider]
		if (memProvider.FreeRsrc >= reqMem) && (memProvider.FreeRsrc > maxFreeMem) {
			maxFreeMem = memProvider.FreeRsrc
		}
	}

	// Return the first lowest used memory provider
	for _, cpuProvider := range cpuProviderList {
		memProvider := memProviders[cpuProvider.Provider]
		if (memProvider.FreeRsrc >= reqMem) && (memProvider.FreeRsrc >= maxFreeMem) {
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

	return "", errors.New("No Nodes with resource")
}
