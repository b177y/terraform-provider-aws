// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lexv2models

import (
	"context"
	"errors"
	"time"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lexmodelsv2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/lexmodelsv2/types"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/enum"
	intflex "github.com/hashicorp/terraform-provider-aws/internal/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	"github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	lexschema "github.com/hashicorp/terraform-provider-aws/internal/service/lexv2models/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource(name="Slot")
func newResourceSlot(_ context.Context) (resource.ResourceWithConfigure, error) {
	r := &resourceSlot{}

	r.SetDefaultCreateTimeout(30 * time.Minute)
	r.SetDefaultUpdateTimeout(30 * time.Minute)
	r.SetDefaultDeleteTimeout(30 * time.Minute)

	return r, nil
}

const (
	ResNameSlot = "Slot"

	slotIDPartCount = 5
)

type resourceSlot struct {
	framework.ResourceWithConfigure
	framework.WithTimeouts
}

func (r *resourceSlot) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "aws_lexv2models_slot"
}

func (r *resourceSlot) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	multValueSettingsLNB := schema.ListNestedBlock{
		CustomType: fwtypes.NewListNestedObjectTypeOf[MultipleValuesSettingData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"allow_multiple_values": schema.BoolAttribute{
					Optional: true,
				},
			},
		},
	}

	obfuscationSettingLNB := schema.ListNestedBlock{
		CustomType: fwtypes.NewListNestedObjectTypeOf[ObfuscationSettingData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"obfuscation_setting_type": schema.StringAttribute{
					CustomType: fwtypes.StringEnumType[awstypes.ObfuscationSettingType](),
					Required:   true,
				},
			},
		},
	}

	defaultValueSpecificationLNB := schema.ListNestedBlock{
		CustomType: fwtypes.NewListNestedObjectTypeOf[DefaultValueSpecificationData](ctx),
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"default_value_list": schema.ListNestedBlock{
					CustomType: fwtypes.NewListNestedObjectTypeOf[DefaultValueData](ctx),
					Validators: []validator.List{
						listvalidator.IsRequired(),
					},
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"default_value": schema.StringAttribute{
								Required: true,
								Validators: []validator.String{
									stringvalidator.LengthBetween(1, 202),
								},
							},
						},
					},
				},
			},
		},
	}

	messageNBO := schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"custom_playload": schema.ListNestedBlock{
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				CustomType: fwtypes.NewListNestedObjectTypeOf[CustomPayloadData](ctx),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
			"image_response_card": schema.ListNestedBlock{
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				CustomType: fwtypes.NewListNestedObjectTypeOf[ImageResponseCardData](ctx),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"image_url": schema.StringAttribute{
							Optional: true,
						},
						"subtitle": schema.StringAttribute{
							Optional: true,
						},
						"title": schema.StringAttribute{
							Required: true,
						},
					},
					Blocks: map[string]schema.Block{
						"button": schema.ListNestedBlock{
							CustomType: fwtypes.NewListNestedObjectTypeOf[ButtonData](ctx),
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"text": schema.StringAttribute{
										Required: true,
									},
									"value": schema.StringAttribute{
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"plain_text_message": schema.ListNestedBlock{
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				CustomType: fwtypes.NewListNestedObjectTypeOf[PlainTextMessageData](ctx),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
			"ssml_message": schema.ListNestedBlock{
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				CustomType: fwtypes.NewListNestedObjectTypeOf[SSMLMessageData](ctx),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
		},
	}

	messageGroupLNB := schema.ListNestedBlock{
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
		},
		CustomType: fwtypes.NewListNestedObjectTypeOf[MessageGroupData](ctx),
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"message": schema.ListNestedBlock{
					Validators: []validator.List{
						listvalidator.SizeBetween(1, 1),
					},
					CustomType:   fwtypes.NewListNestedObjectTypeOf[MessageData](ctx),
					NestedObject: messageNBO,
				},
				"variation": schema.ListNestedBlock{
					CustomType:   fwtypes.NewListNestedObjectTypeOf[MessageData](ctx),
					NestedObject: messageNBO,
				},
			},
		},
	}

	allowedInputTypesLNB := schema.ListNestedBlock{
		Validators: []validator.List{
			listvalidator.SizeBetween(1, 1),
		},
		CustomType: fwtypes.NewListNestedObjectTypeOf[AllowedInputTypesData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"allow_audio_input": schema.BoolAttribute{
					Required: true,
				},
				"allow_dtmf_input": schema.BoolAttribute{
					Required: true,
				},
			},
		},
	}

	audioSpecificationLNB := schema.ListNestedBlock{
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		CustomType: fwtypes.NewListNestedObjectTypeOf[AudioSpecificationData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"end_timeout_ms": schema.Int64Attribute{
					Required: true,
					Validators: []validator.Int64{
						int64validator.AtLeast(1),
					},
				},
				"max_length_ms": schema.Int64Attribute{
					Required: true,
					Validators: []validator.Int64{
						int64validator.AtLeast(1),
					},
				},
			},
		},
	}

	dmfSpecificationLNB := schema.ListNestedBlock{
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		CustomType: fwtypes.NewListNestedObjectTypeOf[DTMFSpecificationData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"deletion_character": schema.StringAttribute{
					Required: true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexache.MustCompile(`^[A-D0-9#*]{1}$`),
							"alphanumeric characters",
						),
					},
				},
				"end_character": schema.StringAttribute{
					Required: true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexache.MustCompile(`^[A-D0-9#*]{1}$`),
							"alphanumeric characters",
						),
					},
				},
				"end_timeout_ms": schema.Int64Attribute{
					Required: true,
					Validators: []validator.Int64{
						int64validator.AtLeast(1),
					},
				},
				"max_length": schema.Int64Attribute{
					Required: true,
					Validators: []validator.Int64{
						int64validator.Between(1, 1024),
					},
				},
			},
		},
	}

	audioAndDTMFInputSpecificationLNB := schema.ListNestedBlock{
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		CustomType: fwtypes.NewListNestedObjectTypeOf[AudioAndDTMFInputSpecificationData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"start_timeout_ms": schema.Int64Attribute{
					Required: true,
					Validators: []validator.Int64{
						int64validator.AtLeast(1),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"audio_specification": audioSpecificationLNB,
				"dtmf_specification":  dmfSpecificationLNB,
			},
		},
	}

	textInputSpecificationLNB := schema.ListNestedBlock{
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		CustomType: fwtypes.NewListNestedObjectTypeOf[TextInputSpecificationData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"start_timeout_ms": schema.Int64Attribute{
					Required: true,
					Validators: []validator.Int64{
						int64validator.AtLeast(1),
					},
				},
			},
		},
	}

	promptAttemptsSpecificationLNB := schema.ListNestedBlock{
		CustomType: fwtypes.NewListNestedObjectTypeOf[PromptAttemptsSpecificationData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"map_block_key": schema.StringAttribute{
					Required:   true,
					CustomType: fwtypes.StringEnumType[PromptAttemptsType](),
				},
				"allow_interrupt": schema.BoolAttribute{
					Optional: true,
				},
			},
			Blocks: map[string]schema.Block{
				"allowed_input_types":                allowedInputTypesLNB,
				"audio_and_dtmf_input_specification": audioAndDTMFInputSpecificationLNB,
				"text_input_specification":           textInputSpecificationLNB,
			},
		},
	}

	promptSpecificationLNB := schema.ListNestedBlock{
		Validators: []validator.List{
			listvalidator.SizeBetween(1, 1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"allow_interrupt": schema.BoolAttribute{
					Optional: true,
				},
				"max_retries": schema.Int64Attribute{
					Required: true,
				},
				"message_selection_strategy": schema.StringAttribute{
					Optional: true,
					Validators: []validator.String{
						enum.FrameworkValidate[awstypes.MessageSelectionStrategy](),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"message_groups":                messageGroupLNB,
				"prompt_attempts_specification": promptAttemptsSpecificationLNB,
			},
		},
	}

	sampleUtteranceLNB := schema.ListNestedBlock{
		CustomType: fwtypes.NewListNestedObjectTypeOf[SampleUtteranceData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"utterance": schema.StringAttribute{
					Required: true,
				},
			},
		},
	}

	slotResolutionSettingLNB := schema.ListNestedBlock{
		CustomType: fwtypes.NewListNestedObjectTypeOf[SlotResolutionSettingData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"slot_resolution_strategy": schema.StringAttribute{
					CustomType: fwtypes.StringEnumType[awstypes.SlotResolutionStrategy](),
					Required:   true,
				},
			},
		},
	}

	responseSpecificationLNB := schema.ListNestedBlock{
		CustomType: fwtypes.NewListNestedObjectTypeOf[ResponseSpecificationData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"allow_interrupt": schema.BoolAttribute{
					Optional: true,
				},
			},
			Blocks: map[string]schema.Block{
				"message_groups": messageGroupLNB,
			},
		},
	}

	stillWaitingResponseSpecificationLNB := schema.ListNestedBlock{
		CustomType: fwtypes.NewListNestedObjectTypeOf[StillWaitingResponseSpecificationData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"allow_interrupt": schema.BoolAttribute{
					Optional: true,
				},
				"frequency_in_seconds": schema.Int64Attribute{
					Required: true,
				},
				"timeout_in_seconds": schema.Int64Attribute{
					Required: true,
				},
			},
			Blocks: map[string]schema.Block{
				"message_groups": messageGroupLNB,
			},
		},
	}

	waitAndContinueSpecificationLNB := schema.ListNestedBlock{
		CustomType: fwtypes.NewListNestedObjectTypeOf[WaitAndContinueSpecificationData](ctx),
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"active": schema.BoolAttribute{
					Optional: true,
				},
			},
			Blocks: map[string]schema.Block{
				"continue_response":      responseSpecificationLNB,
				"still_waiting_response": stillWaitingResponseSpecificationLNB,
				"waiting_response":       responseSpecificationLNB,
			},
		},
	}

	valueElicitationSettingLNB := schema.ListNestedBlock{
		CustomType: fwtypes.NewListNestedObjectTypeOf[ValueElicitationSettingData](ctx),
		Validators: []validator.List{
			listvalidator.IsRequired(),
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"slot_constraint": schema.StringAttribute{
					Required: true,
					Validators: []validator.String{
						enum.FrameworkValidate[awstypes.SlotConstraint](),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"default_value_specification":     defaultValueSpecificationLNB,
				"prompt_specification":            promptSpecificationLNB,
				"sample_utterance":                sampleUtteranceLNB,
				"slot_resolution_setting":         slotResolutionSettingLNB,
				"wait_and_continue_specification": waitAndContinueSpecificationLNB,
			},
		},
	}

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"bot_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bot_version": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"id": framework.IDAttribute(),
			"intent_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"locale_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"slot_type_id": schema.StringAttribute{
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"multiple_values_setting":   multValueSettingsLNB,
			"obfuscation_setting":       obfuscationSettingLNB,
			"value_elicitation_setting": valueElicitationSettingLNB,
			//sub_slot_setting
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *resourceSlot) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	conn := r.Meta().LexV2ModelsClient(ctx)

	var plan resourceSlotData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := &lexmodelsv2.CreateSlotInput{
		SlotName: aws.String(plan.Name.ValueString()),
	}

	resp.Diagnostics.Append(flex.Expand(ctx, plan, &in)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := conn.CreateSlot(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.LexV2Models, create.ErrActionCreating, ResNameSlot, plan.Name.String(), err),
			err.Error(),
		)
		return
	}
	if out == nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.LexV2Models, create.ErrActionCreating, ResNameSlot, plan.Name.String(), nil),
			errors.New("empty output").Error(),
		)
		return
	}

	idParts := []string{
		aws.ToString(out.BotId),
		aws.ToString(out.BotVersion),
		aws.ToString(out.IntentId),
		aws.ToString(out.LocaleId),
		aws.ToString(out.SlotId),
	}
	id, err := intflex.FlattenResourceId(idParts, slotIDPartCount, false)
	if err != nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.LexV2Models, create.ErrActionCreating, ResNameSlot, plan.Name.String(), err),
			err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(id)

	resp.Diagnostics.Append(flex.Flatten(ctx, out, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceSlot) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	conn := r.Meta().LexV2ModelsClient(ctx)

	var state resourceSlotData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := findSlotByID(ctx, conn, state.ID.ValueString())
	if tfresource.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.LexV2Models, create.ErrActionSetting, ResNameSlot, state.ID.String(), err),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(flex.Flatten(ctx, out, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceSlot) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	conn := r.Meta().LexV2ModelsClient(ctx)

	var plan, state resourceSlotData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if slotHasChanges(ctx, plan, state) {
		input := &lexmodelsv2.UpdateSlotInput{}

		// TODO: expand here, or check for updatable arguments individually?

		resp.Diagnostics.Append(flex.Expand(context.WithValue(ctx, flex.ResourcePrefix, ResNameSlot), &plan, input)...)
		if resp.Diagnostics.HasError() {
			return
		}

		out, err := conn.UpdateSlot(ctx, input)
		if err != nil {
			resp.Diagnostics.AddError(
				create.ProblemStandardMessage(names.LexV2Models, create.ErrActionUpdating, ResNameSlot, plan.ID.String(), err),
				err.Error(),
			)
			return
		}
		if out == nil {
			resp.Diagnostics.AddError(
				create.ProblemStandardMessage(names.LexV2Models, create.ErrActionUpdating, ResNameSlot, plan.ID.String(), nil),
				errors.New("empty output").Error(),
			)
			return
		}

		// resp.Diagnostics.Append(flex.Flatten(ctx, out, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceSlot) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	conn := r.Meta().LexV2ModelsClient(ctx)

	var state resourceSlotData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := &lexmodelsv2.DeleteSlotInput{
		BotId:      aws.String(state.ID.ValueString()),
		BotVersion: aws.String(state.ID.ValueString()),
		IntentId:   aws.String(state.ID.ValueString()),
		LocaleId:   aws.String(state.ID.ValueString()),
		SlotId:     aws.String(state.ID.ValueString()),
	}

	_, err := conn.DeleteSlot(ctx, in)
	if err != nil {
		var nfe *awstypes.ResourceNotFoundException
		if errors.As(err, &nfe) {
			return
		}
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.LexV2Models, create.ErrActionDeleting, ResNameSlot, state.ID.String(), err),
			err.Error(),
		)
		return
	}
}

