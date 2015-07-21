package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/contiv/symphony/zeus/common"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

type HttpApiFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string) (interface{}, error)

var altaCtrler common.AltaCtrlInterface
var apiCtrler *ApiController

// Create a HTTP Server and initialize the router
func CreateServer(port int, ctrlers *common.ZeusCtrlers) {
	listenAddr := ":" + strconv.Itoa(port)

	altaCtrler = ctrlers.AltaCtrler

	// Create a router
	router := createRouter()

	// Create the API controller
	apiCtrler = NewApiController(router)


	log.Infof("HTTP server listening on %s", listenAddr)

	// Start the HTTP server
	log.Fatal(http.ListenAndServe(listenAddr, router))
}

// Create a router and initialize the routes
func createRouter() *mux.Router {
	// Create a new router instance
	router := mux.NewRouter()

	// serve static files
	router.PathPrefix("/web/").Handler(http.StripPrefix("/web/", http.FileServer(http.Dir("./web/"))))

	// Special case to serve main index.html
	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "./web/index.html")
	})

	// List of routes
	routeMap := map[string]map[string]HttpApiFunc{
		"GET": {
			"/node/": httpGetNodeList,
			"/alta/": httpGetAltaList,
		},
		"POST": {
			"/alta/create": httpPostAltaCreate,
		},
		"DELETE": {
		// "/alta/{altaId}":        httpRemoveAlta,
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

// Simple Wrapper for http handlers
func makeHttpHandler(localMethod string, localRoute string, handlerFunc HttpApiFunc) http.HandlerFunc {
	// Create a closure and return an anonymous function
	return func(w http.ResponseWriter, r *http.Request) {
		// log the request
		log.Debugf("Calling %s %s", localMethod, localRoute)
		log.Debugf("%s %s", r.Method, r.RequestURI)

		// Call the handler
		resp, err := handlerFunc(w, r, mux.Vars(r))
		if err != nil {
			// Log error
			log.Errorf("Handler for %s %s returned error: %s", localMethod, localRoute, err)

			// Send HTTP response
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			respJson, _ := json.Marshal(resp)
			if localMethod == "GET" {
				log.Debugf("Handler for %s %s returned Resp: %s", localMethod, localRoute, respJson)
			} else {
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
