package flat

import "github.com/codegangsta/cli"

// Exported Flag Opts
var (
	//
	FlaggwIP = cli.StringFlag{Name: "gateway", Value: cligwIP, Usage: "IP of the default gateway."}
	FlagIP   = cli.StringFlag{Name: "ip", Value: cliIP, Usage: "IP of the container"}
	FlagIF   = cli.StringFlag{Name: "host-interface", Value: cliIF, Usage: "Host interface which create macvlan device"}
)

var (
	cliIP   = "10.10.100.2"
	cliIF   = "eth0"        // parent interface to the macvlan iface
	cligwIP = "10.10.100.1" // this is the address of an external route
	cliMTU  = 1500          // generally accepted default MTU
)
