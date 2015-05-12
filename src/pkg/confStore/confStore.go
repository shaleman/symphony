package confStore

import (
    "pkg/confStore/confStoreApi"
    "pkg/confStore/etcdClient"

    "github.com/golang/glog"
)

// Create a new conf store
func NewConfStore() confStoreApi.ConfStorePlugin {
    defaultConfStore := "etcd"

    // Initialize all conf store plugins
    etcdClient.InitPlugin()

    // Get the plugin
    confStore := confStoreApi.GetPlugin(defaultConfStore)

    // Initialize the conf store
    err := confStore.Init([]string{})
    if err != nil {
        glog.Errorf("Error initializing confstore plugin. Err: %v", err)
        glog.Fatal("Error initializing confstore plugin")
    }

    return confStore
}
