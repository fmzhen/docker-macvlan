package flat

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	minMTU = 68
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
	err := VerifyFlatParam(ctx)
	if err != nil {
		fmt.Print(err)
	} else {

	}
}

// log.fatalf will block the process and return errors. so the errors.new can be remove,fmz.
// when no param pass, it will not equal "", because i set the value(default) first, fmz.
func VerifyFlatParam(ctx *cli.Context) error {
	if ctx.String("host-interface") == "" {
		log.Fatalf("Required flag [ host-interface ] is missing")
		return errors.New("Required flag [ host-interface ] is missing")
	}

	cliIF = ctx.String("host-interface")
	if ctx.String("ip") == "" || ctx.String("gateway") == "" || ctx.String("container-name") == "" {
		log.Fatalf("Required flag [ ip or gateway ] is missing")
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

//netlink is not avaible in MAC. build fail
func AddContainerNetworking() {
	//create the macvlan device
	macvlandev := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        preMoveName,
			ParentIndex: hostEth.Attrs().Index,
		},
		Mode: mode,
	}

}
