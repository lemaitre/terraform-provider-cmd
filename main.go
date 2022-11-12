package main

import (
	"context"

  "github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/lemaitre/terraform-provider-cmd/provider"
)

// Provider documentation generation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name hashicups

func main() {
	providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "lemaitre.re/lemaitre/cmd",
	})
}
