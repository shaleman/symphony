package rsrcMgr

import (
	"errors"

	"github.com/golang/glog"
	"github.com/jainvipin/bitset"
)

// Wait in a loop for resource user or allocator message
func rsrcMgrLoop() {
	for {
		select {
		case prvdMsg := <-rsrcMgr.providerChan:
			rsrcProvideMsg(prvdMsg)

		case useMsg := <-rsrcMgr.userChan:
			rsrcUseMsg(useMsg)
		}
	}
}

// Handle a resource provider message
func rsrcProvideMsg(prvdMsg ResourceProvideMsg) {
	for _, rsrcPrvd := range prvdMsg.ResourceList {
		if prvdMsg.RsrcOp == "add" {
			// Add a new provider
			err := rsrcProviderAdd(rsrcPrvd)
			if err != nil {
				glog.Errorf("Error adding resource provider: %+v", rsrcPrvd)

				// Send an error response
				rsrcProvideResp(prvdMsg, err)
				return
			}
		} else if prvdMsg.RsrcOp == "remove" {
			// Remove an existing provider
			err := rsrcProviderRemove(rsrcPrvd)
			if err != nil {
				glog.Errorf("Error removing resource provider: %+v", rsrcPrvd)

				// Send an error response
				rsrcProvideResp(prvdMsg, err)
				return
			}
		} else {
			glog.Fatalf("Unknown resource Op: %s in %+v", prvdMsg.RsrcOp, rsrcPrvd)
		}
	}

	// Send a success response
	rsrcProvideResp(prvdMsg, nil)
}

// Send a response
func rsrcProvideResp(prvdMsg ResourceProvideMsg, err error) {
	// Format the message
	resp := ResourceProvideResp{
		Error: err,
	}

	glog.Infof("Sending response: %+v to message: %+v", resp, prvdMsg)

	// Send it on resp channel
	prvdMsg.RespChan <- resp
}

// Add a resource provider. If the provider already exists, just update the
// available resources
func rsrcProviderAdd(rsrcPrvd ResourceProvide) error {
	rsrcType := rsrcPrvd.Type
	rcrcProvider := rsrcPrvd.Provider

	// Add the resource type if it doesnt already exist
	if rsrcMgr.rsrcDb[rsrcType] == nil {
		rsrcMgr.rsrcDb[rsrcType] = &Resource{
			Type:      rsrcType,
			Providers: make(map[string]*RsrcProvider),
		}

		glog.Infof("Added Resource: %+v", rsrcMgr.rsrcDb[rsrcType])
	}

	rsrc := rsrcMgr.rsrcDb[rsrcType]

	// Check if the provider already exist
	if rsrc.Providers[rcrcProvider] != nil {
		// FIXME: handle this gracefully
		return nil
	}

	// Add the provider to the DB
	rsrc.Providers[rcrcProvider] = &RsrcProvider{
		Type:      rsrcType,
		Provider:  rcrcProvider,
		UnitType:  rsrcPrvd.UnitType,
		NumRsrc:   rsrcPrvd.NumRsrc,
		UsedRsrc:  0,
		FreeRsrc:  rsrcPrvd.NumRsrc,
		RsrcUsers: make(map[string]*RsrcUser),
	}

	// For descrete units, create a bitmap
	if rsrcPrvd.UnitType == "descrete" {
		rsrc.Providers[rcrcProvider].rsrcBitset = bitset.New(uint(rsrcPrvd.NumRsrc))
	}

	glog.Infof("Added Resource Provider: %+v", rsrc.Providers[rcrcProvider])

	// Store the resource onto confStore
	err := cstoreSaveProvider(rsrc.Providers[rcrcProvider])
	if err != nil {
		glog.Errorf("Error saving provider to conf store")
		return err
	}

	return nil
}

// Restore a resource provider state
func rsrcProviderRestore(provider *RsrcProvider) error {
	rsrcType := provider.Type
	rcrcProvider := provider.Provider

	// Add the resource type if it doesnt already exist
	if rsrcMgr.rsrcDb[rsrcType] == nil {
		rsrcMgr.rsrcDb[rsrcType] = &Resource{
			Type:      rsrcType,
			Providers: make(map[string]*RsrcProvider),
		}

		glog.Infof("Added Resource: %+v", rsrcMgr.rsrcDb[rsrcType])
	}

	// Add the provider to the DB
	rsrc := rsrcMgr.rsrcDb[rsrcType]
	rsrc.Providers[rcrcProvider] = provider

	// For descrete units, restore the bitmap
	if provider.UnitType == "descrete" {
		provider.rsrcBitset = bitset.New(uint(provider.NumRsrc))

		// Set each used rsrc index
		for _, user := range provider.RsrcUsers {
			for _, rsrcIndex := range user.RsrcIndexes {
				provider.rsrcBitset.Set(uint(rsrcIndex))
			}
		}
	}

	glog.Infof("Restored Resource Provider: %+v", rsrc.Providers[rcrcProvider])

	return nil
}

