package altaCtrler

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/contiv/symphony/zeus/common"
	"github.com/contiv/symphony/zeus/netCtrler"
	"github.com/contiv/symphony/zeus/nodeCtrler"

	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/confStore/confStoreApi"

	log "github.com/Sirupsen/logrus"
)

// State of alta manager
type AltaMgr struct {
	altaDb     map[string]*AltaActor        // Main DB of alta containers
	altaNameDb map[string]*AltaActor        // mapping from alta names to container
	cStore     confStoreApi.ConfStorePlugin // persistence store
}

var altaCtrl *AltaMgr

// Create a new alta Mgr
func NewAltaCtrler(cStore confStoreApi.ConfStorePlugin) *AltaMgr {
	altaCtrl = new(AltaMgr)

	// Create the mapping databases
	altaCtrl.altaDb = make(map[string]*AltaActor)
	altaCtrl.altaNameDb = make(map[string]*AltaActor)

	// Keep a ref to cStore
	altaCtrl.cStore = cStore

	return altaCtrl
}

// Generate a Unique Id for the Alta container
func (self *AltaMgr) genAltaId() string {

	// Loop till we find an id that doesnt exist
	// Generate a random number and check if the id already exist in the remote
	// chance there is a collision.
	for {
		b := make([]byte, 8)
		_, err := rand.Read(b)
		if err != nil {
			fmt.Println("Error: ", err)
			return ""
		}

		uuid := strings.ToLower(fmt.Sprintf("%X", b[0:]))

		// If we found a unique id, return it
		if self.altaDb[uuid] == nil {
			return uuid
		}
	}

	return ""
}

// Build alta spec from user specified alta config
func buildAltaSpec(altaConfig *altaspec.AltaConfig, altaSpec *altaspec.AltaSpec) {
	// Initialize the parameters
	altaSpec.AltaName = altaConfig.Name
	altaSpec.Image = altaConfig.Image
	altaSpec.Command = []string{altaConfig.Command}
	altaSpec.EnvList = altaConfig.Environment

	// Parse CPU option
	if altaConfig.Cpu == "" {
		altaSpec.NumCpu = 1
		altaSpec.CpuPerc = 100
	} else {
		numCpu, _ := strconv.ParseUint(altaConfig.Cpu, 10, 32)
		altaSpec.NumCpu = uint32(numCpu)
		altaSpec.CpuPerc = 100
	}

	// Parse memory option
	if altaConfig.Memory == "" {
		altaSpec.Memory = 512 * 1024 * 1024
	} else if strings.Contains(altaConfig.Memory, "M") {
		mem, _ := strconv.Atoi(strings.Split(altaConfig.Memory, "M")[0])
		altaSpec.Memory = int64(mem) * 1024 * 1024
	} else if strings.Contains(altaConfig.Memory, "m") {
		mem, _ := strconv.Atoi(strings.Split(altaConfig.Memory, "m")[0])
		altaSpec.Memory = int64(mem) * 1024 * 1024
	} else if strings.Contains(altaConfig.Memory, "G") {
		mem, _ := strconv.Atoi(strings.Split(altaConfig.Memory, "G")[0])
		altaSpec.Memory = int64(mem) * 1024 * 1024 * 1024
	} else if strings.Contains(altaConfig.Memory, "g") {
		mem, _ := strconv.Atoi(strings.Split(altaConfig.Memory, "g")[0])
		altaSpec.Memory = int64(mem) * 1024 * 1024 * 1024
	} else {
		mem, _ := strconv.Atoi(altaConfig.Memory)
		altaSpec.Memory = int64(mem)
	}

	// Parse network option
	if len(altaConfig.Network) == 0 {
		netIf, err := netCtrler.CreateAltaNetIf(altaSpec.AltaId, "default", 0)
		if err != nil {
			log.Errorf("Error creating default network intf for %s", altaSpec.AltaId)
		} else {
			altaSpec.NetworkIfs = []altaspec.AltaNetIf{*netIf}
		}
	} else {
		var netIfs []altaspec.AltaNetIf

		// Loop thru each network name
		for indx, networkName := range altaConfig.Network {
			netIf, err := netCtrler.CreateAltaNetIf(altaSpec.AltaId, networkName, indx)
			if err != nil {
				log.Errorf("Error creating intf for %s, network %s", altaSpec.AltaId, networkName)
			} else {
				netIfs = append(netIfs, *netIf)
			}
		}

		// Set the network intf list
		altaSpec.NetworkIfs = netIfs
	}

	// Default volumes to mount
/* Disable this for now
	altaSpec.Volumes = []altaspec.AltaVolumeBind{
		{
			DatastoreType: "PersistentVolume",
			DatastoreVolumeId: altaSpec.AltaId,
			BindMountPoint: "/var/data",
		},
		{
			DatastoreType:     "HostVolume",
			DatastoreVolumeId: altaSpec.AltaId + ".log",
			BindMountPoint:    "/var/log",
		},
	}
*/
}

