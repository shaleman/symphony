package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/golang/glog"
	"github.com/socketplane/bonjour"
	"github.com/socketplane/ecc"
)

const DOCKER_CLUSTER_SERVICE = "_docker._cluster"
const DOCKER_CLUSTER_SERVICE_PORT = 9999 //TODO : fix this
const DOCKER_CLUSTER_DOMAIN = "local"

const dataDir = "/tmp/ecc"

func Bonjour(intfName string) {
	b := bonjour.Bonjour{
		ServiceName:   DOCKER_CLUSTER_SERVICE,
		ServiceDomain: DOCKER_CLUSTER_DOMAIN,
		ServicePort:   DOCKER_CLUSTER_SERVICE_PORT,
		InterfaceName: intfName,
		BindToIntf:    true,
		Notify:        notify{},
	}
	b.Start()
}

type notify struct{}

func (n notify) NewMember(addr net.IP) {
	glog.Infof("New Member Added : ", addr)
	JoinDatastore(addr.String())
}
func (n notify) RemoveMember(addr net.IP) {
	glog.Infof("Member Left : ", addr)
}

func JoinDatastore(address string) error {
	return ecc.Join(address)
}

func LeaveDatastore() error {
	if err := ecc.Leave(); err != nil {
		glog.Errorf("Error leaving datastore: %v", err)
		return err
	}
	if err := os.RemoveAll(dataDir); err != nil {
		glog.Errorf("Error deleting data directory %s", err)
		return err
	}
	return nil
}

type eccListener struct {
}

var listener eccListener

func (e eccListener) NotifyNodeUpdate(nType ecc.NotifyUpdateType, nodeAddress string) {
	if nType == ecc.NOTIFY_UPDATE_ADD {
		glog.Infof("New Node joined the cluster : %s", nodeAddress)
	} else if nType == ecc.NOTIFY_UPDATE_DELETE {
		glog.Infof("Node left the cluster : %s", nodeAddress)
	}
}

func (e eccListener) NotifyKeyUpdate(nType ecc.NotifyUpdateType, key string, data []byte) {
}
func (e eccListener) NotifyStoreUpdate(nType ecc.NotifyUpdateType, store string, data map[string][]byte) {
}

func main() {
	// FIXME: Temporary hack for testing
	flag.Lookup("logtostderr").Value.Set("true")

	bootstrap := false
	if (len(os.Args) > 1) && (os.Args[1] == "-bootstrap") {
		bootstrap = true
	}
	glog.Infof("Starting Bonjour service")

	// Initialize Bonjour service
	Bonjour("eth1")

	glog.Infof("Starting Consul and waiting for other members")

	// Initialize ecc
	err := ecc.Start(true, bootstrap, "eth1", dataDir)
	if err == nil {
		glog.Infof("Registering for ecc notifs")
		go ecc.RegisterForNodeUpdates(listener)
	} else {
		glog.Errorf("Error starting ecc. Err %+v", err)
	}

	glog.Infof("Waiting for ecc thread")

	handler := make(chan os.Signal, 1)
	signal.Notify(handler, os.Interrupt)
	go func() {
		for sig := range handler {
			glog.Errorf("Received signal %v. Exiting...", sig)
			os.Exit(0)
		}
	}()

	// Just wait for ecc thread to run
	time.Sleep(time.Hour * 1)
}
