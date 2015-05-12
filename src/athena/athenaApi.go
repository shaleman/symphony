package main

import (
    "os"
    // "fmt"
    "log"
    // "time"
    "strconv"
    "net/http"
    "encoding/json"

    "pkg/libdocker"
    "pkg/altaspec"
    "pkg/psutil"

    "github.com/golang/glog"
    "github.com/gorilla/mux"
)

type HttpApiFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error)


// Create a HTTP Server and initialize the router
func createServer(port int) {
    listenAddr := ":" + strconv.Itoa(port)

    // Create a router
    router := createRouter()

    glog.Infof("HTTP server listening on %s", listenAddr)

    // Start the HTTP server
    log.Fatal(http.ListenAndServe(listenAddr, router))
}

// Create a router and initialize the routes
func createRouter() *mux.Router {
    // Create a new router instance
    router := mux.NewRouter()

    // List of routes
    routeMap := map[string]map[string]HttpApiFunc{
        "GET": {
            "/node":                         httpGetNodeInfo,
            "/image/{imgName}/ispresent":    httpGetIsImagePresent,
            "/alta":                         httpGetAltaList,
            "/alta/{altaId}":                httpGetAltaInfo,
        },
        "POST": {
            "/image/{imgName}/pull":         httpPostImagePull,
            "/alta/create":                  httpPostAltaCreate,
            "/alta/{altaId}/start":          httpPostAltaStart,
            "/alta/{altaId}/stop":           httpPostAltaStop,
            "/network/{networkName}/create": httpPostNetworkCreate,
            "/volume/create":                httpPostVolumeCreate,
            "/volume/mount":                 httpPostVolumeMount,
            "/volume/unmount":               httpPostVolumeUnmount,

        },
        "DELETE": {
            "/alta/{altaId}":        httpRemoveAlta,
            "/images/{altaId}":        httpRemoveImage,
            "/network/{networkName}":    httpRemoveNetwork,
        },
    }

    // Register each method/path
    for method, routes := range routeMap {
        for route, funct := range routes {
            glog.Infof("Registering %s %s", method, route)

            // NOTE: scope issue, make sure the variables are local and won't be changed
            localRoute := route
            localFunct := funct
            localMethod := method

            // Create a closure for the handlers
            f := makeHttpHandler(localMethod, localRoute, localFunct)

            // Register the handler
            router.Path(localRoute).Methods(localMethod).HandlerFunc(f)
        }
    }

    return router
}

// Simple Wrapper for http handlers
func makeHttpHandler(localMethod string, localRoute string, handlerFunc HttpApiFunc) http.HandlerFunc {
    // Create a closure and return an anonymous function
    return func(w http.ResponseWriter, r *http.Request) {
        // log the request
        glog.Infof("Calling %s %s", localMethod, localRoute)
        glog.Infof("%s %s", r.Method, r.RequestURI)

        // Call the handler
        resp, err := handlerFunc(w, r, mux.Vars(r));
        if (err != nil) {
            // Log error
            glog.Errorf("Handler for %s %s returned error: %s", localMethod, localRoute, err)

            // Send HTTP response
            http.Error(w, err.Error(), http.StatusInternalServerError)
        } else {
            respJson, _ := json.Marshal(resp)
            glog.Infof("Handler for %s %s returned Resp: %s", localMethod, localRoute, respJson)

            // Send HTTP response as Json
            writeJSON(w, http.StatusOK, resp)
        }
    }
}

// writeJSON: writes the value v to the http response stream as json with standard
// json encoding.
func writeJSON(w http.ResponseWriter, code int, v interface{}) error {
    // Set content type as json
    w.Header().Set("Content-Type", "application/json")

    // write the HTTP status code
    w.WriteHeader(code)

    // Write the Json output
    return json.NewEncoder(w).Encode(v)
}

// ***********************************************************
//         HTTP Handler functions
// ***********************************************************

// Check if an image present on the host
func httpGetIsImagePresent(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    glog.Infof("Received GET isImagePresent: %+v", vars)

    imgName := vars["imgName"]
    isPresent, err := libdocker.IsImagePresent(imgName)
    if (err != nil) {
        glog.Errorf("Error checking if image present. Err %v", err)
    }

    // Create a struct to return
    isPresentResp := altaspec.ReqSuccess{
        Success: isPresent,
    }

    // Return the struct
    return isPresentResp, err
}

// Get a list of all running Alta containers
func httpGetAltaList(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    glog.Infof("Received GET alta list: %+v", vars)

    altaList := make([]*AltaState, 0)
    altaMap := altaMgr.ListAlta()
    for _, altaState := range altaMap {
        altaList = append(altaList, altaState)
    }
    return altaList, nil
}

// Get info about specific Alta container
func httpGetAltaInfo(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    glog.Infof("Received GET alta info: %+v", vars)

    return nil, nil
}

// Pull an image from the registry
func httpPostImagePull(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    glog.Infof("Received POST image pull request. params: %+v", vars)
    success := true

    imgName := vars["imgName"]

    // Ask docker to pull the image
    err := libdocker.PullImage(imgName)
    if (err != nil) {
        glog.Errorf("Error pulling image %s, err %v", imgName, err)
        success = false
    }

    // Create a struct to return
    imgPullResp := altaspec.ReqSuccess{
        Success: success,
    }

    return imgPullResp, err
}

