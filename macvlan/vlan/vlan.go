package vlan

import (
	"log"

	"github.com/codegangsta/cli"
)

func CreateVlan(ctx *cli.Context) {
	err := ParseParam(ctx)
	if err != nil {
		log.Fatalf("Parse param error:", err)
	}
	AddContainerNetworking()
}
