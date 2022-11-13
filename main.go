package main

import (
	"context"

  "github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/lemaitre/terraform-provider-cmd/cmd"
)

// Provider documentation generation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name hashicups

func main() {
	providerserver.Serve(context.Background(), cmd.New, providerserver.ServeOpts{
		Address: "lemaitre.re/lemaitre/cmd",
	})
}
