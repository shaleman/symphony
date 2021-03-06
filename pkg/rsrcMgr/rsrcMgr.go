package rsrcMgr

import (
	"encoding/json"

	"github.com/contiv/objmodel/objdb"

	log "github.com/Sirupsen/logrus"
	"github.com/jainvipin/bitset"
)

// Resource allocated to a user
type RsrcUser struct {
	UserKey     string   // Id of the resource user
	UsedRsrc    float64  // number of resources used
	RsrcIndexes []uint64 // for descret resources, index of the resource allocated
}

// Provider of a resource type
type RsrcProvider struct {
	Type       string               // Resource type
	Provider   string               // Resource provider Id
	UnitType   string               // 'descrete' or 'fluid'
	NumRsrc    float64              // Available resource on this provider
	UsedRsrc   float64              // used resources
	FreeRsrc   float64              // Free resources = Num - Used
	rsrcBitset *bitset.BitSet       // Allocated resources
	RsrcUsers  map[string]*RsrcUser // List of users
}

// state for a type of resource
type Resource struct {
	Type      string                   // resource type
	Providers map[string]*RsrcProvider // List of providers
	// NotUsed: TotalRsrc   float64                      // Total number of resources(cache: sum of all providers)
	// NotUsed: UsedRsrc    float64                      // Used resources (cache: sum of all providers)
}

// Resource response (mainly used for allocs)
type ResourceUseResp struct {
	Type        string   // Type of resource
	Provider    string   // Resource provider where resource is from
	NumRsrc     float64  // Number of resources allocated
	RsrcIndexes []uint64 // for descrete resources, index allocated
}

// Response message for resource request
type ResourceUseRespMsg struct {
	Error        error             // nil on success, or an error code
	ResourceList []ResourceUseResp // list of allocated resources
}

// One Resource request
type ResourceUse struct {
	Type     string  // Type of resource
	Provider string  // Resource provider to get the resource from
	UserKey  string  // Uniq key for the user of the resource
	NumRsrc  float64 // Number of resources needed
}

// Resource request messages
type ResourceUserMsg struct {
	RsrcOp       string                  // "alloc" or "free"
	ResourceList []ResourceUse           // List of resources to be requested
	RespChan     chan ResourceUseRespMsg // Channel for the response
}

type ResourceProvideResp struct {
	Error error // nil on success or an error
}

// Provide one resource
type ResourceProvide struct {
	Type     string  // Type of resource
	Provider string  // Resource provider to get the resource from
	UnitType string  // 'descrete' or 'fluid'
	NumRsrc  float64 // Number of resources needed
}

// Resource provider message
type ResourceProvideMsg struct {
	RsrcOp       string                   // "add" or "remove"
	ResourceList []ResourceProvide        // List of resources
	RespChan     chan ResourceProvideResp // response channel
}

// State of resource mgr
type RsrcMgr struct {
	rsrcDb       map[string]*Resource    // DB of resource types
	cdb          objdb.ObjdbApi          // conf store client
	providerChan chan ResourceProvideMsg // Channel for provider msg
	userChan     chan ResourceUserMsg    // Channel for user message
}

// Resource manager
var rsrcMgr *RsrcMgr

// Initialize the resource mgr
func Init(cdb objdb.ObjdbApi) {
	rsrcMgr = new(RsrcMgr)

	// Initialize the state
	rsrcMgr.cdb = cdb
	rsrcMgr.rsrcDb = make(map[string]*Resource)
	rsrcMgr.providerChan = make(chan ResourceProvideMsg, 200)
	rsrcMgr.userChan = make(chan ResourceUserMsg, 200)

	// Start the resource mgr loop
	go rsrcMgrLoop()
}

// Restore resource manager state
func Restore() error {

	log.Infof("Restoring resources..")

	// Get the list of resource providers
	jsonArr, err := rsrcMgr.cdb.ListDir("resource")
	if err != nil {
		log.Errorf("Error getting resources from cdb. Err: %v", err)
		return err
	}

	// Loop thru each provider
	for _, elemStr := range jsonArr {

		log.Infof("Restoring resource provider: %s", elemStr)

		// Parse the json model
		var provider RsrcProvider
		err = json.Unmarshal([]byte(elemStr), &provider)
		if err != nil {
			log.Errorf("Error parsing object %s, Err %v", elemStr, err)
			return err
		}

		// Restore the provider
		err := rsrcProviderRestore(&provider)
		if err != nil {
			log.Errorf("Error restoring provider %+v. Err: %v", provider, err)
			return err
		}
	}

	return nil
}

