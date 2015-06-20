package main

import (
	"errors"
	"os"

	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/cephdriver"

	log "github.com/Sirupsen/logrus"
)

type VolumeAgent struct {
	cephDriver *cephdriver.CephDriver // Ceph library
}

// Create a volume agent
func NewVolumeAgent() *VolumeAgent {
	volAgent := new(VolumeAgent)

	// Initialize Ceph driver
	volAgent.cephDriver = cephdriver.NewCephDriver()

	return volAgent
}

// Create host volume directory
func (self *VolumeAgent) createHostVolume(volumeName string) error {
	// Directory to mount the volume
	dataStoreDir := "/mnt/hostvols/"
	volumeDir := dataStoreDir + "/" + volumeName

	// Create the directories
	err := os.Mkdir(dataStoreDir, 0700)
	if err != nil && !os.IsExist(err) {
		log.Errorf("error creating '%s' direcotry \n", dataStoreDir)
		return err
	}
	err = os.Mkdir(volumeDir, 0777)
	if err != nil && !os.IsExist(err) {
		log.Errorf("error creating '%s' direcotry \n", volumeDir)
		return err
	}

	return nil
}

// delete host volume directory
func (self *VolumeAgent) deleteHostVolume(volumeName string) error {
	// Directory to mount the volume
	dataStoreDir := "/mnt/hostvols/"
	volumeDir := dataStoreDir + "/" + volumeName

	// Remove the mounted directory
	err := os.Remove(volumeDir)
	if err != nil {
		log.Errorf("error removing '%s' direcotry \n", volumeDir)
	}

	return nil
}

// ********************* Volume Agent API ***********************
// Create a volume
func (self *VolumeAgent) CreateVolume(volumeSpec altaspec.AltaVolumeSpec) error {
	switch volumeSpec.DatastoreType {
	case "PersistentVolume":
		// Ceph volume info
		cephVolSpec := cephdriver.CephVolumeSpec{
			VolumeName: volumeSpec.DatastoreVolumeId,
			VolumeSize: volumeSpec.VolumeSize,
			PoolName:   "rbd",
		}

		// Ask ceph library to create the volume
		return self.cephDriver.CreateVolume(cephVolSpec)

	case "HostVolume":
		// Nothing to do for host volumes
		return nil

	default:
		log.Errorf("Unknown datastore type %s", volumeSpec.DatastoreType)
		return errors.New("Unknown datastore type")
	}
}

// Mount the volume on host file system
func (self *VolumeAgent) MountVolume(volumeSpec altaspec.AltaVolumeSpec) error {
	switch volumeSpec.DatastoreType {
	case "PersistentVolume":

		// Ask ceph library to mount the volume
		return self.cephDriver.MountVolume("rbd", volumeSpec.DatastoreVolumeId)

	case "HostVolume":
		// mount host volumes
		return self.createHostVolume(volumeSpec.DatastoreVolumeId)

	default:
		log.Errorf("Unknown datastore type %s", volumeSpec.DatastoreType)
		return errors.New("Unknown datastore type")
	}
}

// unmount the volume
func (self *VolumeAgent) UnmountVolume(volumeSpec altaspec.AltaVolumeSpec) error {
	switch volumeSpec.DatastoreType {
	case "PersistentVolume":

		// Ask ceph library to unmount the volume
		return self.cephDriver.UnmountVolume("rbd", volumeSpec.DatastoreVolumeId)

	case "HostVolume":
		// delete host volumes
		return self.deleteHostVolume(volumeSpec.DatastoreVolumeId)

	default:
		log.Errorf("Unknown datastore type %s", volumeSpec.DatastoreType)
		return errors.New("Unknown datastore type")
	}
}

// Delete the volume
func (self *VolumeAgent) DeleteVolume(volumeSpec altaspec.AltaVolumeSpec) error {
	switch volumeSpec.DatastoreType {
	case "PersistentVolume":
		// Ask ceph library to delete the volume
		return self.cephDriver.DeleteVolume("rbd", volumeSpec.DatastoreVolumeId)

	case "HostVolume":
		// Nothing to do for host volumes
		return nil

	default:
		log.Errorf("Unknown datastore type %s", volumeSpec.DatastoreType)
		return errors.New("Unknown datastore type")
	}
}

// Return the host directory path where a volume is mounted
func (self *VolumeAgent) GetBindVolumeDir(volumeBind altaspec.AltaVolumeBind) (string, error) {
	switch volumeBind.DatastoreType {
	case "PersistentVolume":
		// ceph volume
		return "/mnt/ceph/rbd/" + volumeBind.DatastoreVolumeId, nil

	case "HostVolume":
		// host volumes
		return "/mnt/hostvols/" + volumeBind.DatastoreVolumeId, nil

	default:
		log.Errorf("Unknown datastore type %s", volumeBind.DatastoreType)
		return "", errors.New("Unknown datastore type")
	}
}
