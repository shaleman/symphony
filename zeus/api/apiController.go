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

package api

import (
	"errors"
	"github.com/contiv/symphony/zeus/api/contivModel"
	"github.com/contiv/symphony/pkg/confStore/modeldb"

	"github.com/gorilla/mux"
	log "github.com/Sirupsen/logrus"
)

type ApiController struct {
	router		*mux.Router
}

// Create a new controller
func NewApiController(router *mux.Router) *ApiController {
	ctrler := new(ApiController)
	ctrler.router = router

	// initialize the model objects
	contivModel.Init(ctrler)

	// Register routes
	contivModel.AddRoutes(router)

	return ctrler
}

func (self *ApiController) AppCreate(app *contivModel.App) error {
	log.Infof("Received AppCreate: %+v", app)
    return nil
}
func (self *ApiController) AppDelete(app *contivModel.App) error {
	log.Infof("Received AppDelete: %+v", app)
    return nil
}
func (self *ApiController) EndpointGroupCreate(endpointGroup *contivModel.EndpointGroup) error {
	log.Infof("Received EndpointGroupCreate: %+v", endpointGroup)
    return nil
}
func (self *ApiController) EndpointGroupDelete(endpointGroup *contivModel.EndpointGroup) error {
	log.Infof("Received EndpointGroupDelete: %+v", endpointGroup)
    return nil
}
func (self *ApiController) NetworkCreate(network *contivModel.Network) error {
	log.Infof("Received NetworkCreate: %+v", network)

	// Make sure tenant exists
	if network.TenantName == "" {
		return errors.New("Invalid tenant name")
	}

	tenant := contivModel.FindTenant(network.TenantName)
	if tenant == nil {
		return errors.New("Tenant not found")
	}

	// Setup links
	modeldb.AddLink(&network.Links.Tenant, tenant)
	modeldb.AddLinkSet(&tenant.LinkSets.Networks, network)

	// Save the tenant too since we added the links
	err := tenant.Write()
	if err != nil {
		log.Errorf("Error updating tenant state(%+v). Err: %v", tenant, err)
		return err
	}

    return nil
}
func (self *ApiController) NetworkDelete(network *contivModel.Network) error {
	log.Infof("Received NetworkDelete: %+v", network)
    return nil
}
func (self *ApiController) PolicyCreate(policy *contivModel.Policy) error {
	log.Infof("Received PolicyCreate: %+v", policy)
    return nil
}
func (self *ApiController) PolicyDelete(policy *contivModel.Policy) error {
	log.Infof("Received PolicyDelete: %+v", policy)
    return nil
}
func (self *ApiController) ServiceCreate(service *contivModel.Service) error {
	log.Infof("Received ServiceCreate: %+v", service)
    return nil
}
func (self *ApiController) ServiceDelete(service *contivModel.Service) error {
	log.Infof("Received ServiceDelete: %+v", service)
    return nil
}
func (self *ApiController) ServiceInstanceCreate(serviceInstance *contivModel.ServiceInstance) error {
	log.Infof("Received ServiceInstanceCreate: %+v", serviceInstance)
    return nil
}
func (self *ApiController) ServiceInstanceDelete(serviceInstance *contivModel.ServiceInstance) error {
	log.Infof("Received ServiceInstanceDelete: %+v", serviceInstance)
    return nil
}
func (self *ApiController) TenantCreate(tenant *contivModel.Tenant) error {
	log.Infof("Received TenantCreate: %+v", tenant)
    return nil
}
func (self *ApiController) TenantDelete(tenant *contivModel.Tenant) error {
	log.Infof("Received TenantDelete: %+v", tenant)
    return nil
}
func (self *ApiController) VolumeCreate(volume *contivModel.Volume) error {
	log.Infof("Received VolumeCreate: %+v", volume)
    return nil
}
func (self *ApiController) VolumeDelete(volume *contivModel.Volume) error {
	log.Infof("Received VolumeDelete: %+v", volume)
    return nil
}
