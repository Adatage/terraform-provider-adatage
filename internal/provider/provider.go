// Package provider is the Terraform and OpenTofu provider of Adatage Cloud.
package provider

import (
	"context"
	"os"

	"github.com/adatage/terraform-provider-adatage/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type adatageProvider struct{ version string }

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &adatageProvider{version: version} }
}

type providerModel struct {
	Project         types.String `tfsdk:"project"`
	Credentials     types.String `tfsdk:"credentials"`
	CredentialsFile types.String `tfsdk:"credentials_file"`
	AccessToken     types.String `tfsdk:"access_token"`
	APIURL          types.String `tfsdk:"api_url"`
}

func (p *adatageProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "adatage"
	resp.Version = p.version
}

func (p *adatageProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Adatage Cloud: compute, networking, load balancing, Kubernetes and access. Sign in with the credentials file of a service account.",
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{Optional: true, Description: "The id of the project to work in. Defaults to the project of the credentials; environment variable ADATAGE_PROJECT."},
			"credentials": schema.StringAttribute{Optional: true, Sensitive: true,
				Description: "The credentials file of a service account (its JSON content, or the path of the file). Environment variable ADATAGE_CREDENTIALS."},
			"credentials_file": schema.StringAttribute{Optional: true, Description: "The path of the credentials file. Environment variable ADATAGE_CREDENTIALS_FILE."},
			"access_token":     schema.StringAttribute{Optional: true, Sensitive: true, Description: "An access token, instead of credentials. Environment variable ADATAGE_ACCESS_TOKEN."},
			"api_url":          schema.StringAttribute{Optional: true, Description: "The address of the API. Defaults to https://api.cloud.adatage.com; environment variable ADATAGE_API_URL."},
		},
	}
}

func pick(v types.String, env string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	return os.Getenv(env)
}

func (p *adatageProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var creds *client.Credentials
	text := pick(cfg.Credentials, "ADATAGE_CREDENTIALS")
	if text == "" {
		text = pick(cfg.CredentialsFile, "ADATAGE_CREDENTIALS_FILE")
	}
	if text != "" {
		var err error
		if creds, err = client.ParseCredentials(text); err != nil {
			resp.Diagnostics.AddError("Invalid credentials", err.Error())
			return
		}
	}
	c, err := client.New(pick(cfg.APIURL, "ADATAGE_API_URL"), pick(cfg.Project, "ADATAGE_PROJECT"), creds, pick(cfg.AccessToken, "ADATAGE_ACCESS_TOKEN"))
	if err != nil {
		resp.Diagnostics.AddError("Cannot configure the provider", err.Error()+". Set credentials (or ADATAGE_CREDENTIALS) and project (or ADATAGE_PROJECT).")
		return
	}
	c.Agent = "terraform-provider-adatage/" + p.version
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *adatageProvider) Resources(context.Context) []func() resource.Resource {
	var out []func() resource.Resource
	for _, spec := range allResources() {
		out = append(out, func() resource.Resource { return &genericResource{spec: spec} })
	}
	return out
}

func (p *adatageProvider) DataSources(context.Context) []func() datasource.DataSource {
	var out []func() datasource.DataSource
	for _, spec := range allDataSources() {
		out = append(out, func() datasource.DataSource { return &genericData{spec: spec} })
	}
	return out
}
