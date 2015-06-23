package cephdriver

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

// Ceph driver object
type CephDriver struct {
	dummy string // Driver has no state for now
}

// Volume specification
type CephVolumeSpec struct {
	VolumeName string // Name of the volume
	VolumeSize uint   // Size in MBs
	PoolName   string // Ceph Pool this volume belongs to default:rbd
}

// Create a new Ceph driver
func NewCephDriver() *CephDriver {
	// Create a new driver
	cephDriver := new(CephDriver)

	// Return it
	return cephDriver
}

// Create an RBD image and initialize ext4 filesystem on the image
func (self *CephDriver) CreateVolume(spec CephVolumeSpec) error {
	// create image name in 'pool/img' format
	imgName := spec.PoolName + "/" + spec.VolumeName
	devName := "/dev/rbd/" + imgName

	// Create an image
	out, err := exec.Command("/usr/bin/rbd", "create", imgName, "--size",
		strconv.Itoa(int(spec.VolumeSize))).CombinedOutput()
	if err != nil {
		log.Errorf("Error creating Ceph RBD image(name: %s, size: %d). Err: %v\n",
			imgName, spec.VolumeSize, err)
		log.Errorf("rbd create Output: %s\n", out)
		return err
	}
	log.Infof("rbd create Output: %s\n", out)

	// Temporarily map the image to create a filesystem
	out, err = exec.Command("/usr/bin/rbd", "map", imgName).CombinedOutput()
	if err != nil {
		log.Errorf("Error mapping the image %s. Error: %v", imgName, err)
		log.Errorf("rbd map output: %s", out)
		return err
	}
	log.Infof("rbd map output: %s", out)

	// Create ext4 filesystem on the device. this will take a while
	out, err = exec.Command("mkfs.ext4", "-m0", devName).CombinedOutput()
	if err != nil {
		log.Errorf("Error creating ext4 filesystem on %s. Error: %v", devName, err)
		log.Errorf("mkfs.ext4 output: %s", out)
		return err
	}
	log.Infof("mkfs.ext4 output: %s", out)

	// finally, Unmap the rbd image
	out, err = exec.Command("/usr/bin/rbd", "unmap", devName).CombinedOutput()
	if err != nil {
		log.Errorf("Error unmapping the device %s. Error: %v", devName, err)
		log.Errorf("rbd unmap output: %s", out)
		return err
	}
	log.Infof("rbd unmap output: %s", out)

	return nil
}

// Map an RBD image and mount it on /mnt/ceph/<datastore>/<volume> directory
// FIXME: Figure out how to use rbd locks
func (self *CephDriver) MountVolume(poolName, volumeName string) error {
	// formatted image name
	imgName := poolName + "/" + volumeName
	devName := "/dev/rbd/" + imgName

	// Directory to mount the volume
	dataStoreDir := "/mnt/ceph/" + poolName
	volumeDir := dataStoreDir + "/" + volumeName

	// Map the RBD image to an rbd block device
	out, err := exec.Command("/usr/bin/rbd", "map", imgName).CombinedOutput()
	if err != nil {
		log.Errorf("Error mapping the image %s. Error: %v", imgName, err)
		log.Errorf("rbd map output: %s", out)
		return err
	}
	log.Infof("rbd map output: %s", out)

	// Create directory to mount
	err = os.Mkdir("/mnt/ceph", 0700)
	if err != nil && !os.IsExist(err) {
		log.Errorf("error creating /mnt/ceph direcotry \n")
		return err
	}
	err = os.Mkdir(dataStoreDir, 0700)
	if err != nil && !os.IsExist(err) {
		log.Errorf("error creating '%s' direcotry \n", dataStoreDir)
		return err
	}
	err = os.Mkdir(volumeDir, 0777)
	if err != nil && !os.IsExist(err) {
		log.Errorf("error creating '%s' direcotry \n", volumeDir)
		return err
	}

	// Mount the RBD
	var flags uintptr = 0
	err = syscall.Mount(devName, volumeDir, "ext4", flags, "")
	if err != nil {
		log.Errorf("Failed to mount RBD dev %s: %s\n", devName, err.Error())
		return err
	}

	return nil
}

// Unount a Ceph volume, remove the mount directory and unmap the RBD device
func (self *CephDriver) UnmountVolume(poolName, volumeName string) error {
	// formatted image name
	imgName := poolName + "/" + volumeName
	devName := "/dev/rbd/" + imgName

	// Directory to mount the volume
	dataStoreDir := "/mnt/ceph/" + poolName
	volumeDir := dataStoreDir + "/" + volumeName

	// Unmount the RBD
	err := syscall.Unmount(volumeDir, 1) // Flag = 1: Force unmounting
	if err != nil {
		log.Errorf("Failed to mount  /dev/rbd/%s: %s\n", imgName, err.Error())
	}

	// Remove the mounted directory
	err = os.Remove(volumeDir)
	if err != nil {
		log.Errorf("error removing '%s' direcotry \n", volumeDir)
	}

	// finally, Unmap the rbd image
	out, err := exec.Command("/usr/bin/rbd", "unmap", devName).CombinedOutput()
	if err != nil {
		log.Errorf("Error unmapping the device %s. Error: %v", devName, err)
		log.Errorf("rbd unmap output: %s", out)
		return err
	}
	log.Infof("rbd unmap output: %s", out)

	return nil
}

// Delete an RBD volume i.e. rbd image
func (self *CephDriver) DeleteVolume(poolName, volumeName string) error {
	// create image name in 'pool/img' format
	imgName := poolName + "/" + volumeName

	// Delete the image
	out, err := exec.Command("/usr/bin/rbd", "rm", imgName).CombinedOutput()
	if err != nil {
		log.Errorf("Error deleting Ceph RBD image(name: %s). Err: %v\n", imgName, err)
		log.Errorf("rbd rm Output: %s\n", out)
		return err
	}
	log.Infof("rbd rm Output: %s\n", out)

	return nil
}
