package dhcp

import (
	"os/exec"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/fmzhen/docker-macvlan/macvlan/flat"
	"github.com/fmzhen/docker-macvlan/macvlan/utils"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	minMTU     = 68
	hostprefix = "macvlan"
	hostlen    = 5
)

func Dhcp(ctx *cli.Context) {
	dhcpParseParam(ctx)
	AddDHCPNetwork()
}

func dhcpParseParam(ctx *cli.Context) {
	if ctx.String("host-interface") == "" || ctx.String("container-name") == "" {
		log.Fatalf("the host-interface or container-name option is missing")
	}

	if ctx.Int("mtu") <= 0 {
		flat.CliMTU = flat.CliMTU
	} else if ctx.Int("mtu") >= minMTU {
		flat.CliMTU = ctx.Int("mtu")
	} else {
		log.Fatalf("The MTU value passed [ %d ] must be greater than [ %d ] bytes per rfc791", ctx.Int("mtu"), minMTU)
	}

	flat.CliCName = ctx.String("container-name")
	flat.CliIF = ctx.String("host-interface")
}

func AddDHCPNetwork() {

	if ok := utils.ValidateHostIface(flat.CliIF); !ok {
		log.Fatalf("the host-interface [ %s ] was not found.", flat.CliIF)
	}

	hostmacvlanname, _ := utils.GenerateRandomName(hostprefix, hostlen)
	hostEth, _ := netlink.LinkByName(flat.CliIF)

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

	dockerPid := utils.DockerPid(flat.CliCName)
	netlink.LinkSetNsPid(macvlandev, dockerPid)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	//get root network namespace
	origns, _ := netns.Get()
	defer origns.Close()

	//enter the docker container network
	dockerNS, _ := netns.GetFromPid(dockerPid)
	defer dockerNS.Close()

	//get the dochclient name
	dhcpClientPath := "/home/fmzhen/go/src/github.com/fmzhen/docker-macvlan/macvlan/dhcp/dhcpclient.sh"
	out, err := exec.Command(dhcpClientPath).Output()
	if err != nil {
		log.Fatal("exec the dhcpclient.sh error: ", err)
	}
	dhcpClient := string(out)

	// like ip netns exec xxx, just the netns ,so the command in host can exec
	netns.Set(dockerNS)

	// use macvlandev can cause error,need type assertion. netlink.Macvlan not must be netlink.Link,fmz
	macvlandev1, _ := netlink.LinkByName(macvlandev.Attrs().Name)
	netlink.LinkSetDown(macvlandev1)
	netlink.LinkSetName(macvlandev1, "eth1")
	netlink.LinkSetUp(macvlandev1)

	//delete the default route
	routes, _ := netlink.RouteList(nil, netlink.FAMILY_V4)
	for _, r := range routes {
		if r.Dst == nil {
			if err := netlink.RouteDel(&r); err != nil {
				log.Warnf("delete the default error: ", err)
			}
		}
	}

	//it doesn't work, the problem at the atgs don't pass to the shell script
	dhcpReqpath := "/home/fmzhen/go/src/github.com/fmzhen/docker-macvlan/macvlan/dhcp/dhcpReq.sh"
	exec.Command(dhcpReqpath, dhcpClient, string(dockerPid), flat.CliCName).Run()

	netns.Set(origns)
}
