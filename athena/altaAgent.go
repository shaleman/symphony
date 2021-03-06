package main

import (
	"errors"

	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/libdocker"

	log "github.com/Sirupsen/logrus"
)

// State of Alta instance
type AltaState struct {
	AltaId       string
	ContainerId  string
	portNames    []string
	spec         altaspec.AltaSpec
	containerCtx *libdocker.ContainerCtx
}

// Database of alta instances
type AltaMgr struct {
	altaDb map[string]*AltaState
}

// Create and initialize new alta manager
func NewAltaMgr() *AltaMgr {
	altaMgr := new(AltaMgr)

	// Initialize the db
	altaMgr.altaDb = make(map[string]*AltaState)

	// FIXME: perform any initialization here

	// Return the alta mgr
	return altaMgr
}

// APIs provided by the alta manager
// FIXME: These APIs are mostly executed out of http request handlers. Each
// http request is handled in its own groutine. Potentially, this could cause
// data sharing problems. We need to look at locking or having a groutine per
// alta instance to avoid these problems

// Create a Alta container
func (self *AltaMgr) CreateAlta(altaSpec altaspec.AltaSpec) (*AltaState, error) {
	altaId := altaSpec.AltaId

	// If Alta already exists, return an error
	if self.altaDb[altaId] != nil {
		return nil, errors.New("Already exists")
	}

	// Handle bind volumes
	volumeBinds := make([]string, 0)
	for _, volBind := range altaSpec.Volumes {
		// Get host directory where volume is mounted
		hostDir, err := volumeAgent.GetBindVolumeDir(volBind)
		if err != nil {
			log.Errorf("Error getting mount point for %+v. Err: %v", volBind, err)
		} else {
			// Form string in the form hostDir:bindMountDir
			volOptStr := hostDir + ":" + volBind.BindMountPoint

			// add it to the volume bind list
			volumeBinds = append(volumeBinds, volOptStr)
		}
	}

	log.Infof("Volume spec: %+v\n, Volume binds: %+v\n", altaSpec.Volumes, volumeBinds)

	// Convert Alta spec to container spec
	containerSpec := libdocker.ContainerSpec{
		Name:       altaSpec.AltaName,
		Hostname:   altaSpec.AltaName,
		Memory:     altaSpec.Memory,
		CpuPerc:    altaSpec.CpuPerc,
		Image:      altaSpec.Image,
		Command:    altaSpec.Command,
		Args:       altaSpec.Args,
		Envs:       altaSpec.EnvList,
		WorkingDir: altaSpec.WorkingDir,

		Privileged:        false,
		RestartPolicyName: "on-failure",
		RestartRetryCount: 5,

		ExposePorts: altaSpec.ExposePorts,
		PortMapList: altaSpec.PortMapList,
		// Set network mode as none so that we can add network interfaces later
		NetworkMode: "none",

		VolumeBinds: volumeBinds, // Volumes to be bind mounted
	}

	// Create the docker container
	dockerCtx, err := libdocker.CreateContainer(&containerSpec)
	if err != nil {
		log.Errorf("Error creating docker container %+v. Error %v", containerSpec, err)
		return nil, err
	}

	// Construct alta state
	altaState := AltaState{
		AltaId:       altaId,
		ContainerId:  dockerCtx.DockerId,
		portNames:    make([]string, 16), // Limit to 16 intf per alta
		spec:         altaSpec,
		containerCtx: dockerCtx,
	}

	// Save the alta state in DB
	self.altaDb[altaId] = &altaState

	// return the alta state
	return &altaState, nil
}

// List all Alta containers on this node
func (self *AltaMgr) ListAlta() map[string]*AltaState {
	return self.altaDb
}

// Get an alta by container id
func (self *AltaMgr) FindAltaByContainerId(containerId string) *AltaState {
	for _, altaState := range self.altaDb {
		if altaState.ContainerId == containerId {
			return altaState
		}
	}

	return nil
}

// List all Alta containers on this node
func (self *AltaMgr) ListContainers() ([]string, error) {
	return libdocker.GetRunningContainers()
}

// Get detailed info about a specific alta
func (self *AltaMgr) GetAltaInfo() {

}

// Update a Alta with new spec. This generally requires stopping and deleting
// old running container and starting new one with new spec.
func (self *AltaMgr) UpdateAlta() {

}

// Repopulate alta spec for a container
func (self *AltaMgr) UpdateAltaInfo(containerId string, altaSpec altaspec.AltaSpec) error {
	// Get container context
	dockerCtx, err := libdocker.GetContainer(containerId)
	if err != nil {
		log.Errorf("Error getting container: %s. Err: %v", containerId, err)
		return err
	}

	// Construct alta state
	altaState := AltaState{
		AltaId:       altaSpec.AltaId,
		ContainerId:  containerId,
		portNames:    make([]string, 16), // Limit to 16 intf per alta
		spec:         altaSpec,
		containerCtx: dockerCtx,
	}

	// Save the alta state in DB
	self.altaDb[altaSpec.AltaId] = &altaState

	return nil
}

// Start a previously created alta container
func (self *AltaMgr) StartAlta(altaId string) error {
	// find the alta in DB
	altaState := self.altaDb[altaId]
	if altaState == nil {
		log.Errorf("Could not find Alta %s", altaId)
		return errors.New("Alta does not exists")
	}

	// Start the container
	err := altaState.containerCtx.StartContainer()
	if err != nil {
		log.Errorf("Error starting the container %s, Error %v", altaState.ContainerId, err)
		return err
	}

	// Create network interfaces for the container
	for ifNum, ifSpec := range altaState.spec.Endpoints {
		// Get container PID
		contPid := altaState.containerCtx.GetContainerPid()

		// Create interface
		portName, err := netAgent.CreateAltaEndpoint(contPid, ifNum, &ifSpec)
		if err != nil {
			log.Errorf("Error creating network interface. %+v\n. Error: %v\n", ifSpec, err)
		} else {
			// Save the port names for later cleanup
			// FIXME: This is another state that exists on athena
			altaState.portNames[ifNum] = portName
		}
	}

	return nil
}

// Stop a running container
func (self *AltaMgr) StopAlta(altaId string) error {
	// find the alta in DB
	altaState := self.altaDb[altaId]
	if altaState == nil {
		log.Errorf("Could not find Alta %s", altaId)
		return errors.New("Alta does not exists")
	}

	// Stop the container
	err := altaState.containerCtx.StopContainer()
	if err != nil {
		log.Errorf("Error stopping the container %s, Error %v", altaState.ContainerId, err)
	}

	// FIXME: remember alta is stopped and stop polling

	// remove associated network interfaces
	for _, portName := range altaState.portNames {
		if portName != "" {
			netAgent.DeleteAltaEndpoint(portName)
		}
	}
	return nil
}

// Remove a stopped container and clean up all associated state
// Note: Any volumes or networks created for this alta needs to be removed
//       after alta has been removed
func (self *AltaMgr) RemoveAlta(altaId string) error {
	// find the alta in DB
	altaState := self.altaDb[altaId]
	if altaState == nil {
		log.Errorf("Could not find Alta %s", altaId)
		return errors.New("Alta does not exists")
	}

	// Stop the container
	err := altaState.containerCtx.RemoveContainer()
	if err != nil {
		log.Errorf("Error removing the container %s, Error %v", altaState.ContainerId, err)
		return err
	}

	// Finally delete it from the DB
	delete(self.altaDb, altaId)

	return nil
}