func (r *resourceSlot) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func findSlotByID(ctx context.Context, conn *lexmodelsv2.Client, id string) (*lexmodelsv2.DescribeSlotOutput, error) {
	parts, err := intflex.ExpandResourceId(id, slotIDPartCount, false)
	if err != nil {
		return nil, err
	}

	in := &lexmodelsv2.DescribeSlotInput{
		BotId:      aws.String(parts[0]),
		BotVersion: aws.String(parts[1]),
		IntentId:   aws.String(parts[2]),
		LocaleId:   aws.String(parts[3]),
		SlotId:     aws.String(parts[4]),
	}

	out, err := conn.DescribeSlot(ctx, in)
	if err != nil {
		var nfe *awstypes.ResourceNotFoundException
		if errors.As(err, &nfe) {
			return nil, &retry.NotFoundError{
				LastError:   err,
				LastRequest: in,
			}
		}

		return nil, err
	}

	if out == nil {
		return nil, tfresource.NewEmptyResultError(in)
	}

	return out, nil
}

type resourceSlotData struct {
	BotID                    types.String                                                           `tfsdk:"bot_id"`
	BotVersion               types.String                                                           `tfsdk:"bot_version"`
	Description              types.String                                                           `tfsdk:"description"`
	ID                       types.String                                                           `tfsdk:"id"`
	IntentID                 types.String                                                           `tfsdk:"intent_id"`
	LocaleID                 types.String                                                           `tfsdk:"locale_id"`
	MultipleValuesSetting    fwtypes.ListNestedObjectValueOf[lexschema.MultipleValuesSettingData]   `tfsdk:"multiple_values_setting"`
	Name                     types.String                                                           `tfsdk:"name"`
	ObfuscationSetting       fwtypes.ListNestedObjectValueOf[lexschema.ObfuscationSettingData]      `tfsdk:"obfuscation_setting"`
	Timeouts                 timeouts.Value                                                         `tfsdk:"timeouts"`
	SlotTypeID               types.String                                                           `tfsdk:"slot_type_id"`
	ValueElicitationSettings fwtypes.ListNestedObjectValueOf[lexschema.ValueElicitationSettingData] `tfsdk:"value_elicitation_settings"`
}

