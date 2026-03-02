package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/ola-krutrim/terrafrorm-provider-krutrim/internal"
)

var (
	version string = "dev"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/ola-krutrim/krutrim",
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
