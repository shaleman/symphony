package cephdriver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

const (
	defaultDeviceBase = "/dev/rbd"
	defaultMountBase  = "/mnt/ceph"
)

// Ceph driver object
type CephDriver struct {
	deviceBase string
	mountBase  string
}

// Volume specification
type CephVolumeSpec struct {
	VolumeName string // Name of the volume
	VolumeSize uint   // Size in MBs
	PoolName   string // Ceph Pool this volume belongs to default:rbd
}

// Create a new Ceph driver
func NewCephDriver() *CephDriver {
	return &CephDriver{
		deviceBase: defaultDeviceBase,
		mountBase:  defaultMountBase,
	}
}

func (cvs *CephVolumeSpec) Path() string {
	return filepath.Join(cvs.PoolName, cvs.VolumeName)
}

func (self *CephDriver) DevicePath(spec CephVolumeSpec) string {
	return filepath.Join(self.deviceBase, spec.Path())
}

func (self *CephDriver) volumeCreate(spec CephVolumeSpec) error {
	// Create an image
	out, err := exec.Command("rbd", "create", spec.Path(), "--size",
		strconv.Itoa(int(spec.VolumeSize))).CombinedOutput()

	log.Debug(string(out))

	if err != nil {
		return fmt.Errorf("Error creating Ceph RBD image(name: %s, size: %d). Err: %v\n",
			spec.Path(), spec.VolumeSize, err)
	}

	return nil
}

func (self *CephDriver) mapImage(spec CephVolumeSpec) error {
	// Temporarily map the image to create a filesystem
	out, err := exec.Command("rbd", "map", spec.Path()).CombinedOutput()

	log.Debug(string(out))

	if err != nil {
		return fmt.Errorf("Error mapping the image %s. Error: %v", spec.Path(), err)
	}

	return nil
}

func (self *CephDriver) mkfsVolume(spec CephVolumeSpec) error {
	// Create ext4 filesystem on the device. this will take a while
	out, err := exec.Command("mkfs.ext4", "-m0", self.DevicePath(spec)).CombinedOutput()

	log.Debug(string(out))

	if err != nil {
		return fmt.Errorf("Error creating ext4 filesystem on %s. Error: %v", self.DevicePath(spec), err)
	}

	return nil
}

func (self *CephDriver) unmapImage(spec CephVolumeSpec) error {
	// finally, Unmap the rbd image
	out, err := exec.Command("rbd", "unmap", self.DevicePath(spec)).CombinedOutput()

	log.Debug(string(out))

	if err != nil {
		return fmt.Errorf("Error unmapping the device %s. Error: %v", self.DevicePath(spec), err)
	}

	return nil
}

// Create an RBD image and initialize ext4 filesystem on the image
func (self *CephDriver) CreateVolume(spec CephVolumeSpec) error {
	if err := self.volumeCreate(spec); err != nil {
		return err
	}

	if err := self.mapImage(spec); err != nil {
		return err
	}

	if err := self.mkfsVolume(spec); err != nil {
		return err
	}

	if err := self.unmapImage(spec); err != nil {
		return err
	}

	return nil
}

// Map an RBD image and mount it on /mnt/ceph/<datastore>/<volume> directory
// FIXME: Figure out how to use rbd locks
func (self *CephDriver) MountVolume(spec CephVolumeSpec) error {
	// formatted image name
	devName := self.DevicePath(spec)

	// Directory to mount the volume
	dataStoreDir := filepath.Join(self.mountBase, spec.PoolName)
	volumeDir := filepath.Join(dataStoreDir, spec.VolumeName)

	if err := self.mapImage(spec); err != nil {
		return err
	}

	// Create directory to mount
	if err := os.Mkdir(self.mountBase, 0700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("error creating %q directory: %v", self.mountBase, err)
	}

	if err := os.Mkdir(dataStoreDir, 0700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("error creating %q directory: %v", dataStoreDir)
	}

	if err := os.Mkdir(volumeDir, 0777); err != nil && !os.IsExist(err) {
		return fmt.Errorf("error creating %q directory: %v", volumeDir)
	}

	// Mount the RBD
	if err := syscall.Mount(devName, volumeDir, "ext4", 0, ""); err != nil {
		return fmt.Errorf("Failed to mount RBD dev %q: %v", devName, err.Error())
	}

	return nil
}

// Unount a Ceph volume, remove the mount directory and unmap the RBD device
func (self *CephDriver) UnmountVolume(spec CephVolumeSpec) error {
	// formatted image name
	devName := self.DevicePath(spec)

	// Directory to mount the volume
	dataStoreDir := filepath.Join(self.mountBase, spec.PoolName)
	volumeDir := filepath.Join(dataStoreDir, spec.VolumeName)

	// Unmount the RBD
	// Flag = 1: Force unmounting
	if err := syscall.Unmount(volumeDir, 1); err != nil && err != syscall.ENOENT {
		return fmt.Errorf("Failed to unmount %q: %v", devName, err)
	}

	// Remove the mounted directory
	if err := os.Remove(volumeDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error removing %q directory: %v", volumeDir, err)
	}

	if err := self.unmapImage(spec); err != nil {
		return err
	}

	return nil
}

// Delete an RBD volume i.e. rbd image
func (self *CephDriver) DeleteVolume(spec CephVolumeSpec) error {
	out, err := exec.Command("rbd", "rm", spec.Path()).CombinedOutput()

	log.Debug(string(out))

	if err != nil {
		return fmt.Errorf("Error deleting Ceph RBD image(name: %s). Err: %v", spec.Path(), err)
	}

	return nil
}