// Remove a resource provider
// If Removed provider still has user on it, this will assert.
// We expect all users to be removed before a provider can be removed
func rsrcProviderRemove(rsrcPrvd ResourceProvide) error {
	rsrcType := rsrcPrvd.Type
	rcrcProvider := rsrcPrvd.Provider

	// Make sure resource type exists
	rsrc := rsrcMgr.rsrcDb[rsrcType]
	if rsrc == nil {
		glog.Fatalf("resource type %s does not exist", rsrcType)
	}

	// Make sure provider exists
	if rsrc.Providers[rcrcProvider] == nil {
		glog.Fatalf("Resource provider %s/%s does not exist", rsrcType, rcrcProvider)
	}

	// Make sure there are no users
	if len(rsrc.Providers[rcrcProvider].RsrcUsers) != 0 {
		glog.Fatalf("Resource Provider %s/%s still has users: %#v", rsrcType,
			rcrcProvider, rsrc.Providers[rcrcProvider].RsrcUsers)
	}

	// remove the resource from conf store
	err := cstoreDelProvider(rsrc.Providers[rcrcProvider])
	if err != nil {
		glog.Errorf("Error removing provider %s from conf store. Err: %v", rcrcProvider, err)
	}

	// finally delete the provider
	delete(rsrc.Providers, rcrcProvider)

	return nil
}

// Handle Resource use message
// This is tricky since resource allocation needs to happen in all-or-nothing
// manner. So, perform it in two phase. first verify we can make the change
// and then actually make the change
func rsrcUseMsg(useMsg ResourceUserMsg) {
	var rsrcRespList []ResourceUseResp

	// First Pass. make sure all operations can succed
	for _, rsrcUse := range useMsg.ResourceList {
		err := rsrcUseCheck(rsrcUse, useMsg.RsrcOp)
		if err != nil {
			glog.Errorf("Error: %v. Resource op %+v can not be performed", err, rsrcUse)

			// Send an error response
			rsrcUseResp(useMsg, rsrcRespList, err)
			return
		}
	}

	// Second pass. Perform the operation. Assert on error here since we dont
	// expect this to fail
	for _, rsrcUse := range useMsg.ResourceList {
		if useMsg.RsrcOp == "alloc" {
			useResp, err := rsrcAlloc(rsrcUse)
			if err != nil {
				glog.Fatalf("FATAL Error allocating resource: %+v. Err: %v", rsrcUse, err)
			}

			// Append to response list
			rsrcRespList = append(rsrcRespList, *useResp)
		} else if useMsg.RsrcOp == "free" {
			useResp, err := rsrcFree(rsrcUse)
			if err != nil {
				glog.Fatalf("FATAL Error freeing resource: %+v. Err: %v", rsrcUse, err)
			}

			// Append to response list
			rsrcRespList = append(rsrcRespList, *useResp)
		} else {
			glog.Fatalf("Unknown resource Op %s, in %+v", useMsg.RsrcOp, rsrcUse)
		}
	}

	// Send a response
	rsrcUseResp(useMsg, rsrcRespList, nil)
}

// Send a response to resource use message
func rsrcUseResp(useMsg ResourceUserMsg, rsrcList []ResourceUseResp, err error) {
	resp := ResourceUseRespMsg{
		ResourceList: rsrcList,
		Error:        err,
	}

	glog.Infof("Sending Resp: %+v to Msg: %+v", resp, useMsg)

	// Send the response
	useMsg.RespChan <- resp
}

// Check if we can perform resource use operation
func rsrcUseCheck(rsrcUse ResourceUse, rsrcOp string) error {
	rsrcType := rsrcUse.Type
	rcrcProvider := rsrcUse.Provider

	// Check we know about resource type
	rsrc := rsrcMgr.rsrcDb[rsrcType]
	if rsrc == nil {
		glog.Errorf("Resource type %s does not exist", rsrcType)
		return errors.New("Resource Type doesnt exist")
	}

	// Check we know about provider
	provider := rsrc.Providers[rcrcProvider]
	if provider == nil {
		glog.Errorf("Resource provider %s/%s does not exist", rsrcType, rcrcProvider)
		return errors.New("Resource Provider does not exist")
	}

	// Nothing more to check if this is not an alloc message
	if rsrcOp != "alloc" {
		return nil
	}

	// Make sure there is enough resource
	if provider.FreeRsrc < rsrcUse.NumRsrc {
		glog.Errorf("Not enough resource available. Req: %+v, Avl: %+v", rsrcUse, provider)
		return errors.New("Not enough resource available")
	}

	// We are done
	return nil
}

