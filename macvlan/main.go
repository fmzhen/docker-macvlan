package main

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/fmzhen/docker-macvlan/macvlan/daemon"
	"github.com/fmzhen/docker-macvlan/macvlan/dhcp"
	"github.com/fmzhen/docker-macvlan/macvlan/flat"
	"github.com/fmzhen/docker-macvlan/macvlan/vlan"
)

const (
	appVersion = "0.1"
	appName    = "macvlan"
	appUsage   = "Docker Macvlan Networking"

	macvlanSocket = "macvlan.sock"
	socketPath    = "/run/docker/plugins/"
)

func main() {
	var flagSocket = cli.StringFlag{
		Name:  "socket, s",
		Value: macvlanSocket,
		Usage: "listening unix socket",
	}
	var flagDebug = cli.BoolFlag{
		Name:  "debug, d",
		Usage: "enable debugging",
	}

	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = appVersion
	app.Commands = []cli.Command{
		{
			Name:    "flat",
			Aliases: []string{"f"},
			Usage:   "add a docker container to host network",
			Flags: []cli.Flag{
				flat.FlaggwIP,
				flat.FlagIP,
				flat.FlagIF,
				flat.FlagMTU,
				flat.FlagContainerName,
			},
			Action: flat.Flat,
		},
		{
			Name:    "dhcp",
			Aliases: []string{"d"},
			Usage:   "add a docker container to host network use local dhcp server",
			Flags: []cli.Flag{
				flat.FlagIF,
				flat.FlagMTU,
				flat.FlagContainerName,
			},
			Action: dhcp.Dhcp,
		},
		{
			Name:    "create-vlan",
			Aliases: []string{"cv"},
			Usage:   "create a vlan network",
			Flags: []cli.Flag{
				vlan.FlagVlanName,
				vlan.FlagVlanSubnet,
				vlan.FlagHostIF,
			},
			Action: vlan.CreateVlan,
		},
		{
			Name:    "vlan",
			Aliases: []string{"v"},
			Usage:   "config a container with vlan",
			Flags: []cli.Flag{
				vlan.FlagAttachName,
				vlan.FlagContainerName,
			},
			Action: vlan.Vlan,
		},
	}
	app.Flags = []cli.Flag{
		flagDebug,
		flagSocket,
		flat.FlagEtcd,
	}
	app.Before = initEnv
	app.Action = Run
	app.Run(os.Args)
}

//create the socket filepath if it does not already exist
func initEnv(ctx *cli.Context) error {
	socketFile := ctx.String("socket")
	// Default log level is Info
	if ctx.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.SetOutput(os.Stderr)
	daemon.InitSock(socketFile, socketPath)
	return nil
}

// Run initializes the driver
func Run(ctx *cli.Context) {
	absSocket := fmt.Sprint(socketPath, ctx.String("socket"))

	flat.CliEtcd = ctx.String("etcd")
	daemon.Listen(absSocket)
}
