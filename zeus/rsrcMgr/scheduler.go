package rsrcMgr

import (
	"errors"

	"github.com/contiv/symphony/pkg/altaspec"

	log "github.com/Sirupsen/logrus"
)

// Responsible for scheduling Alta containers on a node
// Scheduler is designed to support multiple scheduling policies
// with default being "leastUsed"

// Define the scheduler interface
type SchedulerIntf interface {
	GetNodeForAlta(spec *altaspec.AltaSpec) (string, error)
}

// DB of supported scheduler
var schedulers map[string]SchedulerIntf = make(map[string]SchedulerIntf)
var defaultScheduler SchedulerIntf

// Initialize all known schedulers
func initSchedulers() {
	// Initialize each scheduler type
	schedulers["leastUsed"] = newLeastUsedScheduler()

	// set the default
	defaultScheduler = schedulers["leastUsed"]
}

// Return a scheduler instance
func Scheduler(schedPolicy string) SchedulerIntf {
	// Return default if a policy wasnt specified
	if (schedPolicy == "") || (schedPolicy == "default") {
		return defaultScheduler
	}

	// Return the scheduler specified
	if schedulers[schedPolicy] != nil {
		return schedulers[schedPolicy]
	}

	// Error case
	log.Fatal("Scheduler policy not found")
	return nil
}

// ****************** Least Used scheduler implementation *****************

type leastUsedSched struct {
	dummy string
}

// Return a new least used scheduler
func newLeastUsedScheduler() SchedulerIntf {
	return new(leastUsedSched)
}

// Get the least used node for the alta container
// This searches for nodes with least used resources in following order
//   1. Least used CPU
//   2. Least used memory
// When all are equal, it returns the first one
func (self *leastUsedSched) GetNodeForAlta(spec *altaspec.AltaSpec) (string, error) {
	reqCpu := float64(spec.NumCpu)
	reqMem := float64(spec.Memory)
	var maxFreeRsrc float64 = 0

	// Determine the max free cpu resource
	for _, provider := range rsrcMgr.rsrcDb["cpu"].Providers {
		if (provider.FreeRsrc >= reqCpu) && (provider.FreeRsrc > maxFreeRsrc) {
			maxFreeRsrc = provider.FreeRsrc
		}
	}

	// Create a sublist of least used cpu providers
	var cpuProviderList []*RsrcProvider
	for _, provider := range rsrcMgr.rsrcDb["cpu"].Providers {
		if (provider.FreeRsrc >= reqCpu) && (provider.FreeRsrc >= maxFreeRsrc) {
			cpuProviderList = append(cpuProviderList, provider)
		}
	}

	log.Infof("Cpu provider list: %+v", cpuProviderList)

	// Check who has the least used memory
	var maxFreeMem float64 = 0
	for _, cpuProvider := range cpuProviderList {
		memProvider := rsrcMgr.rsrcDb["memory"].Providers[cpuProvider.Provider]
		if (memProvider.FreeRsrc >= reqMem) && (memProvider.FreeRsrc > maxFreeMem) {
			maxFreeMem = memProvider.FreeRsrc
		}
	}

	// Return the first lowest used memory provider
	for _, cpuProvider := range cpuProviderList {
		memProvider := rsrcMgr.rsrcDb["memory"].Providers[cpuProvider.Provider]
		if (memProvider.FreeRsrc >= reqMem) && (memProvider.FreeRsrc >= maxFreeMem) {
			// We can use this provider. reserve the resource
			log.Infof("Picking node %s for Alta: %s", memProvider.Provider, spec.AltaId)

			// resource list
			rsrcList := []ResourceUse{
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
			_, err := AllocResources(rsrcList)
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