// Allocate a resource
func rsrcAlloc(rsrcUse ResourceUse) (*ResourceUseResp, error) {
	rsrcType := rsrcUse.Type
	rcrcProvider := rsrcUse.Provider
	var rsrcUser RsrcUser

	// No error checking here since we would have done that in first phase
	rsrc := rsrcMgr.rsrcDb[rsrcType]
	provider := rsrc.Providers[rcrcProvider]

	// Start building the response
	resp := ResourceUseResp{
		Type:     rsrcType,
		Provider: rcrcProvider,
		NumRsrc:  rsrcUse.NumRsrc,
	}

	// If we had already allocated resource to this user, simply respond with
	// previously allocated value
	if provider.RsrcUsers[rsrcUse.UserKey] != nil {
		resp.RsrcIndexes = provider.RsrcUsers[rsrcUse.UserKey].RsrcIndexes

		glog.Infof("Resource already allocated for %s. Resp: %+v", rsrcUse.UserKey, resp)
		return &resp, nil
	}

	// Allocate resource based on unit type
	if provider.UnitType == "fluid" {
		// Setup the user context
		rsrcUser = RsrcUser{
			UserKey:  rsrcUse.UserKey,
			UsedRsrc: rsrcUse.NumRsrc,
		}

	} else if provider.UnitType == "descrete" {
		var rsrcIndexes []uint64

		// Allocate requested number of resources
		for i := 0; i < int(rsrcUse.NumRsrc); i++ {
			rsrcIndex, found := provider.rsrcBitset.NextClear(0)
			if !found {
				glog.Errorf("Could not allocate resource %s/%s", rsrcType, rcrcProvider)
				return nil, errors.New("Could not allocate resource")
			}

			// Mark the resource as used
			provider.rsrcBitset.Set(rsrcIndex)

			// keep track of each index
			rsrcIndexes = append(rsrcIndexes, uint64(rsrcIndex))
		}

		// Add the indexes to response
		resp.RsrcIndexes = rsrcIndexes

		// Setup user context
		rsrcUser = RsrcUser{
			UserKey:     rsrcUse.UserKey,
			UsedRsrc:    rsrcUse.NumRsrc,
			RsrcIndexes: rsrcIndexes,
		}
	} else {
		glog.Fatalf("Unknown resource unit type: %+v", provider)
	}

	// Save the used resource
	provider.RsrcUsers[rsrcUse.UserKey] = &rsrcUser

	// Consume the resource
	provider.FreeRsrc -= rsrcUse.NumRsrc
	provider.UsedRsrc += rsrcUse.NumRsrc

	// Store the resource change onto confStore
	err := cstoreSaveProvider(provider)
	if err != nil {
		glog.Errorf("Error saving provider to conf store")
		return nil, err
	}

	glog.Infof("Resource allocation resp: %+v for request: %+v", resp, rsrcUse)

	return &resp, nil
}

// Free a resource
func rsrcFree(rsrcUse ResourceUse) (*ResourceUseResp, error) {
	rsrcType := rsrcUse.Type
	rcrcProvider := rsrcUse.Provider

	// No error checking here since we would have done that in first phase
	rsrc := rsrcMgr.rsrcDb[rsrcType]
	provider := rsrc.Providers[rcrcProvider]

	// Start building the response
	resp := ResourceUseResp{
		Type:     rsrcType,
		Provider: rcrcProvider,
		NumRsrc:  rsrcUse.NumRsrc,
	}

	// If we had already allocated resource to this user, simply respond with
	// previously allocated value
	if provider.RsrcUsers[rsrcUse.UserKey] == nil {
		glog.Errorf("Resource %s/%s was not allocated for %s", rsrcType,
			rcrcProvider, rsrcUse.UserKey)
		return nil, errors.New("Resource not allocated for user")
	}

	// Get the user context
	user := provider.RsrcUsers[rsrcUse.UserKey]

	// Free resource based on unit type
	if provider.UnitType == "fluid" {
		// Nothing else to do
	} else if provider.UnitType == "descrete" {
		// Free the allocated resources indexes
		for i := 0; i < int(user.UsedRsrc); i++ {
			rsrcIndex := user.RsrcIndexes[i]

			// Clear the bitset
			provider.rsrcBitset.Clear(uint(rsrcIndex))
		}
	} else {
		glog.Fatalf("Unknown resource unit type: %+v", provider)
	}

	// Remove the user
	delete(provider.RsrcUsers, rsrcUse.UserKey)

	// Free the resource
	provider.FreeRsrc += rsrcUse.NumRsrc
	provider.UsedRsrc -= rsrcUse.NumRsrc

	// Store the resource change onto confStore
	err := cstoreSaveProvider(provider)
	if err != nil {
		glog.Errorf("Error saving provider to conf store")
		return nil, err
	}

	glog.Infof("Resource free resp: %+v for request: %+v", resp, rsrcUse)

	return &resp, nil
}