type MultipleValuesSettingData struct {
	AllowMultipleValues types.Bool `tfsdk:"allow_multiple_values"`
}

type ObfuscationSettingData struct {
	ObfuscationSettingType fwtypes.StringEnum[awstypes.ObfuscationSettingType] `tfsdk:"obfuscation_setting_type"`
}

type DefaultValueSpecificationData struct {
	DefaultValueList fwtypes.ListNestedObjectValueOf[DefaultValueData] `tfsdk:"default_value_list"`
}

type DefaultValueData struct {
	DefaultValue types.String `tfsdk:"default_value"`
}

type PromptSpecificationData struct {
	AllowInterrupt              types.Bool                                               `tfsdk:"allow_interrupt"`
	MaxRetries                  types.Int64                                              `tfsdk:"max_retries"`
	MessageGroup                fwtypes.ListNestedObjectValueOf[MessageGroupData]        `tfsdk:"message_groups"`
	MessageSelectionStrategy    fwtypes.StringEnum[awstypes.MessageSelectionStrategy]    `tfsdk:"message_selection_strategy"`
	PromptAttemptsSpecification fwtypes.ObjectMapValueOf[PromptAttemptSpecificationData] `tfsdk:"prompt_attempts_specification"`
}
type PromptAttemptSpecificationData struct {
	AllowedInputTypes              fwtypes.ListNestedObjectValueOf[AllowedInputTypesData]              `tfsdk:"allowed_input_types"`
	AllowInterrupt                 types.Bool                                                          `tfsdk:"allow_interrupt"`
	AudioAndDTMFInputSpecification fwtypes.ListNestedObjectValueOf[AudioAndDTMFInputSpecificationData] `tfsdk:"audio_and_dtmf_input_specification"`
	TextInputSpecification         fwtypes.ListNestedObjectValueOf[TextInputSpecificationData]         `tfsdk:"text_input_specification"`
}

