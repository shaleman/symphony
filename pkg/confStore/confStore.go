package confStore

import (
	"github.com/contiv/symphony/pkg/confStore/confStoreApi"
	"github.com/contiv/symphony/pkg/confStore/etcdClient"

	log "github.com/Sirupsen/logrus"
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
		log.Errorf("Error initializing confstore plugin. Err: %v", err)
		log.Fatal("Error initializing confstore plugin")
	}

	return confStore
}