// Create a new Alta container
func (self *AltaMgr) CreateAlta(altaConfig *altaspec.AltaConfig) error {
	var altaSpec altaspec.AltaSpec

	// Check if a name was specified and a container of this name already exists
	if altaConfig.Name != "" {
		if self.altaNameDb[altaConfig.Name] != nil {
			log.Errorf("Error: Alta Container %s already exists", altaConfig.Name)
			return errors.New("Alta container already exists")
		}
	}

	log.Infof("Creating alta with config: %#v", altaConfig)

	//Create a unique Id
	altaSpec.AltaId = self.genAltaId()

	// Initialize the parameters
	buildAltaSpec(altaConfig, &altaSpec)

	// Create a new container
	alta, err := NewAlta(&altaSpec)
	if err != nil {
		log.Errorf("Error creating alta: %+v. Err: %v", altaConfig, err)
		return err
	}

	// Save the container in the DB
	self.altaDb[alta.AltaId] = alta
	if altaConfig.Name != "" {
		self.altaNameDb[altaConfig.Name] = alta
	}

	// post schedule event
	alta.AltaEvent("schedule")

	return nil
}

// Return a list of all alta containers
func (self *AltaMgr) ListAlta() []*common.AltaState {
	altaList := make([]*common.AltaState, 0)

	// Append each alta actor's model
	for _, alta := range self.altaDb {
		astate := common.AltaState{
			Spec: alta.Model.Spec,
			CurrNode: alta.Model.CurrNode,
			ContainerId: alta.Model.ContainerId,
			FsmState: alta.Model.Fsm.FsmState,
		}
		altaList = append(altaList, &astate)
	}

	log.Debugf("Returning alta list: %+v", altaList)

	return altaList
}

// AltaEvent trigger an event on the alta actor
func (self *AltaMgr) AltaEvent(altaId string, event string) error {
	// check for errors
	if self.altaDb[altaId] == nil {
		return errors.New("Alta not found")
	}

	// post the event
	alta := self.altaDb[altaId]
	alta.AltaEvent(event)

	return nil
}

