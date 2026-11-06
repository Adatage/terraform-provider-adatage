package main

import (
	"context"
	"flag"
	"log"

	"github.com/adatage/terraform-provider-adatage/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Set by GoReleaser.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/adatage/adatage",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