type DTMFSpecificationData struct {
	EndCharacter      types.String `tfsdk:"end_character"`
	EndTimeoutMs      types.Int64  `tfsdk:"end_timeout_ms"`
	DeletionCharacter types.String `tfsdk:"deletion_character"`
	MaxLength         types.Int64  `tfsdk:"max_length"`
}

type TextInputSpecificationData struct {
	StartTimeoutMs types.Int64 `tfsdk:"start_timeout_ms"`
}

type AllowedInputTypesData struct {
	AllowAudioInput types.Bool `tfsdk:"allow_audio_input"`
	AllowDTMFInput  types.Bool `tfsdk:"allow_dtmf_input"`
}

type AudioAndDTMFInputSpecificationData struct {
	AudioSpecification fwtypes.ListNestedObjectValueOf[AudioSpecificationData] `tfsdk:"audio_specification"`
	StartTimeoutMs     types.Int64                                             `tfsdk:"start_timeout_ms"`
	DTMFSpecification  fwtypes.ListNestedObjectValueOf[DTMFSpecificationData]  `tfsdk:"dtmf_specification"`
}

type AudioSpecificationData struct {
	EndTimeoutMs types.Int64 `tfsdk:"end_timeout_ms"`
	MaxLengthMs  types.Int64 `tfsdk:"max_length_ms"`
}

