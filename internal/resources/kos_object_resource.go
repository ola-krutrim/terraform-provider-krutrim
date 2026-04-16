package resources

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
	"github.com/ola-krutrim/krutrim-go-sdk/option"
)

var _ resource.Resource = &KOSObjectResource{}
var _ resource.ResourceWithConfigure = &KOSObjectResource{}

type KOSObjectResource struct {
	client *krutrim.Client
}

func NewKOSObjectResource() resource.Resource {
	return &KOSObjectResource{}
}

type KOSObjectModel struct {
	ID types.String `tfsdk:"id"`

	BucketKRN types.String `tfsdk:"bucket_krn"`
	ObjectKey types.String `tfsdk:"object_key"`

	Region       types.String `tfsdk:"region"`
	SessionToken types.String `tfsdk:"session_token"`

	FilePath types.String `tfsdk:"file_path"`

	UploadURL   types.String `tfsdk:"upload_url"`
	DownloadURL types.String `tfsdk:"download_url"`
}

func (r *KOSObjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kos_object"
}

func (r *KOSObjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"id": schema.StringAttribute{Computed: true},

			"bucket_krn": schema.StringAttribute{Required: true},
			"object_key": schema.StringAttribute{Required: true},
			"region":     schema.StringAttribute{Required: true},

			"session_token": schema.StringAttribute{
				Required:  true,
			},

			"file_path": schema.StringAttribute{
				Required: true,
			},

			"upload_url":   schema.StringAttribute{Computed: true},
			"download_url": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *KOSObjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*krutrim.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data Type", "Expected *krutrim.Client")
		return
	}

	r.client = client
}

func (r *KOSObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	if r.client == nil {
		resp.Diagnostics.AddError("Client not initialized", "Provider client is nil")
		return
	}

	var plan KOSObjectModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Starting object upload", map[string]interface{}{
		"bucket": plan.BucketKRN.ValueString(),
		"key":    plan.ObjectKey.ValueString(),
	})

	// ✅ STEP 1: Wait + Retry InitUpload
	var res map[string]string

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {

		rres, err := r.client.Ko.V1.Objects.InitUpload(
			ctx,
			plan.BucketKRN.ValueString(),
			plan.ObjectKey.ValueString(),
			plan.Region.ValueString(),
			plan.SessionToken.ValueString(),
		)

		if err != nil {
			if strings.Contains(err.Error(), "Bucket is not in active state") {
				tflog.Debug(ctx, "Bucket not active yet, retrying...")
				return retry.RetryableError(err)
			}

			return retry.NonRetryableError(err)
		}

		res = rres
		return nil
	})

	if err != nil {
		resp.Diagnostics.AddError("Init upload failed after retry", err.Error())
		return
	}

	uploadURL := res["uploadPreSignedUrl"]
	if uploadURL == "" {
		resp.Diagnostics.AddError("Missing upload URL", "uploadPreSignedUrl not found")
		return
	}

	// ✅ STEP 2: Read file
	file, err := os.Open(plan.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("File open failed", err.Error())
		return
	}
	defer file.Close()

	// ✅ STEP 3: Upload file with retry (FIXED)

	err = retry.RetryContext(ctx, 60*time.Minute, func() *retry.RetryError {

		file, err := os.Open(plan.FilePath.ValueString())
		if err != nil {
			return retry.NonRetryableError(err)
		}
		defer file.Close()

		stat, _ := file.Stat()

		reqHTTP, err := http.NewRequest(http.MethodPut, uploadURL, file)
		if err != nil {
			return retry.NonRetryableError(err)
		}

		reqHTTP.ContentLength = stat.Size()
		reqHTTP.Header.Set("Content-Type", "application/octet-stream")
		reqHTTP.Header.Set("Expect", "100-continue")

		httpClient := &http.Client{
			Timeout: 0,
			Transport: &http.Transport{
				TLSHandshakeTimeout:   30 * time.Second,
				ExpectContinueTimeout: 10 * time.Second,
				IdleConnTimeout:       90 * time.Second,
				ResponseHeaderTimeout: 0,
			},
		}

		resHTTP, err := httpClient.Do(reqHTTP)
		if err != nil {
			if strings.Contains(err.Error(), "TLS handshake timeout") {
				tflog.Debug(ctx, "TLS timeout, retrying upload...")
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}
		defer resHTTP.Body.Close()

		if resHTTP.StatusCode >= 300 {
			return retry.RetryableError(
				fmt.Errorf("upload failed: %s", resHTTP.Status),
			)
		}

		return nil
	})

	if err != nil {
		resp.Diagnostics.AddError("Upload failed after retry", err.Error())
		return
	}

	// ✅ STEP 4: Get download URL
	downloadRes, err := r.client.Ko.V1.Objects.GetPreSignedDownloadURL(
		ctx,
		plan.BucketKRN.ValueString(),
		plan.ObjectKey.ValueString(),
		plan.Region.ValueString(),
		plan.SessionToken.ValueString(),
	)

	if err == nil {
		plan.DownloadURL = types.StringValue(downloadRes["url"])
	}

	plan.UploadURL = types.StringValue(uploadURL)
	plan.ID = types.StringValue(plan.ObjectKey.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *KOSObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state KOSObjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	_, err := r.client.Ko.V1.Objects.List(
		ctx,
		state.BucketKRN.ValueString(),
		state.Region.ValueString(),
		state.SessionToken.ValueString(),
	)

	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *KOSObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state KOSObjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	err := r.client.Ko.V1.Objects.Delete(
		ctx,
		state.BucketKRN.ValueString(),
		state.ObjectKey.ValueString(),
		option.WithHeader("x-region-id", state.Region.ValueString()),
		option.WithHeader("x-session-token", state.SessionToken.ValueString()),
	)

	if err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
	}
}

func (r *KOSObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Object must be recreated")
}