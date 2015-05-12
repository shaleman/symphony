package netutils

import (
    "fmt"
    "os"
    "path"
    "strconv"
    "runtime"
    "net"
    "errors"

    "github.com/golang/glog"
    "github.com/vishvananda/netlink"
    "github.com/vishvananda/netns"

)

// Move an interface to new network namespace
func MoveIntfToNetns(portName string, contPid int) error {
    netnsDir := "/var/run/netns"

    // Create /var/run/netns directory if it doesnt exist
    err := os.Mkdir(netnsDir, 0700)
    if err != nil && !os.IsExist(err) {
        glog.Errorf("error creating '%s' direcotry \n", netnsDir)
        return err
    }

    // Remove any old symlinks
    netnsPidFile := path.Join(netnsDir, strconv.Itoa(contPid))
    fmt.Printf("netnsPidFile: %s\n", netnsPidFile)
    err = os.Remove(netnsPidFile)
    if err != nil && !os.IsNotExist(err) {
        glog.Errorf("error removing file '%s' \n", netnsPidFile)
        return err
    }

    // Create new symlink
    procNetNs := path.Join("/proc", strconv.Itoa(contPid), "ns/net")
    err = os.Symlink(procNetNs, netnsPidFile)
    if err != nil {
        glog.Errorf("error symlink file '%s' with '%s' \n", netnsPidFile)
        return err
    }

    // Change namespace
    err = SetInterfaceNamespace(portName, contPid)
    if err != nil {
        glog.Errorf("unable to move interface '%s' to pid %d. Err:%v \n", portName, contPid, err)
        return err
    }

    /* // Note: This is another variation of changing namespace
    targetns, err := netns.GetFromName(strconv.Itoa(contPid))
    if err != nil {
        glog.Errorf("Error getting targetns. Err %v\n", err)
        return err
    }
    defer targetns.Close()

    if err = SetInterfaceInNamespaceFd(portName, uintptr(int(targetns))); err != nil {
        glog.Errorf("Unable to move interface '%s' to pid %d. Err:%v \n", portName, contPid, err)
        return err
    }
    */

    return nil
}

type NetnsIntfIdentify struct {
    PortName        string
    MacAddr         string
    IPAddr          string
    NetmaskLen      int
    DefaultGw       string
}

// Set Interface identity including its name, mac addr & ip addr within a network namespace
func SetNetnsIntfIdentity(nsPid int, portName string, identity NetnsIntfIdentify) error {
    // Lock the OS Thread so we don't accidentally switch namespaces
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()

    origns, err := netns.Get()
    if err != nil {
        glog.Errorf("Error getting current network namespace")
        return err
    }
    defer origns.Close()

    targetns, err := netns.GetFromName(strconv.Itoa(nsPid))
    if err != nil {
        glog.Errorf("Error getting network namespace for container")
        return err
    }
    defer targetns.Close()

    if err = netns.Set(targetns); err != nil {
        glog.Errorf("Error switching network namespace")
        return err
    }
    defer netns.Set(origns)

    if err = InterfaceDown(portName); err != nil {
        glog.Errorf("Error bringing down interface")
        return err
    }

    newPortName := portName
    /* FIXME: Dont rename the interface for now
              OVS seem to behave strangely when you rename the interface
    if err = ChangeInterfaceName(portName, newPortName); err != nil {
        return err
    }
    */

    ipAddrMask := identity.IPAddr + "/" + strconv.Itoa(identity.NetmaskLen)
    if err = SetInterfaceIp(newPortName, ipAddrMask); err != nil {
        glog.Errorf("Error setting interface IP")
        return err
    }

    if err = SetInterfaceMac(newPortName, identity.MacAddr); err != nil {
        glog.Errorf("Error setting interface Mac")
        return err
    }

    if err = InterfaceUp(newPortName); err != nil {
        glog.Errorf("Error setting interface Up")
        return err
    }

    if err = SetDefaultGateway(identity.DefaultGw, newPortName); err != nil {
        glog.Errorf("Error setting default GW")
        // FIXME: setting default gw fails if container has more than one intf
        // return err
    }

    return nil
}

func InterfaceUp(name string) error {
    iface, err := netlink.LinkByName(name)
    if err != nil {
        return err
    }
    return netlink.LinkSetUp(iface)
}

func InterfaceDown(name string) error {
    iface, err := netlink.LinkByName(name)
    if err != nil {
        return err
    }
    return netlink.LinkSetDown(iface)
}

func ChangeInterfaceName(old, newName string) error {
    iface, err := netlink.LinkByName(old)
    if err != nil {
        return err
    }
    return netlink.LinkSetName(iface, newName)
}

func SetInterfaceNamespace(name string, nsPid int) error {
    iface, err := netlink.LinkByName(name)
    if err != nil {
        return err
    }
    return netlink.LinkSetNsPid(iface, nsPid)
}

func SetInterfaceInNamespaceFd(name string, fd uintptr) error {
    iface, err := netlink.LinkByName(name)
    if err != nil {
        return err
    }
    return netlink.LinkSetNsFd(iface, int(fd))
}

func SetDefaultGateway(ip, ifaceName string) error {
    iface, err := netlink.LinkByName(ifaceName)
    if err != nil {
        return err
    }
    gw := net.ParseIP(ip)
    if gw == nil {
        return errors.New("Invalid gateway address")
    }

    _, dst, err := net.ParseCIDR("0.0.0.0/0")
    if err != nil {
        return err
    }
    defaultRoute := &netlink.Route{
        LinkIndex: iface.Attrs().Index,
        Dst:       dst,
        Gw:        gw,
    }
    return netlink.RouteAdd(defaultRoute)
}

func SetInterfaceMac(name string, macaddr string) error {
    iface, err := netlink.LinkByName(name)
    if err != nil {
        return err
    }
    hwaddr, err := net.ParseMAC(macaddr)
    if err != nil {
        return err
    }
    return netlink.LinkSetHardwareAddr(iface, hwaddr)
}

func SetInterfaceIp(name string, rawIp string) error {
    iface, err := netlink.LinkByName(name)
    if err != nil {
        return err
    }

    ipNet, err := netlink.ParseIPNet(rawIp)
    if err != nil {
        return err
    }
    addr := &netlink.Addr{ipNet, ""}
    return netlink.AddrAdd(iface, addr)
}

func SetMtu(name string, mtu int) error {
    iface, err := netlink.LinkByName(name)
    if err != nil {
        return err
    }
    return netlink.LinkSetMTU(iface, mtu)
}