type CustomPayloadData struct {
	Value types.String `tfsdk:"value"`
}

type ImageResponseCardData struct {
	Title    types.String                                `tfsdk:"title"`
	Button   fwtypes.ListNestedObjectValueOf[ButtonData] `tfsdk:"buttons"`
	ImageURL types.String                                `tfsdk:"image_url"`
	Subtitle types.String                                `tfsdk:"subtitle"`
}

type ButtonData struct {
	Text  types.String `tfsdk:"text"`
	Value types.String `tfsdk:"value"`
}

type PlainTextMessageData struct {
	Value types.String `tfsdk:"value"`
}

type SSMLMessageData struct {
	Value types.String `tfsdk:"value"`
}
type MessageGroupData struct {
	Message    fwtypes.ListNestedObjectValueOf[MessageData] `tfsdk:"message"`
	Variations fwtypes.ListNestedObjectValueOf[MessageData] `tfsdk:"variations"`
}

type MessageData struct {
	CustomPayload     fwtypes.ListNestedObjectValueOf[CustomPayloadData]     `tfsdk:"custom_payload"`
	ImageResponseCard fwtypes.ListNestedObjectValueOf[ImageResponseCardData] `tfsdk:"image_response_card"`
	PlainTextMessage  fwtypes.ListNestedObjectValueOf[PlainTextMessageData]  `tfsdk:"plain_text_message"`
	SSMLMessage       fwtypes.ListNestedObjectValueOf[SSMLMessageData]       `tfsdk:"ssml_message"`
}

