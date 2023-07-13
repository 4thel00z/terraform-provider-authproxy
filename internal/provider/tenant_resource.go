// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io"
	"net/http"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TenantResource{}
var _ resource.ResourceWithImportState = &TenantResource{}

func NewTenantResource() resource.Resource {
	return &TenantResource{}
}

// TenantResource defines the resource implementation.
type TenantResource struct {
	providerData *ProviderData
	name         string
}

// TenantResourceModel describes the resource data model.
type TenantResourceModel struct {
	Name types.String `tfsdk:"name"`
	ID   types.String `tfsdk:"id"`
}

func (r *TenantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

func (r *TenantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Tenant resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the tenant",
				Optional:            false,
				Required:            true,
			},
			// "defaulted": schema.StringAttribute{
			// 	MarkdownDescription: "Example configurable attribute with default value",
			// 	Optional:            true,
			// 	Computed:            true,
			// 	Default:             stringdefault.StaticString("example value when not configured"),
			// },
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The database uuid",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TenantResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*ProviderData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.providerData = data
}

type createRequest struct {
	Name string `json:"tenant"`
}

type createResponse struct {
	ID string `json:"id"`
}

type updateRequest struct {
	Name    string `json:"tenant"`
	NewName string `json:"new_tenant"`
}

type updateResponse struct {
	ID string `json:"id"`
}

type readResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type deleteResponse struct {
	ID string `json:"id"`
}

func (r *TenantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *TenantResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider providerData data and make a call using it.
	// httpResp, err := r.providerData.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
	//     return
	// }

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	marshalled, err := json.Marshal(createRequest{Name: data.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tenant, got error: %s", err))
		return
	}
	request, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/tenants", r.providerData.endpoint), bytes.NewReader(marshalled))

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tenant, tenantdata: %#v got error: %s", r.providerData, err))
		return
	}
	tflog.Debug(ctx, "Setting basic auth")
	request.SetBasicAuth(r.providerData.username, r.providerData.password)
	tflog.Debug(ctx, "Making request")

	res, err := r.providerData.client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tenant, got error: %s", err))
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to handle non 200 status code on tenant creation, got error: %s", err))
			return
		}
		tflog.Error(ctx, "could not create tenant", map[string]interface{}{
			"body": string(resBody),
		})
		return
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tenant, got error: %s", err))
		return
	}
	var cr createResponse
	err = json.Unmarshal(resBody, &cr)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tenant, got error: %s", err))
		return
	}

	data.ID = types.StringValue(cr.ID)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Debug(ctx, "created a tenant resource, response was", map[string]interface{}{
		"id": cr.ID,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TenantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *TenantResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	request, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/tenants/%s", r.providerData.endpoint, data.Name.ValueString()), nil)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tenant, got error: %s", err))
		return
	}
	tflog.Debug(ctx, "Setting basic auth")
	request.SetBasicAuth(r.providerData.username, r.providerData.password)
	tflog.Debug(ctx, "Making request")

	res, err := r.providerData.client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tenant, got error: %s", err))
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to handle non 200 status code on tenant read, got error: %s", err))
			return
		}
		tflog.Error(ctx, "could not read tenant", map[string]interface{}{
			"body": string(resBody),
		})
		return
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tenant, got error: %s", err))
		return
	}
	var newTenant readResponse
	err = json.Unmarshal(resBody, &newTenant)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tenant, got error: %s", err))
		return
	}
	data.ID = types.StringValue(newTenant.ID)

	// If applicable, this is a great opportunity to initialize any necessary
	// provider providerData data and make a call using it.
	// httpResp, err := r.providerData.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TenantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *TenantResourceModel
	var old *TenantResourceModel

	// Read Terraform old data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider providerData data and make a call using it.
	// httpResp, err := r.providerData.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	marshalled, err := json.Marshal(updateRequest{Name: old.Name.ValueString(), NewName: data.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update tenant, got error: %s", err))
		return
	}
	request, err := http.NewRequestWithContext(ctx, "PATCH", fmt.Sprintf("%s/tenants", r.providerData.endpoint), bytes.NewReader(marshalled))

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update tenant, got error: %s", err))
		return
	}
	tflog.Debug(ctx, "Setting basic auth")
	request.SetBasicAuth(r.providerData.username, r.providerData.password)
	tflog.Debug(ctx, "Making request")

	res, err := r.providerData.client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update tenant, got error: %s", err))
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to handle non 200 status code on tenant update, got error: %s", err))
			return
		}
		tflog.Error(ctx, "could not update tenant", map[string]interface{}{
			"body": string(resBody),
		})
		return
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update tenant, got error: %s", err))
		return
	}
	var cr updateResponse
	err = json.Unmarshal(resBody, &cr)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update tenant, got error: %s", err))
		return
	}
	data.ID = types.StringValue(cr.ID)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "updated a tenant resource")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TenantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *TenantResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider providerData data and make a call using it.
	// httpResp, err := r.providerData.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }

	request, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/tenants/%s", r.providerData.endpoint, r.name), nil)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tenant, got error: %s", err))
		return
	}
	tflog.Debug(ctx, "Setting basic auth")
	request.SetBasicAuth(r.providerData.username, r.providerData.password)
	tflog.Debug(ctx, "Making request")

	res, err := r.providerData.client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete tenant, got error: %s", err))
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to handle non 200 status code on tenant deletion, got error: %s", err))
			return
		}
		tflog.Error(ctx, "could not delete tenant", map[string]interface{}{
			"body": string(resBody),
		})
		return
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete tenant, got error: %s", err))
		return
	}
	var newTenant readResponse
	err = json.Unmarshal(resBody, &newTenant)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete tenant, got error: %s", err))
		return
	}
	data.ID = types.StringValue(newTenant.ID)
	data.Name = types.StringValue(newTenant.Name)

}

func (r *TenantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
