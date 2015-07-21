// This file is auto generated by modelgen tool
// Do not edit this file manually

package contivModel

import (
	"errors"
	"net/http"
	"encoding/json"
	"github.com/gorilla/mux"
	log "github.com/Sirupsen/logrus"
)

type HttpApiFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error)

type Network struct {
	Key		string		`json:"key,omitempty"`
	Name	string		`json:"name,omitempty"`
	IsPublic	bool		`json:"isPublic,omitempty"`
	IsPrivate	bool		`json:"isPrivate,omitempty"`
	Encap	string		`json:"encap,omitempty"`
	Subnet	string		`json:"subnet,omitempty"`
	Labels	[]string		`json:"labels,omitempty"`
	Links	NetworkLinks		`json:"links,omitempty"`
}

type NetworkLinks struct {
	Tenant	NetworkTenantLink		`json:"tenant,omitempty"`
}

type NetworkTenantLink struct {
	Type	string		`json:"type,omitempty"`
	Key		string		`json:"key,omitempty"`
	tenant		*Tenant		`json:"-"`
}

type Tenant struct {
	Key		string		`json:"key,omitempty"`
	Name	string		`json:"name,omitempty"`
	LinkSets	TenantLinkSets		`json:"link-sets,omitempty"`
}

type TenantLinkSets struct {
	Networks	[]TenantNetworksLinkSet		`json:"networks,omitempty"`
}

type TenantNetworksLinkSet struct {
	Type	string		`json:"type,omitempty"`
	Key		string		`json:"key,omitempty"`
	network		*Network			`json:"-"`
}



type Collections struct {
	networks    map[string]*Network
	tenants    map[string]*Tenant
}

var collections Collections

type Callbacks interface {
	NetworkCreate(network *Network) error
	NetworkDelete(network *Network) error
	TenantCreate(tenant *Tenant) error
	TenantDelete(tenant *Tenant) error
}

var objCallbackHandler Callbacks


func Init(handler Callbacks) {
objCallbackHandler = handler

	collections.networks = make(map[string]*Network)
	collections.tenants = make(map[string]*Tenant)
}