// Add a resource provider
func AddResourceProvider(rsrcList []ResourceProvide) error {
	// Create response channel
	respChan := make(chan ResourceProvideResp, 1)

	// Build the message to send
	msg := ResourceProvideMsg{
		RsrcOp:       "add",
		ResourceList: rsrcList,
		RespChan:     respChan,
	}

	// Send the message
	rsrcMgr.providerChan <- msg

	// Block on the response
	resp := <-respChan

	return resp.Error
}

// Find a resource provider
func FindResourceProvider(rsrcType string, rsrcProvider string) *RsrcProvider {
	// If the resource type is unknown, return nil
	if rsrcMgr.rsrcDb[rsrcType] == nil {
		return nil
	}

	// Return the provider
	return rsrcMgr.rsrcDb[rsrcType].Providers[rsrcProvider]
}

func ListProviders(rsrcType string) map[string]*RsrcProvider {
	// Check if resource type exists
	if rsrcMgr.rsrcDb[rsrcType] == nil {
		return nil
	}

	return rsrcMgr.rsrcDb[rsrcType].Providers
}

// Remove a resource provider
func RemoveResourceProvider(rsrcList []ResourceProvide) error {
	// Create response channel
	respChan := make(chan ResourceProvideResp, 1)

	// Build the message to send
	msg := ResourceProvideMsg{
		RsrcOp:       "remove",
		ResourceList: rsrcList,
		RespChan:     respChan,
	}

	// Send the message
	rsrcMgr.providerChan <- msg

	// Block on the response
	resp := <-respChan

	return resp.Error
}

// Allocate one or more resources
func AllocResources(rsrcList []ResourceUse) ([]ResourceUseResp, error) {
	// Create response channel
	respChan := make(chan ResourceUseRespMsg, 1)

	// Build the message to send
	msg := ResourceUserMsg{
		RsrcOp:       "alloc",
		ResourceList: rsrcList,
		RespChan:     respChan,
	}

	// Send the message
	rsrcMgr.userChan <- msg

	// Block on the response
	resp := <-respChan

	log.Infof("Received allc resource Resp: %+v", resp)

	return resp.ResourceList, resp.Error
}

// Allocate one or more resources
func FreeResources(rsrcList []ResourceUse) error {
	// Create response channel
	respChan := make(chan ResourceUseRespMsg, 1)

	// Build the message to send
	msg := ResourceUserMsg{
		RsrcOp:       "free",
		ResourceList: rsrcList,
		RespChan:     respChan,
	}

	// Send the message
	rsrcMgr.userChan <- msg

	// Block on the response
	resp := <-respChan

	return resp.Error
}

// ******************* Internal utility functions **********************
// Save a resource provider to conf store
func cdbSaveProvider(provider *RsrcProvider) error {
	// If there is no conf store, just ignore it. mainly for unit testing
	if rsrcMgr.cdb == nil {
		return nil
	}

	// Save it to conf store
	storeKey := "resource/" + provider.Type + "/" + provider.Provider
	err := rsrcMgr.cdb.SetObj(storeKey, provider)
	if err != nil {
		log.Errorf("Error storing object %+v. Err: %v", provider, err)
		return err
	}

	return nil
}

// Delete provider from conf store
func cdbDelProvider(provider *RsrcProvider) error {
	// If there is no conf store, just ignore it. mainly for unit testing
	if rsrcMgr.cdb == nil {
		return nil
	}

	// Save it to conf store
	storeKey := "resource/" + provider.Type + "/" + provider.Provider
	err := rsrcMgr.cdb.DelObj(storeKey)
	if err != nil {
		log.Errorf("Error deleting object %+v. Err: %v", storeKey, err)
		return err
	}

	return nil
}