type PromptAttemptsSpecificationData struct {
	AllowedInputTypes              fwtypes.ListNestedObjectValueOf[AllowedInputTypes]              `tfsdk:"allowed_input_types"`
	AllowInterrupt                 types.Bool                                                      `tfsdk:"allow_interrupt"`
	AudioAndDTMFInputSpecification fwtypes.ListNestedObjectValueOf[AudioAndDTMFInputSpecification] `tfsdk:"audio_and_dtmf_input_specification"`
	MapBlockKey                    fwtypes.StringEnum[PromptAttemptsType]                          `tfsdk:"map_block_key"`
	TextInputSpecification         fwtypes.ListNestedObjectValueOf[TextInputSpecification]         `tfsdk:"text_input_specification"`
}

type SampleUtteranceData struct {
	Utterance types.String `tfsdk:"utterance"`
}

type SlotResolutionSettingData struct {
	SlotResolutionStrategy fwtypes.StringEnum[awstypes.SlotResolutionStrategy] `tfsdk:"slot_resolution_strategy"`
}

type ResponseSpecificationData struct {
	AllowInterrupt types.Bool                                        `tfsdk:"allow_interrupt"`
	MessageGroups  fwtypes.ListNestedObjectValueOf[MessageGroupData] `tfsdk:"message_groups"`
}

type StillWaitingResponseSpecificationData struct {
	AllowInterrupt     types.Bool                                        `tfsdk:"allow_interrupt"`
	FrequencyInSeconds types.Int64                                       `tfsdk:"frequency_in_seconds"`
	MessageGroups      fwtypes.ListNestedObjectValueOf[MessageGroupData] `tfsdk:"message_groups"`
	TimeoutInSeconds   types.Int64                                       `tfsdk:"timeout_in_seconds"`
}

type WaitAndContinueSpecificationData struct {
	Active               types.Bool                                                             `tfsdk:"active"`
	ContinueResponse     fwtypes.ListNestedObjectValueOf[ResponseSpecificationData]             `tfsdk:"continue_response"`
	StillWaitingResponse fwtypes.ListNestedObjectValueOf[StillWaitingResponseSpecificationData] `tfsdk:"still_waiting_response"`
	WaitingResponse      fwtypes.ListNestedObjectValueOf[ResponseSpecificationData]             `tfsdk:"waiting_response"`
}

type ValueElicitationSettingData struct {
	SlotConstraint               fwtypes.StringEnum[awstypes.SlotConstraint]                       `tfsdk:"slot_constraint"`
	DefaultValueSpecification    fwtypes.ListNestedObjectValueOf[DefaultValueSpecificationData]    `tfsdk:"default_value_specification"`
	PromptSpecification          fwtypes.ListNestedObjectValueOf[PromptSpecificationData]          `tfsdk:"prompt_specification"`
	SampleUtterance              fwtypes.ListNestedObjectValueOf[SampleUtteranceData]              `tfsdk:"sample_utterance"`
	SlotResolutionSetting        fwtypes.ListNestedObjectValueOf[SlotResolutionSettingData]        `tfsdk:"slot_resolution_setting"`
	WaitAndContinueSpecification fwtypes.ListNestedObjectValueOf[WaitAndContinueSpecificationData] `tfsdk:"wait_and_continue_specification"`
}

func slotHasChanges(_ context.Context, plan, state resourceSlotData) bool {
	return !plan.Description.Equal(state.Description) ||
		!plan.Name.Equal(state.Name) ||
		!plan.Description.Equal(state.Description) ||
		!plan.SlotTypeID.Equal(state.SlotTypeID) ||
		!plan.ObfuscationSetting.Equal(state.ObfuscationSetting)
}