// Simple Wrapper for http handlers
func makeHttpHandler(handlerFunc HttpApiFunc) http.HandlerFunc {
	// Create a closure and return an anonymous function
	return func(w http.ResponseWriter, r *http.Request) {
		// Call the handler
		resp, err := handlerFunc(w, r, mux.Vars(r))
		if err != nil {
			// Log error
			log.Errorf("Handler for %!s(MISSING) %!s(MISSING) returned error: %!s(MISSING)", r.Method, r.URL, err)

			// Send HTTP response
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			// Send HTTP response as Json
			err = writeJSON(w, http.StatusOK, resp)
			if err != nil {
				log.Errorf("Error generating json. Err: %!v(MISSING)", err)
			}
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

// Add all routes for REST handlers
func AddRoutes(router *mux.Router) {
	var route, listRoute string

	// Register network
	route = "/api/networks/{key}/"
	listRoute = "/api/networks/"
	log.Infof("Registering %s", route)
	router.Path(listRoute).Methods("GET").HandlerFunc(makeHttpHandler(httpListNetworks))
	router.Path(route).Methods("GET").HandlerFunc(makeHttpHandler(httpGetNetwork))
	router.Path(route).Methods("POST").HandlerFunc(makeHttpHandler(httpCreateNetwork))
	router.Path(route).Methods("PUT").HandlerFunc(makeHttpHandler(httpCreateNetwork))
	router.Path(route).Methods("DELETE").HandlerFunc(makeHttpHandler(httpDeleteNetwork))

	// Register tenant
	route = "/api/tenants/{key}/"
	listRoute = "/api/tenants/"
	log.Infof("Registering %s", route)
	router.Path(listRoute).Methods("GET").HandlerFunc(makeHttpHandler(httpListTenants))
	router.Path(route).Methods("GET").HandlerFunc(makeHttpHandler(httpGetTenant))
	router.Path(route).Methods("POST").HandlerFunc(makeHttpHandler(httpCreateTenant))
	router.Path(route).Methods("PUT").HandlerFunc(makeHttpHandler(httpCreateTenant))
	router.Path(route).Methods("DELETE").HandlerFunc(makeHttpHandler(httpDeleteTenant))

}

// LIST REST call
func httpListNetworks(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Debugf("Received httpListNetworks: %+v", vars)

	var list []*Network
	for _, obj := range collections.networks {
		list = append(list, obj)
	}

	// Return the list
	return list, nil
}

// GET REST call
func httpGetNetwork(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Debugf("Received httpGetNetwork: %+v", vars)

	key := vars["key"]

	obj := collections.networks[key]
	if obj == nil {
		log.Errorf("network %s not found", key)
		return nil, errors.New("network not found")
	}

	// Return the obj
	return obj, nil
}

// CREATE REST call
func httpCreateNetwork(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Debugf("Received httpGetNetwork: %+v", vars)

	var obj Network
	key := vars["key"]

	// Get object from the request
	err := json.NewDecoder(r.Body).Decode(&obj)
	if err != nil {
		log.Errorf("Error decoding network create request. Err %v", err)
		return nil, err
	}

	// set the key
	obj.Key = key

	// Perform callback
	err = objCallbackHandler.NetworkCreate(&obj)
	if err != nil {
		log.Errorf("NetworkCreate retruned error for: %+v. Err: %v", obj, err)
		return nil, err
	}

	// save it
	collections.networks[key] = &obj

	// Return the obj
	return obj, nil
}

// DELETE rest call
func httpDeleteNetwork(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Debugf("Received httpDeleteNetwork: %+v", vars)

	key := vars["key"]

	obj := collections.networks[key]
	if obj == nil {
		log.Errorf("network %s not found", key)
		return nil, errors.New("network not found")
	}

	// set the key
	obj.Key = key

	// Perform callback
	err := objCallbackHandler.NetworkDelete(obj)
	if err != nil {
		log.Errorf("NetworkDelete retruned error for: %+v. Err: %v", obj, err)
		return nil, err
	}

	// delete it
	delete(collections.networks, key)

	// Return the obj
	return obj, nil
}

// Return a pointer to network from collection
func FindNetwork(key string) *Network {
	obj := collections.networks[key]
	if obj == nil {
		log.Errorf("network %s not found", key)
		return nil
	}

	return obj
}

// LIST REST call
func httpListTenants(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Debugf("Received httpListTenants: %+v", vars)

	var list []*Tenant
	for _, obj := range collections.tenants {
		list = append(list, obj)
	}

	// Return the list
	return list, nil
}

// GET REST call
func httpGetTenant(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Debugf("Received httpGetTenant: %+v", vars)

	key := vars["key"]

	obj := collections.tenants[key]
	if obj == nil {
		log.Errorf("tenant %s not found", key)
		return nil, errors.New("tenant not found")
	}

	// Return the obj
	return obj, nil
}

// CREATE REST call
func httpCreateTenant(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Debugf("Received httpGetTenant: %+v", vars)

	var obj Tenant
	key := vars["key"]

	// Get object from the request
	err := json.NewDecoder(r.Body).Decode(&obj)
	if err != nil {
		log.Errorf("Error decoding tenant create request. Err %v", err)
		return nil, err
	}

	// set the key
	obj.Key = key

	// Perform callback
	err = objCallbackHandler.TenantCreate(&obj)
	if err != nil {
		log.Errorf("TenantCreate retruned error for: %+v. Err: %v", obj, err)
		return nil, err
	}

	// save it
	collections.tenants[key] = &obj

	// Return the obj
	return obj, nil
}

// DELETE rest call
func httpDeleteTenant(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
	log.Debugf("Received httpDeleteTenant: %+v", vars)

	key := vars["key"]

	obj := collections.tenants[key]
	if obj == nil {
		log.Errorf("tenant %s not found", key)
		return nil, errors.New("tenant not found")
	}

	// set the key
	obj.Key = key

	// Perform callback
	err := objCallbackHandler.TenantDelete(obj)
	if err != nil {
		log.Errorf("TenantDelete retruned error for: %+v. Err: %v", obj, err)
		return nil, err
	}

	// delete it
	delete(collections.tenants, key)

	// Return the obj
	return obj, nil
}

// Return a pointer to tenant from collection
func FindTenant(key string) *Tenant {
	obj := collections.tenants[key]
	if obj == nil {
		log.Errorf("tenant %s not found", key)
		return nil
	}

	return obj
}

