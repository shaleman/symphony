package libdocker

import (
	"bytes"
	"flag"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/parsers"
	docker "github.com/fsouza/go-dockerclient"
)

const (
	DOCKER_STOP_WAIT_TIME = 30 // Wait 30sec before killing the container
	// Taken from lmctfy https://github.com/google/lmctfy/blob/master/lmctfy/controllers/cpu_controller.cc
	minShares    = 2
	sharesPerCPU = 1024
)

// Create a new docker client
var dockerClient = newClient()

// Connect to docker and return the client
func newClient() *docker.Client {
	// We assume libdocker runs locally on docker host
	// When libdocker is running inside a container, it should be a privilaged
	// Container and docker.sock needs to be mounted into the container
	endpoint := "unix:///var/run/docker.sock"

	// Connect to docker
	client, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatal("Could not connect to docker")
	}

	// HACK: temporary hack to log to console
	flag.Lookup("logtostderr").Value.Set("true")

	return client
}

// Check if an image(by name) exists on this host
// DEPRECATED: dont use this API
func CheckImageExists(imgName string) bool {
	// Get the list of images from docker
	imgList, err := dockerClient.ListImages(docker.ListImagesOptions{All: false})
	if err != nil {
		log.Error("Error Getting image list %s", err)

		return false
	}

	// DEBUG: Print the response we got
	// fmt.Printf("Image List: \n %+v\n", imgList)

	// Walk the array of images
	for _, img := range imgList {
		// walk each repo tag
		for _, repoTag := range img.RepoTags {
			// Check if the image or image:latest exists
			if (repoTag == imgName) || (repoTag == (imgName + ":latest")) {
				log.Infof("Image %s exists as %s\n", imgName, repoTag)

				// Yes, exists
				return true
			}
		}
	}

	log.Infof("Image %s does not exist\n", imgName)
	return false
}

// Check if an image is present on this docker host
func IsImagePresent(imgName string) (bool, error) {
	imgInfo, err := dockerClient.InspectImage(imgName)
	if (err == nil) && (imgInfo != nil) {
		return true, nil
	}
	// This is super brittle, but its the best we got.
	// TODO: Land code in the docker client to use docker.Error here instead.
	if err.Error() == "no such image" {
		return false, nil
	}
	return false, err
}

// Pull an image from docker registry.
//     This blocks till the image is successfully pulled or an error occurs
// FIXME: add an option to specify the registry
// FIXME 2: Handle authentication
func PullImage(imgName string) error {
	repoToPull, tag := parsers.ParseRepositoryTag(imgName)

	// If no tag was specified, use the default "latest".
	if len(tag) == 0 {
		tag = "latest"
	}

	// Options for pull image
	var buf bytes.Buffer
	opts := docker.PullImageOptions{
		Repository:    repoToPull,
		Tag:           tag,
		OutputStream:  &buf,
		RawJSONStream: true,
	}

	// Ask docker to pull the image
	err := dockerClient.PullImage(opts, docker.AuthConfiguration{})
	if err != nil {
		log.Errorf("Error pulling image: %v", err)
		return err
	}

	// DEBUG: print the raw output stream
	fmt.Printf("Got resp: %s\nDone\n", buf)

	return nil
}

/* Sample usage:
    if (!libdocker.IsImagePresent(imgName)) {
        libdocker.PullImage(imgName)
    }

    // Create the container
    var container = libdocker.CreateContainer()

    // Start the container
    container.StartContainer()


}
*/
