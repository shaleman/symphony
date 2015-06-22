package libdocker

import (
	// "fmt"
	"testing"
)

// Test image exists call
/* DEPRECATED
func TestCheckImageExists(t *testing.T) {
    if (!CheckImageExists("busybox")) {
        t.Errorf("Failed. expecting busybox to exists");
    }

    if (!CheckImageExists("busybox:latest")) {
        t.Errorf("Failed. expecting busybox to exists");
    }

    if (CheckImageExists("some-non-existing-img")) {
        t.Errorf("Failed. expecting busybox to exists");
    }
}
*/

// Test if an image exists
func TestIsImagePresent(t *testing.T) {
	// Check if busybox image exists
	isPresent, err := IsImagePresent("some-non-existing-img")
	if (err != nil) || (isPresent) {
		t.Errorf("Failed. Was not expecting busybox image yet")
	}
}

func TestPullImage(t *testing.T) {
	err := PullImage("busybox:latest")
	if err != nil {
		t.Errorf("Failed to pull the image")
	}

	// double check to make sure image exists now
	isPresent, err := IsImagePresent("busybox:latest")
	if (err != nil) || (!isPresent) {
		t.Errorf("Failed. Was expecting busybox image to be present now")
	}
}

// Try creating and running a container
func TestContainer(t *testing.T) {
	containerSpec := ContainerSpec{
		Name:  "my-busybox",
		Image: "busybox:latest",
	}
	// create the container
	container, err := CreateContainer(&containerSpec)
	if err != nil {
		t.Errorf("Failed to create the container. Error %v", err)
		return
	}

	// Start the container
	err = container.StartContainer()
	if err != nil {
		t.Errorf("Failed to start the container. Error %v", err)
	}

	// Execute a command
	_, err = container.ExecCmdInContainer([]string{"ls", "-l"})
	if err != nil {
		t.Errorf("Failed to Execute command in the container. Error %v", err)
	}

	// stop the container
	err = container.StopContainer()
	if err != nil {
		t.Errorf("Failed to stop the container. Error %v", err)
	}

	// Remove the container
	err = container.RemoveContainer()
	if err != nil {
		t.Errorf("Failed to remove the container. Error %v", err)
	}

}
