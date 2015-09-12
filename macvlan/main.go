package main

import (
	"net"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"github.com/fmzhen/docker-macvlan/macvlan/flat"
	"github.com/fmzhen/docker-macvlan/macvlan/utils"
	"github.com/fmzhen/macvlan-docker-plugin/plugin/macvlan"
	"github.com/gorilla/mux"
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
	}
	app.Flags = []cli.Flag{
		flagDebug,
		flagSocket,
		macvlan.FlagMacvlanMode,
		macvlan.FlagGateway,
		macvlan.FlagBridgeSubnet,
		macvlan.FlagMacvlanEth,
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
	utils.InitSock(socketFile, socketPath)
	return nil
}

// Run initializes the driver
func Run(ctx *cli.Context) {
	absSocket := fmt.Sprint(pluginPath, ctx.String("socket"))

	listener, err := net.Listen("unix", absSocket)
	if err != nil {
		return err
	}
	router := mux.NewRouter()
	return http.Serve(listener)
}
