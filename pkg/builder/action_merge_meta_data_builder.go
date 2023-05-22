package builder

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/turbonomic/turbo-go-sdk/pkg/proto"
)

// Create action execution target when it is directly connected to the entity.
type ActionAggregationTargetBuilder struct {
	relatedEntityType proto.EntityDTO_EntityType
	relatedBy         proto.ConnectedEntity_ConnectionType
}

func NewActionAggregationTargetBuilder(relatedEntityType proto.EntityDTO_EntityType,
	relatedBy proto.ConnectedEntity_ConnectionType) *ActionAggregationTargetBuilder {
	return &ActionAggregationTargetBuilder{
		relatedEntityType: relatedEntityType,
		relatedBy:         relatedBy,
	}
}

func (builder *ActionAggregationTargetBuilder) Create() *proto.ActionMergeTargetData {
	target := &proto.ActionMergeTargetData{
		RelatedTo: &builder.relatedEntityType,
		RelatedBy: &proto.ActionMergeTargetData_EntityRelationship{
			EntityRelationship: &proto.ActionMergeTargetData_EntityRelationship_ConnectionType{
				ConnectionType: builder.relatedBy,
			},
		},
	}
	return target
}

// Create action execution target when the execution target entity is connected via a deduplication entity.
type ActionDeDuplicateAndAggregationTargetBuilder struct {
	deDuplicationTarget *ActionAggregationTargetBuilder
	aggregationTarget   *ActionAggregationTargetBuilder
}

func NewActionDeDuplicateAndAggregationTargetBuilder() *ActionDeDuplicateAndAggregationTargetBuilder {
	return &ActionDeDuplicateAndAggregationTargetBuilder{}
}

func (builder *ActionDeDuplicateAndAggregationTargetBuilder) DeDuplicatedBy(
	deDuplicationTarget *ActionAggregationTargetBuilder) *ActionDeDuplicateAndAggregationTargetBuilder {
	builder.deDuplicationTarget = deDuplicationTarget
	return builder
}

func (builder *ActionDeDuplicateAndAggregationTargetBuilder) AggregatedBy(
	aggregationTarget *ActionAggregationTargetBuilder) *ActionDeDuplicateAndAggregationTargetBuilder {
	builder.aggregationTarget = aggregationTarget
	return builder
}
func (builder *ActionDeDuplicateAndAggregationTargetBuilder) Create() *proto.ChainedActionMergeTargetData {
	chainedMergeTarget := &proto.ChainedActionMergeTargetData{}
	if builder.deDuplicationTarget == nil || builder.aggregationTarget == nil {
		return chainedMergeTarget
	}

	true_flag := true
	deDuplicationTargetLink := &proto.ChainedActionMergeTargetData_TargetDataLink{
		MergeTarget: builder.deDuplicationTarget.Create(),
		DeDuplicate: &true_flag,
	}
	chainedMergeTarget.TargetLinks = append(chainedMergeTarget.TargetLinks, deDuplicationTargetLink)

	false_flag := false
	aggregationTargetLink := &proto.ChainedActionMergeTargetData_TargetDataLink{
		MergeTarget: builder.aggregationTarget.Create(),
		DeDuplicate: &false_flag,
	}
	chainedMergeTarget.TargetLinks = append(chainedMergeTarget.TargetLinks, aggregationTargetLink)

	return chainedMergeTarget
}

// Resize Merge Policy DTO builder
type ResizeMergePolicyBuilder struct {
	entityType                *proto.EntityDTO_EntityType
	aggregationTargets        []*ActionAggregationTargetBuilder
	chainedAggregationTargets []*ActionDeDuplicateAndAggregationTargetBuilder
	commTypes                 []*CommodityMergeData
}

type CommodityMergeData struct {
	commType    proto.CommodityDTO_CommodityType
	changedAttr proto.ActionItemDTO_CommodityAttribute
}

func NewResizeMergeSpecBuilder() *ResizeMergePolicyBuilder {
	return &ResizeMergePolicyBuilder{}
}

func (rb *ResizeMergePolicyBuilder) ForEntityType(entityType proto.EntityDTO_EntityType) *ResizeMergePolicyBuilder {
	rb.entityType = &entityType
	return rb
}