// Create a Alta container
func httpPostAltaCreate(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    var createReq altaspec.AltaSpec

    // Get Alta parameters from the request
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&createReq)
    if (err != nil) {
        glog.Errorf("Error decoding create request. Err %v", err)
        return nil, err
    }

    // Ask altaMgr to create it
    alta, err := altaMgr.CreateAlta(createReq)
    if (err != nil) {
        glog.Errorf("Error creating Alta %+v. Error %v", createReq, err)
        return nil, err
    }

    return alta, err
}

// Start a alta container
func httpPostAltaStart(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    // Get altaId
    altaId := vars["altaId"]

    // Start the alta
    err := altaMgr.StartAlta(altaId)
    if (err != nil) {
        glog.Errorf("Error starting Alta %s, Error: %v", altaId, err)
        return nil, err
    }

    // Create response
    startResp := altaspec.ReqSuccess{
        Success: true,
    }

    // Send response
    return startResp, nil
}

// Stop a alta container
func httpPostAltaStop(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    // Get altaId
    altaId := vars["altaId"]

    // Stop the alta
    err := altaMgr.StopAlta(altaId)
    if (err != nil) {
        glog.Errorf("Error starting Alta %s, Error: %v", altaId, err)
        return nil, err
    }

    // Create response
    stopResp := altaspec.ReqSuccess{
        Success: true,
    }

    // Send response
    return stopResp, nil
}

// Remove a Alta container
func httpRemoveAlta(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    // Get altaId
    altaId := vars["altaId"]

    // Remove the alta
    err := altaMgr.RemoveAlta(altaId)
    if (err != nil) {
        glog.Errorf("Error removing Alta %s, Error: %v", altaId, err)
        return nil, err
    }

    // Create response
    removeResp :=altaspec.ReqSuccess{
        Success: true,
    }

    // Send response
    return removeResp, nil
}

// Remove an image
func httpRemoveImage(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {

    return nil, nil
}

// Create a network
func httpPostNetworkCreate(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    // Get the network name
    networkName := vars["networkName"]

    // Create the network
    err := netAgent.CreateNetwork(networkName)
    if (err != nil) {
        glog.Errorf("Error creating network %s, Error: %v", networkName, err)
        return nil, err
    }

    // Create response
    createResp := altaspec.ReqSuccess{
        Success: true,
    }

    // Send response
    return createResp, nil
}

// Delete a network
func httpRemoveNetwork(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    // Get the network name
    networkName := vars["networkName"]

    // Create the network
    err := netAgent.DeleteNetwork(networkName)
    if (err != nil) {
        glog.Errorf("Error deleting network %s, Error: %v", networkName, err)
        return nil, err
    }

    // Create response
    deleteResp := altaspec.ReqSuccess{
        Success: true,
    }

    // Send response
    return deleteResp, nil
}

// Create a volume
func httpPostVolumeCreate(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    var volumeSpec altaspec.AltaVolumeSpec

    // Get Alta parameters from the request
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&volumeSpec)
    if (err != nil) {
        glog.Errorf("Error decoding create volume request. Err %v", err)
        return nil, err
    }

    // Ask altaMgr to create it
    err = volumeAgent.CreateVolume(volumeSpec)
    if (err != nil) {
        glog.Errorf("Error creating volume %+v. Error %v", volumeSpec, err)
        return nil, err
    }

    // Create response
    createResp := altaspec.ReqSuccess{
        Success: true,
    }

    return createResp, err
}

// Create a volume
func httpPostVolumeMount(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    var volumeSpec altaspec.AltaVolumeSpec

    // Get Alta parameters from the request
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&volumeSpec)
    if (err != nil) {
        glog.Errorf("Error decoding mount volume request. Err %v", err)
        return nil, err
    }

    // Ask altaMgr to create it
    err = volumeAgent.MountVolume(volumeSpec)
    if (err != nil) {
        glog.Errorf("Error mounting volume %+v. Error %v", volumeSpec, err)
        return nil, err
    }

    // Create response
    createResp := altaspec.ReqSuccess{
        Success: true,
    }

    return createResp, err
}

// Create a volume
func httpPostVolumeUnmount(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    var volumeSpec altaspec.AltaVolumeSpec

    // Get Alta parameters from the request
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&volumeSpec)
    if (err != nil) {
        glog.Errorf("Error decoding unmount volume request. Err %v", err)
        return nil, err
    }

    // Ask altaMgr to create it
    err = volumeAgent.UnmountVolume(volumeSpec)
    if (err != nil) {
        glog.Errorf("Error unmounting volume %+v. Error %v", volumeSpec, err)
        return nil, err
    }

    // Create response
    createResp := altaspec.ReqSuccess{
        Success: true,
    }

    return createResp, err
}

// Check if an image present on the host
func httpGetNodeInfo(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    glog.Infof("Received GET node info")

    // Get the number of CPU
    numCpu, _ := psutil.CPUCounts(true)

    // CPU speed
    cpuInfo, _ := psutil.CPUInfo()
    cpuMhz := uint64(cpuInfo[0].Mhz)

    // Get the total memory
    memInfo, _ := psutil.VirtualMemory()
    memTotal := memInfo.Total

    // Get the host name
    hostName, _ := os.Hostname()

    // Create response
    nodeInfoResp := altaspec.NodeSpec{
        HostName:    hostName,
        NumCpuCores: numCpu,
        CpuMhz:      cpuMhz,
        MemTotal:    memTotal,
    }

    return nodeInfoResp, nil
}
