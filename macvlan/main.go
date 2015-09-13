package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/fmzhen/docker-macvlan/macvlan/flat"
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
	InitSock(socketFile, socketPath)
	return nil
}

// Run initializes the driver
func Run(ctx *cli.Context) {
	absSocket := fmt.Sprint(socketPath, ctx.String("socket"))

	listener, err := net.Listen("unix", absSocket)
	if err != nil {
		log.Fatalf("net listen error: ", err)
	}
	router := mux.NewRouter()
	http.Serve(listener, router)
}

// initSock create the socket file if it does not already exist
func InitSock(socketFile string, socketPath string) {
	if err := os.MkdirAll(socketPath, 0755); err != nil && !os.IsExist(err) {
		log.Warnf("Could not create net plugin path directory: [ %s ]", err)
	}
	// concatenate the absolute path to the spec file handle
	absFile := fmt.Sprint(socketPath, socketFile)
	// If the plugin socket file already exists, remove it.
	if _, err := os.Stat(absFile); err == nil {
		log.Debugf("socket file [ %s ] already exists, unlinking the old file handle..", absFile)
		RemoveSock(absFile)
	}
	log.Debugf("The plugin absolute path and handle is [ %s ]", absFile)
}

// removeSock if an old filehandle exists remove it
func RemoveSock(absFile string) {
	err := os.RemoveAll(absFile)
	if err != nil {
		log.Fatalf("Unable to remove the old socket file [ %s ] due to: %s", absFile, err)
	}
}
