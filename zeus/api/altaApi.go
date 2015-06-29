package api

import (
	"encoding/json"
	"net/http"

	"github.com/contiv/symphony/pkg/altaspec"

	log "github.com/Sirupsen/logrus"
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
	if err != nil {
		log.Errorf("Error decoding create request. Err %v", err)
		return nil, err
	}

	// Create the alta container
	err = altaCtrler.CreateAlta(&altaConfig)
	if err != nil {
		log.Errorf("Error creating alta container(%+v), Err: %v", altaConfig, err)
		return nil, err
	}

	return altaConfig, nil
}
