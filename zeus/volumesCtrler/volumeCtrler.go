package volumesCtrler

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/confStore/confStoreApi"
	"github.com/contiv/symphony/pkg/libfsm"

	log "github.com/Sirupsen/logrus"
)

// Manage volumes

const MAX_VOLUMES = 1000

// volume controller state
type VolumeCtrler struct {
	volumeDb map[string]*VolumeActor      // DB of volumes
	cStore   confStoreApi.ConfStorePlugin // conf store
}

// Main controller for the volumes
var ctrler *VolumeCtrler

// Initialize the volume controller
func Init(cStore confStoreApi.ConfStorePlugin) error {
	ctrler = new(VolumeCtrler)

	// Initialize
	ctrler.volumeDb = make(map[string]*VolumeActor)
	ctrler.cStore = cStore

	return nil
}

// Create a volumes
func CreateVolume(volumeSpec altaspec.AltaVolumeSpec, hostAddr string) error {
	volumeKey := volumeSpec.DatastoreType + ":" + volumeSpec.DatastoreVolumeId

	// Check if the volume already exists
	if ctrler.volumeDb[volumeKey] != nil {
		log.Errorf("Volume %s already exists", volumeKey)
		return errors.New("Volume already exists")
	}

	// Create an actor for the volume
	volume, err := NewVolumeActor(volumeSpec)
	if err != nil {
		log.Errorf("Error creating volume %s. Err: %v", volumeKey, err)
		return err
	}

	// Save it in the DB
	ctrler.volumeDb[volumeKey] = volume

	// Initialize the volume
	volume.VolumeEvent(libfsm.Event{"init", hostAddr})

	log.Infof("Created volume: %s", volumeKey)

	return nil
}

// Mount a volume on a host. Create the volume with default parameters if it
// does not exist
func MountVolume(volumeBind altaspec.AltaVolumeBind, hostAddr string) error {
	volumeKey := volumeBind.DatastoreType + ":" + volumeBind.DatastoreVolumeId

	// Create the volume with default params if it doesnt exist
	if ctrler.volumeDb[volumeKey] == nil {
		volumeSpec := altaspec.AltaVolumeSpec{
			DatastoreType:     volumeBind.DatastoreType,
			DatastoreVolumeId: volumeBind.DatastoreVolumeId,
			VolumeSize:        1024,
		}
		err := CreateVolume(volumeSpec, hostAddr)
		if err != nil {
			log.Errorf("Error creating the volume %s. Err: %v", volumeKey, err)
			return err
		}
	}

	volume := ctrler.volumeDb[volumeKey]

	// Trigger mount event
	volume.VolumeEvent(libfsm.Event{"mount", hostAddr})

	// Wait till its mounted or a timeout is reached
	// FIXME: handle the error case where a volume is still mounted on a stale
	//        host. Possible unmount on old host and mount on new host
	cnt := 0
	for {
		// Check every second
		<-time.After(time.Second)
		if (volume.Model.Fsm.FsmState == "mounted") &&
			(volume.Model.CurrNode == hostAddr) {
			return nil
		}

		cnt++
		if cnt > 15 {
			log.Errorf("Timeout while mounting volume %s on host %s", volumeKey, hostAddr)
			return errors.New("Volume mount timeout")
		}
	}
}

// Unmount the volume from a host
func UnmountVolume(volumeBind altaspec.AltaVolumeBind) error {
	volumeKey := volumeBind.DatastoreType + ":" + volumeBind.DatastoreVolumeId

	// Check if the volume exists
	if ctrler.volumeDb[volumeKey] == nil {
		log.Errorf("Volume %s not found", volumeKey)
		return errors.New("Volume not found")
	}

	// Check if volume is in expected state
	volume := ctrler.volumeDb[volumeKey]
	if volume.Model.Fsm.FsmState != "mounted" {
		log.Errorf("Volume %s is not mounted", volumeKey)
	}

	// Trigger unmount event
	volume.VolumeEvent(libfsm.Event{"unmount", nil})

	return nil
}

// Delete a volume.
// This generally happens when we delete an alta container
func DeleteVolume(volumeSpec altaspec.AltaVolumeSpec) error {
	return nil
}

// Restore all Volumes
func RestoreVolumes() error {
	// Get the list of elements
	jsonArr, err := ctrler.cStore.ListDir("volume")
	if err != nil {
		log.Errorf("Error restoring volume state")
		return err
	}

	// Loop thru each volume
	for _, elemStr := range jsonArr {
		// Parse the json model
		var model VolumeModel
		err = json.Unmarshal([]byte(elemStr), &model)
		if err != nil {
			log.Errorf("Error parsing object %s, Err %v", elemStr, err)
			return err
		}

		volumeKey := model.Spec.DatastoreType + ":" + model.Spec.DatastoreVolumeId

		// Create an actor for the volume
		volume, err := NewVolumeActor(model.Spec)
		if err != nil {
			log.Errorf("Error creating volume %s. Err: %v", volumeKey, err)
			return err
		}

		// Restore state
		volume.Model.CurrNode = model.CurrNode
		volume.Model.Fsm.FsmState = model.Fsm.FsmState

		// Save it in the DB
		ctrler.volumeDb[volumeKey] = volume

		log.Infof("Restored volume: %#v", volume)
	}

	return nil
}
