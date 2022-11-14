package cmd

import (
	"context"
	//"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	//"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	//"github.com/hashicorp/terraform-plugin-log/tflog"
)


var (
	_ provider.Provider = &cmdProvider{}
)

// cmdProvider satisfies the tfsdk.Provider interface and usually is included
// with all Resource and DataSource implementations.
type cmdProvider struct {
	// client can contain the upstream provider SDK or HTTP client used to
	// communicate with the upstream service. Resource and DataSource
	// implementations can then make calls using this client.
	//
	// TODO: If appropriate, implement upstream provider SDK or HTTP client.
	// client vendorsdk.ExampleClient

	// configured is set to true at the end of the Configure method.
	// This can be used in Resource and DataSource implementations to verify
	// that the provider was previously configured.
	configured bool

	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// providerData can be used to store data from the Terraform configuration.
type cmdProviderModel struct {
	Example types.String `tfsdk:"example"`
}

func New() provider.Provider {
  return &cmdProvider{}
}

// Metadata returns the provider type name.
func (p *cmdProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cmd"
}

// GetSchema defines the provider-level schema for configuration data.
func (p *cmdProvider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
  return tfsdk.Schema{
    Attributes: map[string]tfsdk.Attribute{
      "example": {
        MarkdownDescription: "Example provider attribute",
        Optional:            true,
        Type:                types.StringType,
      },
    },
  }, nil
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *cmdProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config cmdProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// if data.Example.Null { /* ... */ }

	// If the upstream provider SDK or HTTP client requires configuration, such
	// as authentication or logging, this is a great opportunity to do so.

	p.configured = true
}

// DataSources defines the data sources implemented in the provider.
func (p *cmdProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
	}
}

// Resources defines the resources implemented in the provider.
func (p *cmdProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
    func() resource.Resource {
      return &cmdResource{
        shell: nil,
        shellFactory: shellLocalFactory,
      }
    },
    func() resource.Resource {
      return &cmdResource{
        shell: nil,
        shellFactory: shellSshFactory,
      }
    },
	}
}
