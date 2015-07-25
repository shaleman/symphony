package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/contiv/symphony/pkg/altaspec"
	"github.com/contiv/symphony/pkg/libdocker"
	"github.com/contiv/symphony/pkg/confStore/confStoreApi"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

type HttpApiFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error)

// Create a HTTP Server and initialize the router
func createServer(port int) {
	listenAddr := ":" + strconv.Itoa(port)

	// Create a router
	router := createRouter()

	log.Infof("HTTP server listening on %s", listenAddr)

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
			"/node": 		httpGetNodeInfo,
			"/alta":          httpGetAltaList,
			"/alta/{altaId}": httpGetAltaInfo,
		},
		"POST": {
			"/node/register": 		 httpPostNodeRegister,
			"/image/ispresent": 	 httpPostIsImagePresent,
			"/image/pull": 			 httpPostImagePull,
			"/alta/create":          httpPostAltaCreate,
			"/alta/{cntId}/update":  httpPostAltaUpdate,
			"/alta/{altaId}/start":  httpPostAltaStart,
			"/alta/{altaId}/stop":   httpPostAltaStop,
			"/network/create":       httpPostNetworkCreate,
			"/peer/{peerAddr}":      httpPostPeerAdd,
			"/volume/create":        httpPostVolumeCreate,
			"/volume/mount":         httpPostVolumeMount,
			"/volume/unmount":       httpPostVolumeUnmount,
		},
		"DELETE": {
			"/alta/{altaId}":         httpRemoveAlta,
			"/images/{altaId}":       httpRemoveImage,
			"/network/{networkName}": httpRemoveNetwork,
			"/peer/{peerAddr}":       httpRemovePeer,
		},
	}

	// Register each method/path
	for method, routes := range routeMap {
		for route, funct := range routes {
			log.Infof("Registering %s %s", method, route)

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

// return true if route is a periodic route
// used for suppressing debug messages..
func routeIsPeriodic(localMethod string, localRoute string) bool {
	switch localMethod {
	case "GET":
		switch localRoute {
		case "/alta":
			return true
		}
	}

	return false
}

// Simple Wrapper for http handlers
func makeHttpHandler(localMethod string, localRoute string, handlerFunc HttpApiFunc) http.HandlerFunc {
	// Create a closure and return an anonymous function
	return func(w http.ResponseWriter, r *http.Request) {
		// log the request
		if !routeIsPeriodic(localMethod, localRoute) {
			log.Infof("Calling %s %s", localMethod, localRoute)
			log.Infof("%s %s", r.Method, r.RequestURI)
		}

		// Call the handler
		resp, err := handlerFunc(w, r, mux.Vars(r))
		if err != nil {
			// Log error
			log.Errorf("Handler for %s %s returned error: %s", localMethod, localRoute, err)

			// Send HTTP response
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			respJson, _ := json.Marshal(resp)
			if !routeIsPeriodic(localMethod, localRoute) {
				log.Infof("Handler for %s %s returned Resp: %s", localMethod, localRoute, respJson)
			}

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

// Get a list of all running Alta containers
func httpGetAltaList(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Debugf("Received GET alta list: %+v", vars)

/* DPRECATED
	altaList := make([]*altaspec.AltaContext, 0)
	altaMap := altaMgr.ListAlta()
	for _, altaState := range altaMap {
		actx := altaspec.AltaContext{
			AltaId: altaState.AltaId
			ContainerId: altaState.ContainerId
		}
		altaList = append(altaList, actx)
	}
	*/

	// Get a list of containers
	containerList, err := altaMgr.ListContainers()
	if err != nil {
		log.Errorf("Error getting container list %v. Retrying..", err)
		return nil, err
	}

	log.Debugf("Got container list %+v", containerList)

	// Build a list of container ctx
	var altaList []altaspec.AltaContext
	for _, cid := range containerList {
		var altaId string
		altaState := altaMgr.FindAltaByContainerId(cid)
		if altaState == nil {
			altaId = ""
		} else {
			altaId = altaState.AltaId
		}
		altaContext := altaspec.AltaContext{
			AltaId: altaId,
			ContainerId: cid,
		}

		altaList = append(altaList, altaContext)
	}

	return altaList, nil
}

// Get info about specific Alta container
func httpGetAltaInfo(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Infof("Received GET alta info: %+v", vars)

	return nil, nil
}

// Check if an image present on the host
func httpPostIsImagePresent(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Infof("Received POST isImagePresent: %+v", vars)
	var imgName string

	// Get parameters from the request
	err := json.NewDecoder(r.Body).Decode(&imgName)
	if err != nil {
		log.Errorf("Error decoding isImagePresent request. Err %v", err)
		return nil, err
	}

	isPresent, err := libdocker.IsImagePresent(imgName)
	if err != nil {
		log.Errorf("Error checking if image present. Err %v", err)
	}

	// Create a struct to return
	isPresentResp := altaspec.ReqSuccess{
		Success: isPresent,
	}

	// Return the struct
	return isPresentResp, err
}

// Pull an image from the registry
func httpPostImagePull(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Infof("Received POST image pull request. params: %+v", vars)

	var imgName string

	// Get parameters from the request
	err := json.NewDecoder(r.Body).Decode(&imgName)
	if err != nil {
		log.Errorf("Error decoding isImagePresent request. Err %v", err)
		return nil, err
	}

	// Ask docker to pull the image
	err = libdocker.PullImage(imgName)
	if err != nil {
		log.Errorf("Error pulling image %s, err %v", imgName, err)
		return nil, err
	}

	// Create a struct to return
	imgPullResp := altaspec.ReqSuccess{
		Success: true,
	}

	return imgPullResp, err
}

// Create a Alta container
func httpPostAltaCreate(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	var createReq altaspec.AltaSpec

	// Get Alta parameters from the request
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&createReq)
	if err != nil {
		log.Errorf("Error decoding create request. Err %v", err)
		return nil, err
	}

	// Ask altaMgr to create it
	alta, err := altaMgr.CreateAlta(createReq)
	if err != nil {
		log.Errorf("Error creating Alta %+v. Error %v", createReq, err)
		return nil, err
	}

	return alta, err
}

// Update info about an existing container
func httpPostAltaUpdate(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	var altaSpec altaspec.AltaSpec

	containerId := vars["cntId"]

	// Get Alta parameters from the request
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&altaSpec)
	if err != nil {
		log.Errorf("Error decoding create request. Err %v", err)
		return nil, err
	}

	// Ask altaMgr to create it
	err = altaMgr.UpdateAltaInfo(containerId, altaSpec)
	if err != nil {
		log.Errorf("Error updating Alta %+v. Error %v", altaSpec, err)
		return nil, err
	}

	// Create response
	updateResp := altaspec.ReqSuccess{
		Success: true,
	}

	// Send response
	return updateResp, nil
}

// Start a alta container
func httpPostAltaStart(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	// Get altaId
	altaId := vars["altaId"]

	// Start the alta
	err := altaMgr.StartAlta(altaId)
	if err != nil {
		log.Errorf("Error starting Alta %s, Error: %v", altaId, err)
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
	if err != nil {
		log.Errorf("Error starting Alta %s, Error: %v", altaId, err)
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
	if err != nil {
		log.Errorf("Error removing Alta %s, Error: %v", altaId, err)
		return nil, err
	}

	// Create response
	removeResp := altaspec.ReqSuccess{
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
	var netReq altaspec.AltaNetSpec

	// Get Alta parameters from the request
	err := json.NewDecoder(r.Body).Decode(&netReq)
	if err != nil {
		log.Errorf("Error decoding network create request. Err %v", err)
		return nil, err
	}

	// Create the network
	err = netAgent.CreateNetwork(netReq)
	if err != nil {
		log.Errorf("Error creating network %s, Error: %v", netReq.NetworkName, err)
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
	if err != nil {
		log.Errorf("Error deleting network %s, Error: %v", networkName, err)
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
	if err != nil {
		log.Errorf("Error decoding create volume request. Err %v", err)
		return nil, err
	}

	// Ask altaMgr to create it
	err = volumeAgent.CreateVolume(volumeSpec)
	if err != nil {
		log.Errorf("Error creating volume %+v. Error %v", volumeSpec, err)
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
	if err != nil {
		log.Errorf("Error decoding mount volume request. Err %v", err)
		return nil, err
	}

	// Ask altaMgr to create it
	err = volumeAgent.MountVolume(volumeSpec)
	if err != nil {
		log.Errorf("Error mounting volume %+v. Error %v", volumeSpec, err)
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
	if err != nil {
		log.Errorf("Error decoding unmount volume request. Err %v", err)
		return nil, err
	}

	// Ask altaMgr to create it
	err = volumeAgent.UnmountVolume(volumeSpec)
	if err != nil {
		log.Errorf("Error unmounting volume %+v. Error %v", volumeSpec, err)
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
	log.Infof("Received GET node info")

	nodeInfoResp := clusterAgent.getNodeSpec()

	return nodeInfoResp, nil
}

// Register the node with zeus
func httpPostNodeRegister(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	var masterInfo confStoreApi.ServiceInfo

	// Get Alta parameters from the request
	err := json.NewDecoder(r.Body).Decode(&masterInfo)
	if err != nil {
		log.Errorf("Error decoding unmount volume request. Err %v", err)
		return nil, err
	}

	// Add the master
	err = clusterAgent.addMaster(masterInfo)
	if err != nil {
		log.Errorf("Error adding master info. Err: %v", err)
	}

	// Get the node spec
	nodeInfoResp := clusterAgent.getNodeSpec()

	return nodeInfoResp, nil
}

// Add a peer host
func httpPostPeerAdd(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	// Get the peer ip address
	peerAddr := vars["peerAddr"]

	err := netAgent.AddPeerHost(peerAddr)
	if err != nil {
		log.Errorf("Error adding peer host %s. Err: %v", peerAddr, err)
		return nil, err
	}

	addResp := altaspec.ReqSuccess{
		Success: true,
	}

	return addResp, nil
}

// Remove a peer host
func httpRemovePeer(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	// Get the peer ip address
	peerAddr := vars["peerAddr"]

	err := netAgent.RemovePeerHost(peerAddr)
	if err != nil {
		log.Errorf("Error removing peer host %s. Err: %v", peerAddr, err)
		return nil, err
	}

	delResp := altaspec.ReqSuccess{
		Success: true,
	}

	return delResp, nil
}