func (rb *ResizeMergePolicyBuilder) AggregateBy(mergeTarget *ActionAggregationTargetBuilder) *ResizeMergePolicyBuilder {
	rb.aggregationTargets = append(rb.aggregationTargets, mergeTarget)
	return rb
}

func (rb *ResizeMergePolicyBuilder) DeDuplicateAndAggregateBy(mergeTarget *ActionDeDuplicateAndAggregationTargetBuilder) *ResizeMergePolicyBuilder {
	rb.chainedAggregationTargets = append(rb.chainedAggregationTargets, mergeTarget)
	return rb
}

func (rb *ResizeMergePolicyBuilder) ForCommodity(commType proto.CommodityDTO_CommodityType) *ResizeMergePolicyBuilder {
	comm := &CommodityMergeData{
		commType: commType,
	}
	rb.commTypes = append(rb.commTypes, comm)
	return rb
}

func (rb *ResizeMergePolicyBuilder) ForCommodityAndAttribute(commType proto.CommodityDTO_CommodityType,
	changedAttr proto.ActionItemDTO_CommodityAttribute) *ResizeMergePolicyBuilder {
	comm := &CommodityMergeData{
		commType:    commType,
		changedAttr: changedAttr,
	}
	rb.commTypes = append(rb.commTypes, comm)
	return rb
}

// Create the ActionMergePolicyDTO for merging resize actions.
func (rb *ResizeMergePolicyBuilder) Build() (*proto.ActionMergePolicyDTO, error) {
	if rb.entityType == nil {
		return nil, fmt.Errorf("Entity type required for action merge policy")
	}

	if len(rb.aggregationTargets) == 0 && len(rb.chainedAggregationTargets) == 0 {
		return nil, fmt.Errorf("Target type required for action merge policy")
	}

	if len(rb.commTypes) == 0 {
		return nil, fmt.Errorf("Commodity types required for resize merge policy")
	}

	commMergeDataList := []*proto.ResizeMergeSpec_CommodityMergeData{}
	for _, commData := range rb.commTypes {
		commMergeData := &proto.ResizeMergeSpec_CommodityMergeData{
			CommodityType: &commData.commType,
			ChangedAttr:   &commData.changedAttr,
		}
		commMergeDataList = append(commMergeDataList, commMergeData)
	}
	resizeSpec := &proto.ResizeMergeSpec{
		CommodityData: commMergeDataList,
	}

	mergeSpec := &proto.ActionMergePolicyDTO{
		EntityType: rb.entityType,

		ActionSpec: &proto.ActionMergePolicyDTO_ResizeSpec{
			ResizeSpec: resizeSpec,
		},
	}

	var executionTargetList []*proto.ActionMergeExecutionTarget
	for _, targetData := range rb.aggregationTargets {
		executionTarget := &proto.ActionMergeExecutionTarget{
			ExecutionTarget: &proto.ActionMergeExecutionTarget_MergeTarget{
				MergeTarget: targetData.Create(),
			},
		}
		executionTargetList = append(executionTargetList, executionTarget)
	}

	for _, targetData := range rb.chainedAggregationTargets {
		chainedTarget := targetData.Create()
		if len(chainedTarget.TargetLinks) == 0 {
			glog.Errorf("Invalid chained merge target")
			continue
		}
		executionTarget := &proto.ActionMergeExecutionTarget{
			ExecutionTarget: &proto.ActionMergeExecutionTarget_ChainedMergeTarget{
				ChainedMergeTarget: chainedTarget,
			},
		}
		executionTargetList = append(executionTargetList, executionTarget)
	}

	mergeSpec.ExecutionTargets = executionTargetList
	return mergeSpec, nil
}

// Horizontal Scale Merge Policy DTO builder
type HorizontalScaleMergePolicyBuilder struct {
	entityType                *proto.EntityDTO_EntityType
	chainedAggregationTargets []*ActionDeDuplicateAndAggregationTargetBuilder
	commTypes                 []*CommodityMergeData
	entityFilters             []*proto.EntityDTO_ContainerPodData
}

func NewHorizontalScaleMergeSpecBuilder() *HorizontalScaleMergePolicyBuilder {
	return &HorizontalScaleMergePolicyBuilder{}
}

func (hsb *HorizontalScaleMergePolicyBuilder) ForEntityType(entityType proto.EntityDTO_EntityType) *HorizontalScaleMergePolicyBuilder {
	hsb.entityType = &entityType
	return hsb
}

