package libdocker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/golang/glog"
)

// Internal context for each docker container
type ContainerCtx struct {
	DockerId   string
	dockerInfo *docker.Container
}

type ContainerSpec struct {
	Name        string   // Name of the container
	Hostname    string   // Hostname to set inside the container
	Memory      int64    // Memory limit format: <number><optional unit>, where unit = b, k, m or g
	CPUSet      string   // CPUs cores in which to allow execution (0-3, 0,1)
	CpuPerc     int64    // CPU percentage on each core
	Envs        []string // Environment variables
	Command     []string // Primary command aka entry point
	Args        []string // Arguments to the command
	Image       string   // Image name
	WorkingDir  string   // Working directory for the command
	Privileged  bool     // is this a privilaged container?
	VolumeBinds []string // Volumes that needs to be bind mounted
	ExposePorts []string // Expose these ports(alternative to EXPOSE in Dockerfile)
	PortMapList []string // Port mapping from container port to host ports
	NetworkMode string   // Network mode to be used for inheriting other container's network namespace
}

// Convert CPU percentage unit to CPU shares docker uses.
// CPU shares are not strictly guaranteed by the kernel. This is more of a
// minimum cpu guarantee on the cores where container is allowed to run
func cpuPercToShares(cpuPerc int64) int64 {
	if cpuPerc == 0 {
		// zero means unset. Use kernel default.
		return 0
	}
	// convert to docker cpu shares
	shares := (cpuPerc * sharesPerCPU) / 100
	if shares < minShares {
		return minShares
	}
	return shares
}

// Create a container on this host, but dont start it yet
func CreateContainer(cSpec *ContainerSpec) (*ContainerCtx, error) {
	// Convert exposed ports
	// FIXME: Handle ranges and allow specifying proto
	exposePorts := make(map[docker.Port]struct{})
	for _, port := range cSpec.ExposePorts {
		exposePorts[docker.Port(port+"/tcp")] = struct{}{}
		exposePorts[docker.Port(port+"/udp")] = struct{}{}
	}

	// convert Port mapping
	// FIXME: allow proto specification, allow hostIP specification
	// FIXME-2: String parsing is very brittle here
	portMapList := make(map[docker.Port][]docker.PortBinding)
	for _, portMap := range cSpec.PortMapList {
		pmap := strings.Split(portMap, ":")
		containerPort := pmap[0]
		hostPort := pmap[1]
		portMapList[docker.Port(containerPort+"/tcp")] = []docker.PortBinding{{
			HostIP:   "0.0.0.0",
			HostPort: hostPort,
		}}
	}

	// Convert container spec to docker format
	dockerOpts := docker.CreateContainerOptions{
		Name: cSpec.Name,
		Config: &docker.Config{
			Env:          cSpec.Envs,
			ExposedPorts: exposePorts,
			Hostname:     cSpec.Hostname,
			Image:        cSpec.Image,
			Memory:       cSpec.Memory,
			CPUShares:    cpuPercToShares(cSpec.CpuPerc),
			CPUSet:       cSpec.CPUSet,
			WorkingDir:   cSpec.WorkingDir,
			Entrypoint:   cSpec.Command,
			Cmd:          cSpec.Args,

			AttachStdin:  false,
			AttachStdout: false,
			AttachStderr: false,
			Tty:          false,
			OpenStdin:    true,
			StdinOnce:    false,
		},
		HostConfig: &docker.HostConfig{
			PortBindings: portMapList,
			Binds:        cSpec.VolumeBinds,
			NetworkMode:  cSpec.NetworkMode,
			// FIXME: figure out how to set network mode
			// IpcMode:      cSpec.NetworkMode,
			Privileged: cSpec.Privileged,
		},
	}

	jsonStr, _ := json.Marshal(dockerOpts)
	glog.Infof("Creating docker container using params: %s\n", jsonStr)

	// Create the container
	dockerContainer, err := dockerClient.CreateContainer(dockerOpts)
	if err != nil {
		glog.Errorf("Error creating the container %s. Error: %v", cSpec.Name, err)

		return nil, err
	}
	// Create a containerCtx
	containerCtx := ContainerCtx{
		DockerId: dockerContainer.ID,
	}

	// Populate all fields of a the container
	containerInfo, err := dockerClient.InspectContainer(dockerContainer.ID)
	if err != nil {
		glog.Errorf("Error getting containerInfo. Err: %v", err)
	} else {
		containerCtx.dockerInfo = containerInfo
	}

	// Print some info
	glog.Infof("Created container: %s", cSpec.Name)

	// Return the container context
	return &containerCtx, nil
}

// Start an existing container. To be used while starting a previously
// stopped container or starting a previously created container
func (self *ContainerCtx) StartContainer() error {
	err := dockerClient.StartContainer(self.DockerId, nil)
	if err != nil {
		glog.Errorf("Error removing container %s, error: %v", self.DockerId, err)
	}

	// Repopulate all fields of a the container
	containerInfo, err := dockerClient.InspectContainer(self.DockerId)
	if err != nil {
		glog.Errorf("Error getting containerInfo. Err: %v", err)
	} else {
		self.dockerInfo = containerInfo
	}

	// Report the success
	glog.Infof("Started container %s", self.DockerId)

	return err
}

// Gracefully Stop a running container.
func (self *ContainerCtx) StopContainer() error {
	err := dockerClient.StopContainer(self.DockerId, DOCKER_STOP_WAIT_TIME)
	if err != nil {
		glog.Errorf("Error removing container %s, error: %v", self.DockerId, err)
	}

	glog.Infof("Stopped container %s", self.DockerId)

	return err
}

// Remove container from this host
func (self *ContainerCtx) RemoveContainer() error {
	// Ask docker to remove the container
	err := dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: self.DockerId})
	if err != nil {
		glog.Errorf("Error removing container %s, error: %v", self.DockerId, err)
	}

	glog.Infof("Removed container %s", self.DockerId)

	return err
}

// Execute a command in a container's context
func (self *ContainerCtx) ExecCmdInContainer(cmds []string) (*bytes.Buffer, error) {
	// Options for exec
	execOpts := docker.CreateExecOptions{
		Container:    self.DockerId,
		Cmd:          cmds,
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	// Create an exec context
	execCtx, err := dockerClient.CreateExec(execOpts)
	if err != nil {
		glog.Errorf("Failed to create exec Ctx for %s", self.DockerId)
		return nil, err
	}

	// Options for start exec
	var buf bytes.Buffer
	startExecOpts := docker.StartExecOptions{
		Detach:       false,
		OutputStream: &buf,
	}

	// Execute the comands
	dockerClient.StartExec(execCtx.ID, startExecOpts)

	fmt.Printf("Got Output: \n %s\n", buf)

	return &buf, nil
}

// Get stdout/stderr output of container (i.e. of primary process in container)
// TODO:
func (self *ContainerCtx) GetContainerLog() {

}

func (self *ContainerCtx) GetContainerPid() int {
	return self.dockerInfo.State.Pid
}

// Get all containers running on this system
// to be used by atheena to periodically scan the host to see if any unwanted
// container is running on the system and to reconsile the state when it starts up
// TODO:
func GetAllContainers() ([]*ContainerCtx, error) {
	return nil, nil
}
