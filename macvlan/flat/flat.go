package flat

import (
	"errors"
	"net"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/fmzhen/docker-macvlan/macvlan/utils"
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
	AddContainerNetworking()
}

// log.fatalf will block the process and return errors. so the errors.new can be remove, or fmt.Errorf(). fmz.
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

	hostmacvlanname, _ := utils.GenerateRandomName(hostprefix, hostlen)
	hostEth, _ := netlink.LinkByName(cliIF)

	//create the macvlan device
	macvlandev := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        hostmacvlanname,
			ParentIndex: hostEth.Attrs().Index,
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}
	if err := netlink.LinkAdd(macvlandev); err != nil {
		log.Fatalf("failed to create Macvlan: [ %v ] with the error: %s", macvlandev.Attrs().Name, err)
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
		log.Fatalf("failed to parse the ip address %v", cliIP)
	}
	netlink.AddrAdd(macvlandev1, addr)
	netlink.LinkSetUp(macvlandev1)

	/* set the default route, have some problem. Dst == 0.0.0.0/0? no
	defaultgw := &netlink.Route{
		Dst: nil,
	}
	netlink.RouteDel(defaultgw)
	ip, _ := net.ParseIP("8.8.8.8")
	routes, _ := netlink.RouteGet(ip)
	for _, r := range routes {
		netlink.RouteDel(&r)
	}
	*/

	//if use ip instruction,  it also can config the container, --privileged have no effect.
	// The sublime test code(test this function) is strange, it only can avaiable in first time. And then fail(even need to reboot)
	// got it,

	//following code successfully delete the default route in docker container,but error in my host ,no such process
	routes, _ := netlink.RouteList(nil, netlink.FAMILY_V4)
	for _, r := range routes {
		if r.Dst == nil {
			if err := netlink.RouteDel(&r); err != nil {
				log.Warnf("delete the default error: ", err)
			}
		}
	}

	if cligwIP == "" {
		log.Fatal("container gw is null")
	}

	defaultRoute := &netlink.Route{
		Dst:       nil,
		Gw:        net.ParseIP(cligwIP),
		LinkIndex: macvlandev1.Attrs().Index,
	}
	if err := netlink.RouteAdd(defaultRoute); err != nil {
		log.Warnf("create default route error: ", err)
	}

	netns.Set(origns)
}
