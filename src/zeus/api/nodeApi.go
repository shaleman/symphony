package api

import (
    "net/http"

    "zeus/nodeCtrler"
)

// Get a list of nodes
func httpGetNodeList(w http.ResponseWriter, r *http.Request,
                    vars map[string]string) (interface{}, error) {
    // Get the node list
    nodeList := nodeCtrler.ListNodes()

    // return the list
    return nodeList, nil
}