// Diff the alta list we got from a node and what we expect
func (self *AltaMgr) DiffNodeAltaLList(nodeAddr string, altaList []altaspec.AltaContext) error {
	// Get the list of altas we expect on this node
	expAltaList := self.listAltaForNode(nodeAddr)

	// convert the list to maps indexed by container id
	expContMap := make(map[string]*AltaModel)
	expAltaMap := make(map[string]*AltaModel)
	contMap := make(map[string]*altaspec.AltaContext)
	altaMap := make(map[string]*altaspec.AltaContext)
	for _, alta := range expAltaList {
		expContMap[alta.ContainerId] = alta
		expAltaMap[alta.Spec.AltaId] = alta
	}
	for _, alta := range altaList {
		contMap[alta.ContainerId] = &alta
		altaMap[alta.AltaId] = &alta
	}

	// check if anything we expect is missing or changed
	for _, alta := range expAltaList {
		// If we dont have a container id yet, ignore it
		if alta.ContainerId == "" {
			log.Infof("DiffNodeAltaLList: Ignoring alta: %+v", alta)
			continue
		}

		// See if node is missing alta id
		if (altaMap[alta.Spec.AltaId] == nil) && (contMap[alta.ContainerId] != nil) {
			log.Infof("DiffNodeAltaLList: node does not know about alta: %+v", alta)

			path := fmt.Sprintf("/alta/%s/update", alta.ContainerId)
			var resp altaspec.ReqSuccess

			// Send the alta info
			err := nodeCtrler.NodePostReq(nodeAddr, path, alta.Spec, &resp)
			if err != nil {
				log.Errorf("Error sending alta info to node %s", nodeAddr)
			}
		}

		// See if node is missing the container
		if (altaMap[alta.Spec.AltaId] != nil) && (contMap[alta.ContainerId] == nil) {
			log.Infof("DiffNodeAltaLList: Container info mismatch for alta: %+v", alta)

			if alta.ContainerId != altaMap[alta.Spec.AltaId].ContainerId {
				log.Infof("Alta info needs update. old: %+v. New: %+v", alta, altaMap[alta.Spec.AltaId])
			} else if altaMap[alta.Spec.AltaId].ContainerId == "" {
				log.Infof("Master does not have container info: %+v", altaMap[alta.Spec.AltaId])
			}
		}

		// See if container is completely missing from the list
		if (altaMap[alta.Spec.AltaId] == nil) && (contMap[alta.ContainerId] == nil) {
			log.Infof("Alta %+v needs to be restarted", alta)

			// Queue failure event to alta.
			self.AltaEvent(alta.Spec.AltaId, "failure")
		}
	}

	// Check if there is a container that we dont expect
	for _, alta := range altaList {
		if alta.AltaId == "" && expContMap[alta.ContainerId] == nil {
			log.Infof("Node has unexpected alta: %+v. Kill it", alta)
			// FIXME:
		} else if alta.AltaId != "" && expAltaMap[alta.AltaId] == nil {
			log.Infof("Node has unexpected altaId %+v", alta)
		}
	}

	return nil
}

// Return a list of all alta containers
func (self *AltaMgr) listAltaForNode(nodeAddr string) []*AltaModel {
	var altaList []*AltaModel

	// Walk thru all altas and see if they match this node
	for _, alta := range self.altaDb {
		if alta.Model.CurrNode == nodeAddr {
			altaList = append(altaList, &alta.Model)
		}
	}

	return altaList
}

// Restore Alta actor state from cStore
func (self *AltaMgr) RestoreAltaActors() error {
	// Get the list of elements
	jsonArr, err := self.cStore.ListDir("alta")
	if err != nil {
		log.Errorf("Error restoring alta actor state")
		return err
	}

	// Loop thru each alta model
	for _, elemStr := range jsonArr {
		// Parse the json model
		var model AltaModel
		err = json.Unmarshal([]byte(elemStr), &model)
		if err != nil {
			log.Errorf("Error parsing object %s, Err %v", elemStr, err)
			return err
		}

		// Create an actor for the alta container
		alta, err := NewAlta(&model.Spec)
		if err != nil {
			log.Errorf("Error restoring Alta %s. Err: %v", model.Spec.AltaId, err)
			return err
		}

		// Restore state
		alta.Model.CurrNode = model.CurrNode
		alta.Model.ContainerId = model.ContainerId
		alta.Model.Fsm.FsmState = model.Fsm.FsmState

		// Save the container in the DB
		self.altaDb[alta.AltaId] = alta
		if model.Spec.AltaName != "" {
			self.altaNameDb[model.Spec.AltaName] = alta
		}

		log.Infof("Restored alta: %#v", alta)
	}

	return nil
}
