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
	"github.com/contiv/symphony/pkg/altaspec"

	log "github.com/Sirupsen/logrus"
)

// Responsible for scheduling Alta containers on a node
// Scheduler is designed to support multiple scheduling policies
// with default being "leastUsed"

// Define the scheduler interface
type SchedulerIntf interface {
	ScheduleAlta(spec *altaspec.AltaSpec) (string, error)
}

// DB of supported scheduler
var schedulers map[string]SchedulerIntf = make(map[string]SchedulerIntf)
var defaultScheduler SchedulerIntf

// Initialize all known schedulers
func Init() {
	// Initialize each scheduler type
	schedulers["leastUsed"] = NewLeastUsedScheduler()
	schedulers["binPack"] = NewBinPackScheduler()

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
