package macvlan

import "github.com/codegangsta/cli"

// Exported Flag Opts
var (
	FlaggwIP = cli.StringFlag{Name: "gateway", Value: cligwIP, Usage: "IP of the default gateway."}
	FlagIP   = cli.StringFlag{Name: "ip", Value: defaultSubnet, Usage: "subnet for the containers (currently IPv4 support)"}
	FlagIF   = cli.StringFlag{Name: "host-interface", Value: macvlanEthIface, Usage: "the ethernet interface on the underlying OS that will be used as the parent interface that the container will use for external communications"}
)

// Unexported variables
var (
	cliIP   = "10.10.100.2"
	cliIF   = "eth0"        // parent interface to the macvlan iface
	cligwIP = "10.10.100.1" // this is the address of an external route
	cliMTU  = 1500          // generally accepted default MTU
)
