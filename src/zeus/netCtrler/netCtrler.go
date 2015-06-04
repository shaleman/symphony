package netCtrler

import (
    "net"
    "errors"
    "strconv"

    "zeus/rsrcMgr"

    "pkg/altaspec"
    "pkg/ofnet"

    "github.com/golang/glog"
)

// network endpoint state, aka network interface state
type EndPoint struct {
    EPKey           string           // End point key, AltaId + intf num
    NetworkName     string           // Name of the network this endpoint is in
    MacAddr         net.HardwareAddr // Mac address for the endpoint
    IPv4Addr        net.IP           // IPv4 address assigned to this endpoint
}

// Network state
type Network struct {
    Name            string          // Name of the network
    NetworkId       uint64          // Unique Id allocated to this network
    IPv4Subnet      net.IPNet       // IP Subnet for this network
    IPv4Gateway     net.IP          // Default IPv4 gateway
    DnsAddr         []net.IP        // DNS addresses
    EndPoints       map[string]*EndPoint // List of end points in this network
}

var netCtrl struct {
    networkDb           map[string]*Network     // DB of networks
    IPv4SubnetStart     net.IP          // Starting IP subnet
    DnsAddr             []net.IP        // DNS server list
    ofnetMaster         *ofnet.OfnetMaster  // ofnet master
}

// Initialize network controller
// Network controller stores all resources in rsrcMgr. So, in a way netCtrler
// is just a wrapper around rsrcMgr
func Init() {
    // Setup basic state
    netCtrl.networkDb = make(map[string]*Network)
    netCtrl.IPv4SubnetStart = net.ParseIP("10.200.1.0")
    netCtrl.DnsAddr = []net.IP{net.ParseIP("4.4.4.4"), net.ParseIP("8.8.8.8")}

    // Initialize ofnet master
    netCtrl.ofnetMaster = ofnet.NewOfnetMaster()
    
    glog.Infof("netCtrl: %#v", netCtrl)

    // Check if global network resources are created/restored
    rsrcProvider := rsrcMgr.FindResourceProvider("network", "global")
    if (rsrcProvider == nil) {
        // Global network ids. This indirectly determines the subnet addr
        // Support 1K networks for now
        err := addNetRsrcProvider("network", "global", 1024)
        if (err != nil) {
            glog.Fatalf("Error adding global network resource. Err: %v", err)
        }

        // Add global mac address resource
        // Support 20K mac addresses for now
        err = addNetRsrcProvider("macaddr", "global", 20000)
        if (err != nil) {
            glog.Fatalf("Error adding global macaddr resource. Err: %v", err)
        }

        // Create the default network
        _, err = NewNetwork("default")
        if (err != nil) {
            glog.Fatalf("Error creating default network. Err: %v", err)
        }
    }
}

// Add a network resource provider
func addNetRsrcProvider(rType string, prvdKey string, numRsrc float64) error {
    // provider info
    provider := []rsrcMgr.ResourceProvide {
        {
            Type:        rType,
            Provider:    prvdKey,
            UnitType:    "descrete",
            NumRsrc:     numRsrc,
        },
    }

    // Add the global network resources
    err := rsrcMgr.AddResourceProvider(provider)
    if (err != nil) {
        return err
    }

    return nil
}

// Allocate a single network resource
func allocNetRsrc(rType, prvdKey, userKey string) (uint64, error) {
    // What to allocate
    rsrcList := []rsrcMgr.ResourceUse{
        {
            Type:        rType,
            Provider:    prvdKey,
            UserKey:     userKey,
            NumRsrc:     1,
        },
    }

    // Allocate the resource
    respRsrsList, err := rsrcMgr.AllocResources(rsrcList)
    if (err != nil) {
        return 0, err
    }

    // Save the network Id we received
    return respRsrsList[0].RsrcIndexes[0], nil
}

