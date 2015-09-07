package flat

import (
	"errors"
	"net"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/fmzhen/docker-macvlan/utils"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	minMTU     = 68
	hostprefix = "macvlan"
	hostlen    = 5
)

/*
type FlatConfig struct {
	IP     string
	GW     string
	CName  string
	HostIF string
	MTU    int
}
*/

// ctx.Args()[0] args are not the parameters, fmz。
func Flat(ctx *cli.Context) {
	err := ParseParam(ctx)
	if err != nil {
		log.Fatalf("Parse param error:", err)
	}

}

// log.fatalf will block the process and return errors. so the errors.new can be remove,fmz.
// when no param pass, it will not equal "", because i set the value(default) first, fmz.
func ParseParam(ctx *cli.Context) error {
	if ctx.String("host-interface") == "" {
		log.Fatalf("Required flag [ host-interface ] is missing")
		return errors.New("Required flag [ host-interface ] is missing")
	}

	cliIF = ctx.String("host-interface")
	if ctx.String("ip") == "" || ctx.String("gateway") == "" || ctx.String("container-name") == "" {
		log.Fatalf("Required flag [ ip or gateway or container-name ] is missing")
		return errors.New("Required flag [ ip or gateway or container-name ] is missing")
	}

	cliIP = ctx.String("ip")
	cligwIP = ctx.String("gateway")
	cliCName = ctx.String("container-name")

	if ctx.Int("mtu") <= 0 {
		cliMTU = cliMTU
	} else if ctx.Int("mtu") >= minMTU {
		cliMTU = ctx.Int("mtu")
	} else {
		log.Fatalf("The MTU value passed [ %d ] must be greater than [ %d ] bytes per rfc791", ctx.Int("mtu"), minMTU)
		return errors.New("the mtu must be int")
	}
	return nil
}

//Varify the parameter. TODO: remove
func VerifyParam() {

}

//netlink is not avaible in MAC OS, build fail.
func AddContainerNetworking() {
	if cliIF == "" {
		log.Fatal("the host-interface is missing,please give one")
	}
	if ok := utils.ValidateHostIface(cliIF); !ok {
		log.Fatalf("the host-interface [ %s ] was not found.", cliIF)
	}

	hostmacvlanname := utils.GenerateRandomName(hostprefix, hostlen)
	hostEth, err := netlink.LinkByName(cliIF)
	if err != nil {
		log.Warnf("Error looking up the parent iface [ %s ] mode: [ %s ] error: [ %s ]", macvlanEthIface, mode, err)
	}
	//create the macvlan device
	macvlandev := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        hostmacvlanname,
			ParentIndex: hostEth.Attrs().Index,
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}
	if err := netlink.LinkAdd(macvlandev); err != nil {
		log.Warnf("failed to create Macvlan: [ %v ] with the error: %s", macvlandev.Attrs().Name, err)
	}
	//	log.Infof("Created Macvlan port: [ %s ] using the mode: [ %s ]", macvlan.Name, macvlanMode)
	// ugly, actually ,can get the ns from netns.getfromDocker. the netns have many function, netns.getformpid
	dockerPid := utils.DockerPid(cliCName)
	//the macvlandev can be use directly, don't get netlink.byname again.
	netlink.LinkSetNsPid(macvlandev, dockerPid)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	//get root network namespace
	origns, _ := netns.Get()
	defer origns.Close()

	//enter the docker container network
	dockerNS, _ := netns.GetFromPid(dockerPid)
	defer dockerNS.Close()

	netns.Set(dockerNS)

	// use macvlandev can cause error,need type assertion. netlink.Macvlan not must be netlink.Link,fmz
	macvlandev1, _ := netlink.LinkByName(macvlandev.Attrs().Name)

	// when the eth is up, set name fail,: Device or resource busy
	netlink.LinkSetDown(macvlandev1)
	netlink.LinkSetName(macvlandev1, "eth1")

	addr, err := netlink.ParseAddr(cliIP)
	if err != nil {
		log.Warnf("failed to parse the ip address %v", cliIP)
	}
	netlink.AddrAdd(macvlandev1, addr)
	netlink.LinkSetUp(macvlandev1)
	
	
	defaultgw := &netlink.Route{
		Dst: 
	}
	/* set the default route, have some problem
	ip, _ := net.ParseIP("8.8.8.8")
	routes, _ := netlink.RouteGet(ip)
	for _, r := range routes {
		netlink.RouteDel(&r)
	}
	*/
	// use ip instruct
	netns.Set(origns)
}
