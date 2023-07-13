// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"net/http"
)

// Ensure AuthProxy satisfies various provider interfaces.
var _ provider.Provider = &AuthProxy{}

// AuthProxy defines the provider implementation.
type AuthProxy struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Model describes the provider data model.
type Model struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Password types.String `tfsdk:"password"`
	Username types.String `tfsdk:"username"`
}

type ProviderData struct {
	client   *http.Client
	endpoint string
	username string
	password string
}

func (p *AuthProxy) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "authproxy"
	resp.Version = p.version
}

func (p *AuthProxy) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Points to the endpoint of the target authproxy instance",
				Optional:            false,
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Authproxy admin password",
				Optional:            false,
				Sensitive:           true,
				Required:            true,
			}, "username": schema.StringAttribute{
				MarkdownDescription: "Authproxy admin username",
				Optional:            false,
				Required:            true,
			},
		},
	}
}

func (p *AuthProxy) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// if data.Endpoint.IsNull() { /* ... */ }

	// Example providerData configuration for data sources and resources
	resp.DataSourceData = &ProviderData{
		client:   http.DefaultClient,
		endpoint: data.Endpoint.ValueString(),
		password: data.Password.ValueString(),
		username: data.Username.ValueString(),
	}

	resp.ResourceData = &ProviderData{
		client:   http.DefaultClient,
		endpoint: data.Endpoint.ValueString(),
		password: data.Password.ValueString(),
		username: data.Username.ValueString(),
	}
}

func (p *AuthProxy) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTenantResource,
	}
}

func (p *AuthProxy) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewTenantDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AuthProxy{
			version: version,
		}
	}
}
