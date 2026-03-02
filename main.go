package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/ola-silicon/krutrim-terraform/internal"
)

var (
	version string = "dev"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/ola-silicon/krutrim",
		Debug:   debug,
	}

	err := providerserver.Serve(
		context.Background(),
		internal.NewProvider(version),
		opts,
	)

	if err != nil {
		log.Fatal(err.Error())
	}
}