// Create a new named network
func NewNetwork(name string) (*Network, error) {
    var err error
    // Create the network
    network := new(Network)
    network.Name = name

    // Check if the named network already exists
    if (netCtrl.networkDb[name] != nil) {
        glog.Errorf("Network %s already exists", name)
        return nil, errors.New("Network already exists")
    }

    // Allocate a new network Id
    network.NetworkId, err = allocNetRsrc("network", "global", name)
    if (err != nil) {
        glog.Errorf("Error allocating network id for %s. Err: %v", name, err)
        return nil, err
    }

    // Create subnet address resource for the network
    // assuming /24 and reserve .0, .1 & .255 addresses
    err = addNetRsrcProvider("subnetAddr", name, 253)
    if (err != nil) {
        glog.Fatalf("Error adding global subnet resource. Err: %v", err)
    }

    // Derive subnet IP addr. netmask is always set to /24
    // WARNING: there is a dangerous assumption on IP addresses here
    netLsb := byte(network.NetworkId % 256)
    netMsb := byte(network.NetworkId / 256)
    netSubnet := netCtrl.IPv4SubnetStart
    netSubnet[13] += netMsb; netSubnet[14] += netLsb;
    network.IPv4Subnet = net.IPNet{
        IP: netSubnet,
        Mask: net.IPv4Mask(255, 255, 255, 0),
    }

    // Default GW is at 10.x.x.1
    network.IPv4Gateway = netSubnet;
    network.IPv4Gateway[15] = 1

    // DNS addresses from global state
    network.DnsAddr = netCtrl.DnsAddr

    // init endpoint db
    network.EndPoints = make(map[string]*EndPoint)

    // Store it in global DB
    netCtrl.networkDb[name] = network

    glog.Infof("Created network: %+v", network)

    // done
    return network, nil
}

// Find the named network
func FindNetwork(name string) (*Network, error) {
    if (netCtrl.networkDb[name] == nil) {
        return nil, errors.New("Network not found")
    }

    return netCtrl.networkDb[name], nil
}

// Create a new network end point
func (self *Network) NewEndPoint(epKey string) (*EndPoint, error) {
    // If the end point already exists, just return it
    if (self.EndPoints[epKey] != nil) {
        return self.EndPoints[epKey], nil
    }

    // Create the end point state
    endPoint := new(EndPoint)
    endPoint.EPKey = epKey
    endPoint.NetworkName = self.Name

    // Allocate mac address
    macId, err := allocNetRsrc("macaddr", "global", epKey)
    if (err != nil) {
        glog.Errorf("Error allocating mac address for %s/%s", self.Name, epKey)
        return nil, err
    }
    // Our grand mac addr allocation scheme is to allocate a unique id and then
    // form a mac addr 02:02:02.xx.xx.xx where last 3 bytes come from unique id
    // Note that x2.xx.xx.xx.xx.xx address is a locally administered mac addr
    endPoint.MacAddr = net.HardwareAddr{2, 2, 2, byte((macId >> 16) & 0xff),
                                byte((macId >> 8) & 0xff), byte(macId & 0xff)}

    // Allocate IPv4 address from our subnet
    ipId, err := allocNetRsrc("subnetAddr", self.Name, epKey)
    if (err != nil) {
        glog.Errorf("Error allocating IP address for %s/%s", self.Name, epKey)
        return nil, err
    }

    // IPv4 address scheme is subnet.xx where xx is unique id + 2
    // subnet.0 and subnet.255 are reserved. subnet.1 is used by default gw
    endPoint.IPv4Addr = self.IPv4Subnet.IP
    endPoint.IPv4Addr[15] = byte(ipId + 2)

    // store it in db
    self.EndPoints[epKey] = endPoint

    // done
    return endPoint, nil
}

// Return a AltaNetIf from network name
func CreateAltaNetIf(altaId string, netName string, ifNum int) (*altaspec.AltaNetIf, error) {
    var network *Network
    var err error

    // find or create the network
    network, _ = FindNetwork(netName)
    if (network == nil) {
        // Network doesnt exist, create it
        network, err = NewNetwork(netName)
        if (err != nil) {
            glog.Errorf("Error creating network %s. Err: %v", netName, err)
            return nil, err
        }
    }

    epKey := altaId + "." + strconv.Itoa(ifNum)

    // Create an end point on the networ
    endPoint, err := network.NewEndPoint(epKey)
    if (err != nil) {
        glog.Errorf("Error creating end point %s/%s", netName, epKey)
        return nil, err
    }

    // Formulate the network if
    altaNetIf := altaspec.AltaNetIf{
        NetworkName:     network.Name,
        IntfMacAddr:     endPoint.MacAddr.String(),
        IntfIpv4Addr:    endPoint.IPv4Addr.String(),
        IntfIpv4Masklen: 24,
        Ipv4Gateway:     network.IPv4Gateway.String(),
    }

    // done
    return &altaNetIf, nil
}
