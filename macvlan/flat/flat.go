package flat

import (
	"errors"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/libnetwork/netutils"
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

	hostmacvlanname := netutils.GenerateRandomName(hostprefix, hostlen)
	hostEth, _ := netlink.LinkByName(macvlanEthIface)
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
		log.Warnf("failed to create Macvlan: [ %v ] with the error: %s", macvlandev, err)
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
	macvlandev, _ = netlink.LinkByName(macvlandev.Attrs().Name)
	netlink.LinkSetName(macvlandev, "eth1")
	addr, _ := netlink.ParseAddr(cliIP)
	netlink.AddrAdd(macvlandev, addr)
	netlink.LinkSetUp(veth1)
}
