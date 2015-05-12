package altaCtrler

import (
    "fmt"
    //"time"
    "errors"
    "strings"
    "strconv"
    "crypto/rand"
    "encoding/json"

    "zeus/netCtrler"

    "pkg/altaspec"
    "pkg/confStore/confStoreApi"

    "github.com/golang/glog"
)

type AltaMgr struct {
    altaDb      map[string]*AltaActor    // Main DB of alta containers
    altaNameDb  map[string]*AltaActor    // mapping from alta names to container
    cStore      confStoreApi.ConfStorePlugin    // persistence store
}

// Local state
var altaCtrl *AltaMgr

// Create a new alta Mgr
func Init(cStore confStoreApi.ConfStorePlugin) error {
    altaCtrl = new(AltaMgr)

    // Create the mapping databases
    altaCtrl.altaDb = make(map[string]*AltaActor)
    altaCtrl.altaNameDb = make(map[string]*AltaActor)

    // Keep a ref to cStore
    altaCtrl.cStore = cStore

    return nil
}

// Generate a Unique Id for the Alta container
func genAltaId() string {

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
        if (altaCtrl.altaDb[uuid] == nil) {
            return uuid
        }
    }

    return ""
}

// Build alta spec from user specified alta config
func buildAltaSpec(altaConfig *altaspec.AltaConfig, altaSpec *altaspec.AltaSpec)  {
    // Initialize the parameters
    altaSpec.AltaName = altaConfig.Name
    altaSpec.Image = altaConfig.Image
    altaSpec.Command = []string{altaConfig.Command}
    altaSpec.EnvList = altaConfig.Environment

    // Parse CPU option
    if (altaConfig.Cpu == "") {
        altaSpec.NumCpu = 1
        altaSpec.CpuPerc = 100
    } else {
        numCpu, _ := strconv.ParseUint(altaConfig.Cpu, 10, 32)
        altaSpec.NumCpu = uint32(numCpu)
        altaSpec.CpuPerc = 100
    }

    // Parse memory option
    if (altaConfig.Memory == "") {
        altaSpec.Memory = 512 * 1024 * 1024
    } else if (strings.Contains(altaConfig.Memory, "M")) {
        mem, _ := strconv.Atoi(strings.Split(altaConfig.Memory, "M")[0])
        altaSpec.Memory = int64(mem) * 1024 * 1024
    } else if (strings.Contains(altaConfig.Memory, "m")) {
        mem, _ := strconv.Atoi(strings.Split(altaConfig.Memory, "m")[0])
        altaSpec.Memory = int64(mem) * 1024 * 1024
    } else if (strings.Contains(altaConfig.Memory, "G")) {
        mem, _ := strconv.Atoi(strings.Split(altaConfig.Memory, "G")[0])
        altaSpec.Memory = int64(mem) * 1024 * 1024 * 1024
    } else if (strings.Contains(altaConfig.Memory, "g")) {
        mem, _ := strconv.Atoi(strings.Split(altaConfig.Memory, "g")[0])
        altaSpec.Memory = int64(mem) * 1024 * 1024 * 1024
    } else {
        mem, _ := strconv.Atoi(altaConfig.Memory)
        altaSpec.Memory = int64(mem)
    }

    // Parse network option
    if (len(altaConfig.Network) == 0) {
        netIf, err := netCtrler.CreateAltaNetIf(altaSpec.AltaId, "default", 0)
        if (err != nil) {
            glog.Errorf("Error creating default network intf for %s", altaSpec.AltaId)
        } else {
            altaSpec.NetworkIfs = []altaspec.AltaNetIf{*netIf}
        }
    } else {
        var netIfs []altaspec.AltaNetIf

        // Loop thru each network name
        for indx, networkName := range altaConfig.Network {
            netIf, err := netCtrler.CreateAltaNetIf(altaSpec.AltaId, networkName, indx)
            if (err != nil) {
                glog.Errorf("Error creating intf for %s, network %s", altaSpec.AltaId, networkName)
            } else {
                netIfs = append(netIfs, *netIf)
            }
        }

        // Set the network intf list
        altaSpec.NetworkIfs = netIfs
    }

    // Default volumes to mount
    altaSpec.Volumes = []altaspec.AltaVolumeBind{
        {
            DatastoreType: "PersistentVolume",
            DatastoreVolumeId: altaSpec.AltaId,
            BindMountPoint: "/var/data",
        },
        {
            DatastoreType: "HostVolume",
            DatastoreVolumeId: altaSpec.AltaId + ".log",
            BindMountPoint: "/var/log",
        },
    }
}

// Create a new Alta container
func CreateAlta(altaConfig *altaspec.AltaConfig) (*AltaActor, error) {
    var altaSpec    altaspec.AltaSpec

    // Check if a name was specified and a container of this name already exists
    if (altaConfig.Name != "") {
        if (altaCtrl.altaNameDb[altaConfig.Name] != nil) {
            glog.Errorf("Error: Alta Container %s already exists", altaConfig.Name)
            return nil, errors.New("Alta container already exists")
        }
    }

    glog.Infof("Creating alta with config: %#v", altaConfig)


    //Create a unique Id
    altaSpec.AltaId = genAltaId()

    // Initialize the parameters
    buildAltaSpec(altaConfig, &altaSpec)


    // Create a new container
    alta, err := NewAlta(&altaSpec)
    if (err != nil) {
        glog.Errorf("Error creating alta: %+v. Err: %v", altaConfig, err)
        return nil, err
    }

    // Save the container in the DB
    altaCtrl.altaDb[alta.AltaId] = alta
    if (altaConfig.Name != "") {
        altaCtrl.altaNameDb[altaConfig.Name] = alta
    }

    // post schedule event
    alta.AltaEvent("schedule")

    return alta, nil
}

// Return a list of all alta containers
func ListAlta() []*AltaModel {
    var altaList []*AltaModel

    // Append each alta actor's model
    for _, alta := range altaCtrl.altaDb {
        altaList = append(altaList, &alta.Model)
    }

    return altaList
}


// Restore Alta actor state from cStore
func RestoreAltaActors() error {
    // Get the list of elements
    jsonArr, err := altaCtrl.cStore.ListDir("alta")
    if (err != nil) {
        glog.Errorf("Error restoring alta actor state")
        return err
    }

    // Loop thru each alta model
    for _, elemStr := range jsonArr {
        // Parse the json model
        var model AltaModel
        err = json.Unmarshal([]byte(elemStr), &model)
        if (err != nil) {
            glog.Errorf("Error parsing object %s, Err %v", elemStr, err)
            return err
        }

        // Create an actor for the alta container
        alta, err := NewAlta(&model.Spec)
        if (err != nil) {
            glog.Errorf("Error restoring Alta %s. Err: %v", model.Spec.AltaId, err)
            return err
        }

        // Restore state
        alta.Model.CurrNode = model.CurrNode
        alta.Model.Fsm.FsmState = model.Fsm.FsmState

        // Save the container in the DB
        altaCtrl.altaDb[alta.AltaId] = alta
        if (model.Spec.AltaName != "") {
            altaCtrl.altaNameDb[model.Spec.AltaName] = alta
        }

        glog.Infof("Restored alta: %#v", alta)
    }

    return nil
}
