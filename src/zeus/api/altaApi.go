package api

import (
    "net/http"
    "encoding/json"

    "zeus/altaCtrler"

    "pkg/altaspec"

    "github.com/golang/glog"
)

func httpGetAltaList(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {

    // Get a list of Alta containers
    altaList := altaCtrler.ListAlta()

    // Return the list
    return altaList, nil
}

// Create an alta container
func httpPostAltaCreate(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error) {
    var altaConfig altaspec.AltaConfig

    // Get Alta config from the request
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&altaConfig)
    if (err != nil) {
        glog.Errorf("Error decoding create request. Err %v", err)
        return nil, err
    }

    // Create the alta container
    alta, err := altaCtrler.CreateAlta(&altaConfig)
    if (err != nil) {
        glog.Errorf("Error creating alta container(%+v), Err: %v", altaConfig, err)
        return nil, err
    }

    return alta.Model, nil
}
