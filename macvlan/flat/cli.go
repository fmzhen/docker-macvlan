package flat

import "github.com/codegangsta/cli"

// the follew option will see in help doc. can set the default value;
var (
	// flat option
	FlaggwIP          = cli.StringFlag{Name: "gateway", Usage: "IP of the default gateway."}
	FlagIP            = cli.StringFlag{Name: "ip", Usage: "IP of the container"}
	FlagIF            = cli.StringFlag{Name: "host-interface", Usage: "Host interface which create macvlan device"}
	FlagMTU           = cli.IntFlag{Name: "mtu", Value: CliMTU, Usage: "the MTU of macvlan device"}
	FlagContainerName = cli.StringFlag{Name: "container-name", Usage: "the container name which want to config"}
)

// flat config struct.
var (
	CliIP    string //CIDR IP
	CliIF    string // parent interface to the macvlan iface
	CligwIP  string // this is the address of an external route
	CliMTU   = 1500 // generally accepted default MTU
	CliCName string // the Docker container name which want to config
)