func (hsb *HorizontalScaleMergePolicyBuilder) DeDuplicateAndAggregateBy(mergeTarget *ActionDeDuplicateAndAggregationTargetBuilder) *HorizontalScaleMergePolicyBuilder {
	hsb.chainedAggregationTargets = append(hsb.chainedAggregationTargets, mergeTarget)
	return hsb
}

func (hsb *HorizontalScaleMergePolicyBuilder) ForContainerPodDataFilter(podData proto.EntityDTO_ContainerPodData) *HorizontalScaleMergePolicyBuilder {
	hsb.entityFilters = append(hsb.entityFilters, &podData)
	return hsb
}

func (hsb *HorizontalScaleMergePolicyBuilder) ForCommodity(commType proto.CommodityDTO_CommodityType) *HorizontalScaleMergePolicyBuilder {
	comm := &CommodityMergeData{
		commType: commType,
	}
	hsb.commTypes = append(hsb.commTypes, comm)
	return hsb
}

func (hsb *HorizontalScaleMergePolicyBuilder) ForCommodityAndAttribute(commType proto.CommodityDTO_CommodityType,
	changedAttr proto.ActionItemDTO_CommodityAttribute) *HorizontalScaleMergePolicyBuilder {
	comm := &CommodityMergeData{
		commType:    commType,
		changedAttr: changedAttr,
	}
	hsb.commTypes = append(hsb.commTypes, comm)
	return hsb
}

// Create the ActionMergePolicyDTO for Horizontal Scale actions.
func (hsb *HorizontalScaleMergePolicyBuilder) Build() (*proto.ActionMergePolicyDTO, error) {
	if hsb.entityType == nil {
		return nil, fmt.Errorf("Entity type required for horizontal scale merge policy")
	}

	if len(hsb.chainedAggregationTargets) == 0 {
		return nil, fmt.Errorf("Target type required for horizontal scale merge merge policy")
	}

	if len(hsb.commTypes) == 0 {
		return nil, fmt.Errorf("Commodity types required for horizontal scale merge merge policy")
	}

	if len(hsb.entityFilters) == 0 {
		return nil, fmt.Errorf("Entity flters required for horizontal scale merge merge policy")
	}

	commMergeDataList := []*proto.HorizontalScaleMergeSpec_CommodityMergeData{}
	for _, commData := range hsb.commTypes {
		commMergeData := &proto.HorizontalScaleMergeSpec_CommodityMergeData{
			CommodityType: &commData.commType,
			ChangedAttr:   &commData.changedAttr,
		}
		commMergeDataList = append(commMergeDataList, commMergeData)
	}
	horizontalScaleSpec := &proto.HorizontalScaleMergeSpec{
		CommodityData: commMergeDataList,
	}

	mergeSpec := &proto.ActionMergePolicyDTO{
		EntityType: hsb.entityType,

		ActionSpec: &proto.ActionMergePolicyDTO_HorizontalScaleSpec{
			HorizontalScaleSpec: horizontalScaleSpec,
		},
	}

	var executionTargetList []*proto.ActionMergeExecutionTarget

	for _, targetData := range hsb.chainedAggregationTargets {
		chainedTarget := targetData.Create()
		if len(chainedTarget.TargetLinks) == 0 {
			glog.Errorf("Invalid chained merge target")
			continue
		}
		executionTarget := &proto.ActionMergeExecutionTarget{
			ExecutionTarget: &proto.ActionMergeExecutionTarget_ChainedMergeTarget{
				ChainedMergeTarget: chainedTarget,
			},
		}
		executionTargetList = append(executionTargetList, executionTarget)
	}
	mergeSpec.ExecutionTargets = executionTargetList

	var filterList []*proto.ActionMergePolicyDTO_EntityFilter
	for _, filter := range hsb.entityFilters {
		entityFilter := &proto.ActionMergePolicyDTO_EntityFilter{
			EntityFilterProps: &proto.ActionMergePolicyDTO_EntityFilter_ContainerPodData{
				ContainerPodData: filter,
			},
		}
		filterList = append(filterList, entityFilter)
	}
	mergeSpec.EntityFilters = filterList

	return mergeSpec, nil
}
