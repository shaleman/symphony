package cephdriver

import (
	//"fmt"
	"testing"
	//"time"
	"flag"
	"os"
	//"os/exec"
	"errors"
	"io"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func TestCreateVolume(t *testing.T) {
	// Hack to log output
	flag.Lookup("logtostderr").Value.Set("true")

	// Create a new driver
	cephDriver := NewCephDriver()

	volumeSpec := CephVolumeSpec{
		VolumeName: "pithos1234",
		VolumeSize: 1024,
		PoolName:   "rbd",
	}
	// Create a volume
	err := cephDriver.CreateVolume(volumeSpec)
	if err != nil {
		log.Errorf("Error creating the volume. Err: %v", err)
		t.Errorf("Failed to create a volume")
	}
}

func readWriteTest(mountDir string) error {
	// Write a file and verify you can read it
	file, err := os.Create(mountDir + "/test.txt")
	if err != nil {
		log.Errorf("Error creating file. Err: %v", err)
		return errors.New("Failed to create a file")
	}
	defer file.Close()

	num, err := file.WriteString("Test string\n")
	if err != nil {
		log.Errorf("Error writing file. Err: %v", err)
		return errors.New("Failed to write a file")
	}
	file.Sync()

	file, err = os.Open(mountDir + "/test.txt")
	if err != nil {
		log.Errorf("Error opening file. Err: %v", err)
		return errors.New("Failed to open a file")
	}
	defer file.Close()

	rb := make([]byte, 200)
	_, err = io.ReadAtLeast(file, rb, num)
	var rbs string = string(rb)
	if (err != nil) || (!strings.Contains(rbs, "Test string")) {
		log.Errorf("Error reading back file(Got %s). Err: %v", rbs, err)
		return errors.New("Failed to read back a file")
	}
	log.Infof("Read back: %s", string(rb))

	return nil
}

func TestMountVolume(t *testing.T) {
	// Create a new driver
	cephDriver := NewCephDriver()

	// mount the volume
	err := cephDriver.MountVolume("rbd", "pithos1234")
	if err != nil {
		log.Errorf("Error mounting the volume. Err: %v", err)
		t.Errorf("Failed to mount a volume")
	}

	err = readWriteTest("/mnt/ceph/rbd/pithos1234")
	if err != nil {
		log.Errorf("Error during read/write test. Err: %v", err)
		t.Errorf("Failed read/write test")
	}
}

func TestUnmountVolume(t *testing.T) {
	// Create a new driver
	cephDriver := NewCephDriver()

	// unmount the volume
	err := cephDriver.UnmountVolume("rbd", "pithos1234")
	if err != nil {
		log.Errorf("Error unmounting the volume. Err: %v", err)
		t.Errorf("Failed to unmount a volume")
	}
}

func TestDeleteVolume(t *testing.T) {
	// Create a new driver
	cephDriver := NewCephDriver()

	// delete the volume
	err := cephDriver.DeleteVolume("rbd", "pithos1234")
	if err != nil {
		log.Errorf("Error deleting the volume. Err: %v", err)
		t.Errorf("Failed to delete a volume")
	}
}

func TestRepeatedMountUnmout(t *testing.T) {
	// Create a new driver
	cephDriver := NewCephDriver()

	volumeSpec := CephVolumeSpec{
		VolumeName: "pithos1234",
		VolumeSize: 1024,
		PoolName:   "rbd",
	}
	// Create a volume
	err := cephDriver.CreateVolume(volumeSpec)
	if err != nil {
		log.Errorf("Error creating the volume. Err: %v", err)
		t.Errorf("Failed to create a volume")
	}

	// Repeatedly perform mount unmount test
	for i := 0; i < 100; i++ {
		// mount the volume
		err := cephDriver.MountVolume("rbd", "pithos1234")
		if err != nil {
			log.Errorf("Error mounting the volume. Err: %v", err)
			t.Errorf("Failed to mount a volume")
		}

		err = readWriteTest("/mnt/ceph/rbd/pithos1234")
		if err != nil {
			log.Errorf("Error during read/write test. Err: %v", err)
			t.Errorf("Failed read/write test")
		}

		// unmount the volume
		err = cephDriver.UnmountVolume("rbd", "pithos1234")
		if err != nil {
			log.Errorf("Error unmounting the volume. Err: %v", err)
			t.Errorf("Failed to unmount a volume")
		}
	}

	// delete the volume
	err = cephDriver.DeleteVolume("rbd", "pithos1234")
	if err != nil {
		log.Errorf("Error deleting the volume. Err: %v", err)
		t.Errorf("Failed to delete a volume")
	}
}
