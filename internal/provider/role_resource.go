package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io"
	"net/http"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RoleResource{}
var _ resource.ResourceWithImportState = &RoleResource{}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

// RoleResource defines the resource implementation.
type RoleResource struct {
	providerData *ProviderData
}

// RoleResourceModel describes the resource data model.
type RoleResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Tenant types.String `tfsdk:"tenant"`
	Scopes types.List   `tfsdk:"scopes"`
}

func (r *RoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

func (r *RoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Tenant resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the role",
				Optional:            false,
				Required:            true,
			},
			"tenant": schema.StringAttribute{
				MarkdownDescription: "Tenant in which to create the role",
				Optional:            false,
				Required:            true,
			},
			"scopes": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				MarkdownDescription: "The scopes of the role",
				// default is an empty list
				Default: listdefault.StaticValue(types.ListValueMust(
					types.StringType,
					[]attr.Value{},
				)),
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

func (r *RoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type createRoleRequest struct {
	Name   string   `json:"name"`
	Tenant string   `json:"tenant"`
	Scopes []string `json:"scopes"`
}

type createRoleResponse struct {
	ID string `json:"id"`
}

type updateRoleRequest struct {
	Name      string `json:"name"`
	Tenant    string `json:"tenant"`
	NewName   string `json:"new_name"`
	NewScopes string `json:"new_scopes"`
}

type updateRoleResponse struct {
	ID string `json:"id"`
}

type readRoleResponse struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Tenant string   `json:"tenant"`
	Scopes []string `json:"scopes"`
}

type deleteRoleResponse struct {
	ID string `json:"id"`
}

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *RoleResourceModel
	var scopes []string
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.Scopes.ElementsAs(ctx, &scopes, false)...)

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
	marshalled, err := json.Marshal(createRoleRequest{
		Name:   data.Name.ValueString(),
		Tenant: data.Tenant.ValueString(),
		Scopes: scopes,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role, got error: %s", err.Error()))
		return
	}
	request, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/roles", r.providerData.endpoint), bytes.NewReader(marshalled))

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role, got error: %s", err.Error()))
		return
	}
	tflog.Debug(ctx, "Setting basic auth")
	request.SetBasicAuth(r.providerData.username, r.providerData.password)
	tflog.Debug(ctx, "Making request")

	res, err := r.providerData.client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role, got error: %s", err))
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to handle non 200 status code on role creation, got error: %s", err))
			return
		}
		tflog.Error(ctx, "could not create role", map[string]interface{}{
			"body": string(resBody),
		})
		return
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role, got error: %s", err.Error()))
		return
	}
	var cr createRoleResponse
	err = json.Unmarshal(resBody, &cr)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role, got error: %s", err.Error()))
		return
	}

	data.ID = types.StringValue(cr.ID)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Debug(ctx, "created a role resource, response was", map[string]interface{}{
		"id": cr.ID,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *RoleResourceModel
	var scopes []string
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.Scopes.ElementsAs(ctx, &scopes, false)...)

	if resp.Diagnostics.HasError() {
		return
	}

	//	/tenants/{tenant}/roles/{name}
	request, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/tenants/%s/roles/%s", r.providerData.endpoint, data.Tenant.ValueString(), data.Name.ValueString()), nil)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tenant, got error: %s", err))
		return
	}
	tflog.Debug(ctx, "Setting basic auth")
	request.SetBasicAuth(r.providerData.username, r.providerData.password)
	tflog.Debug(ctx, "Making request")

	res, err := r.providerData.client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role, got error: %s", err))
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to handle non 200 status code on role read, got error: %s", err))
			return
		}
		tflog.Error(ctx, "could not read role", map[string]interface{}{
			"body": string(resBody),
		})
		return
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role, got error: %s", err))
		return
	}
	var newRole readRoleResponse
	err = json.Unmarshal(resBody, &newRole)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tenant, got error: %s", err))
		return
	}
	data.ID = types.StringValue(newRole.ID)
	listValue, diagnostics := types.ListValueFrom(ctx, types.StringType, scopes)
	resp.Diagnostics.Append(diagnostics...)
	data.Scopes = listValue
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

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *RoleResourceModel
	var old *RoleResourceModel

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

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *RoleResourceModel

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

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
