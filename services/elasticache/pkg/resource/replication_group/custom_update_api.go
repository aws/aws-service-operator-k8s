// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package replication_group

import (
	"context"
	"fmt"
	ackv1alpha1 "github.com/aws/aws-controllers-k8s/apis/core/v1alpha1"
	"github.com/aws/aws-controllers-k8s/pkg/requeue"
	"github.com/pkg/errors"
	"sort"

	ackcompare "github.com/aws/aws-controllers-k8s/pkg/compare"
	svcapitypes "github.com/aws/aws-controllers-k8s/services/elasticache/apis/v1alpha1"
	svcsdk "github.com/aws/aws-sdk-go/service/elasticache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Implements specialized logic for replication group updates.
func (rm *resourceManager) CustomModifyReplicationGroup(
	ctx context.Context,
	desired *resource,
	latest *resource,
	diffReporter *ackcompare.Reporter,
) (*resource, error) {

	latestRGStatus := latest.ko.Status.Status

	allNodeGroupsAvailable := true
	nodeGroupMembersCount := 0
	if latest.ko.Status.NodeGroups != nil {
		for _, nodeGroup := range latest.ko.Status.NodeGroups {
			if nodeGroup.Status == nil || *nodeGroup.Status != "available" {
				allNodeGroupsAvailable = false
				break
			}
		}
		for _, nodeGroup := range latest.ko.Status.NodeGroups {
			if nodeGroup.NodeGroupMembers == nil {
				continue
			}
			nodeGroupMembersCount = nodeGroupMembersCount + len(nodeGroup.NodeGroupMembers)
		}
	}

	if latestRGStatus == nil || *latestRGStatus != "available" || !allNodeGroupsAvailable {
		return nil, requeue.NeededAfter(
			errors.New("Replication Group can not be modified, it is not in 'available' state."),
			requeue.DefaultRequeueAfterDuration)
	}

	memberClustersCount := 0
	if latest.ko.Status.MemberClusters != nil {
		memberClustersCount = len(latest.ko.Status.MemberClusters)
	}
	if memberClustersCount != nodeGroupMembersCount {
		return nil, requeue.NeededAfter(
			errors.New("Replication Group can not be modified, "+
				"need to wait for member clusters and node group members."),
			requeue.DefaultRequeueAfterDuration)
	}

	// Order of operations when diffs map to multiple updates APIs:
	// 1. When automaticFailoverEnabled differs:
	//		if automaticFailoverEnabled == false; do nothing in this custom logic, let the modify execute first.
	// 		else if automaticFailoverEnabled == true then following logic should execute first.
	// 2. When multiAZ differs
	// 		if multiAZ = true  then below is fine.
	// 		else if multiAZ = false ; do nothing in custom logic, let the modify execute.
	// 3. updateReplicaCount() is invoked Before updateShardConfiguration()
	//		because both accept availability zones, however the number of
	//		values depend on replica count.
	if desired.ko.Spec.AutomaticFailoverEnabled != nil && *desired.ko.Spec.AutomaticFailoverEnabled == false {
		latestAutomaticFailoverEnabled := latest.ko.Status.AutomaticFailover != nil && *latest.ko.Status.AutomaticFailover == "enabled"
		if latestAutomaticFailoverEnabled != *desired.ko.Spec.AutomaticFailoverEnabled {
			return rm.modifyReplicationGroup(ctx, desired, latest)
		}
	}
	if desired.ko.Spec.MultiAZEnabled != nil && *desired.ko.Spec.MultiAZEnabled == false {
		latestMultiAZEnabled := latest.ko.Status.MultiAZ != nil && *latest.ko.Status.MultiAZ == "enabled"
		if latestMultiAZEnabled != *desired.ko.Spec.MultiAZEnabled {
			return rm.modifyReplicationGroup(ctx, desired, latest)
		}
	}

	// increase/decrease replica count
	if diff := rm.replicaCountDifference(desired, latest); diff != 0 {
		if diff > 0 {
			return rm.increaseReplicaCount(ctx, desired, latest)
		}
		return rm.decreaseReplicaCount(ctx, desired, latest)
	}

	// increase/decrease shards
	if rm.shardConfigurationsDiffer(desired, latest) {
		return rm.updateShardConfiguration(ctx, desired, latest)
	}

	if rm.pendingStopServiceUpdates(desired, latest) {
		rm.stopServiceUpdates(ctx, desired, latest)
	}
	if rm.pendingApplyServiceUpdates(desired, latest) {
		rm.applyServiceUpdates(ctx, desired, latest)
	}

	return rm.modifyReplicationGroup(ctx, desired, latest)
}

// pendingApplyServiceUpdates returns true if service updates from Spec.ServiceUpdateActions
// are pending (i.e "not-applied", "stopped") as per latest Status.UpdateActions details.
func (rm *resourceManager) pendingApplyServiceUpdates(
	desired *resource,
	latest *resource,
) bool {
	// TODO: implement the logic per method description
	return false
}

// applyServiceUpdates applies the service updates.
func (rm *resourceManager) applyServiceUpdates(
	ctx context.Context,
	desired *resource,
	latest *resource,
) error {
	// Select Spec.UpdateActions that are in "not-applied", "stopped" state
	// and are present in Spec.ServiceUpdateActions
	// TODO: apply identified service updates for this replication group
	// input := &elasticache.BatchApplyUpdateActionInput{}
	// input.SetReplicationGroupIds - replication group id
	// input.SetServiceUpdateName - service update name
	// resp, err := rm.sdkapi.BatchApplyUpdateActionWithContext(ctx, input)
	return nil
}

// pendingStopServiceUpdates returns true if there exist Spec.UpdateActions that are
// being applied (i.e. "waiting-to-start", "in-progress", "scheduling", "scheduled")
// but are not present (i.e. have been removed) in Spec.ServiceUpdateActions
func (rm *resourceManager) pendingStopServiceUpdates(
	desired *resource,
	latest *resource,
) bool {
	// TODO: implement the logic per method description
	return false
}

// stopServiceUpdates stops the service updates for this replication group
func (rm *resourceManager) stopServiceUpdates(
	ctx context.Context,
	desired *resource,
	latest *resource,
) error {
	// Select Spec.UpdateActions that are in "waiting-to-start", "in-progress", "scheduling", "scheduled" state
	// but are not present (i.e. have been removed) in Spec.ServiceUpdateActions
	// TODO: stop identified service updates for this replication group
	// input := &elasticache.BatchStopUpdateActionInput{}
	// input.SetReplicationGroupIds - replication group id
	// input.SetServiceUpdateName - service update name
	// resp, err := rm.sdkapi.BatchStopUpdateActionWithContext(ctx, input)
	return nil
}

// modifyReplicationGroup updates replication group
// it handles properties that put replication group in
// modifying state if these are supplied to modify API
// irrespective of apply immediately.
func (rm *resourceManager) modifyReplicationGroup(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (*resource, error) {
	// Method currently handles SecurityGroupIDs, EngineVersion
	// Avoid making unnecessary DescribeCacheCluster API call if both fields are nil in spec.
	if desired.ko.Spec.SecurityGroupIDs == nil && desired.ko.Spec.EngineVersion == nil {
		// no updates done
		return nil, nil
	}

	// Get details using describe cache cluster to compute diff
	latestCacheCluster, err := rm.describeCacheCluster(ctx, latest)
	if err != nil {
		return nil, err
	}

	// SecurityGroupIds, EngineVersion
	if rm.securityGroupIdsDiffer(desired, latest, latestCacheCluster) ||
		rm.engineVersionDiffer(desired, latest, latestCacheCluster) {
		input := rm.newModifyReplicationGroupRequestPayload(desired, latest, latestCacheCluster)
		resp, respErr := rm.sdkapi.ModifyReplicationGroupWithContext(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "ModifyReplicationGroup", respErr)
		if respErr != nil {
			rm.log.V(1).Info("Error during ModifyReplicationGroup", "error", respErr)
			return nil, respErr
		}

		return rm.provideUpdatedResource(desired, resp.ReplicationGroup)
	}

	// no updates done
	return nil, nil
}

// replicaConfigurationsDifference returns
// positive number if desired replica count is greater than latest replica count
// negative number if desired replica count is less than latest replica count
// 0 otherwise
func (rm *resourceManager) replicaCountDifference(
	desired *resource,
	latest *resource,
) int {
	desiredSpec := desired.ko.Spec

	// There are two ways of setting replica counts for NodeGroups in Elasticache ReplicationGroup.
	// - The first way is to have the same replica count for all node groups.
	//   In this case, the Spec.ReplicasPerNodeGroup field is set to a non-nil-value integer pointer.
	// - The second way is to set different replica counts per node group.
	//   In this case, the Spec.NodeGroupConfiguration field is set to a non-nil NodeGroupConfiguration slice
	//   of NodeGroupConfiguration structs that each have a ReplicaCount non-nil-value integer pointer field
	//   that contains the number of replicas for that particular node group.
	if desiredSpec.ReplicasPerNodeGroup != nil {
		return rm.diffReplicasPerNodeGroup(desired, latest)
	} else if desiredSpec.NodeGroupConfiguration != nil {
		return rm.diffReplicasNodeGroupConfiguration(desired, latest)
	}
	return 0
}

// diffReplicasPerNodeGroup takes desired Spec.ReplicasPerNodeGroup field into account to return
// positive number if desired replica count is greater than latest replica count
// negative number if desired replica count is less than latest replica count
// 0 otherwise
func (rm *resourceManager) diffReplicasPerNodeGroup(
	desired *resource,
	latest *resource,
) int {
	desiredSpec := desired.ko.Spec
	latestStatus := latest.ko.Status

	for _, latestShard := range latestStatus.NodeGroups {
		latestReplicaCount := 0
		if latestShard.NodeGroupMembers != nil {
			if len(latestShard.NodeGroupMembers) > 0 {
				latestReplicaCount = len(latestShard.NodeGroupMembers) - 1
			}
		}
		if desiredReplicaCount := int(*desiredSpec.ReplicasPerNodeGroup); desiredReplicaCount != latestReplicaCount {
			nodeGroupID := ""
			if latestShard.NodeGroupID != nil {
				nodeGroupID = *latestShard.NodeGroupID
			}
			rm.log.V(1).Info(
				"ReplicasPerNodeGroup differs",
				"NodeGroup", nodeGroupID,
				"desired", int(*desiredSpec.ReplicasPerNodeGroup),
				"latest", latestReplicaCount,
			)
			return desiredReplicaCount - latestReplicaCount
		}
	}
	return 0
}

// diffReplicasPerNodeGroup takes desired Spec.NodeGroupConfiguration slice field into account to return
// positive number if desired replica count is greater than latest replica count
// negative number if desired replica count is less than latest replica count
// 0 otherwise
func (rm *resourceManager) diffReplicasNodeGroupConfiguration(
	desired *resource,
	latest *resource,
) int {
	desiredSpec := desired.ko.Spec
	latestStatus := latest.ko.Status
	// each shard could have different value for replica count
	latestReplicaCounts := map[string]int{}
	for _, latestShard := range latestStatus.NodeGroups {
		if latestShard.NodeGroupID == nil {
			continue
		}
		latestReplicaCount := 0
		if latestShard.NodeGroupMembers != nil {
			if len(latestShard.NodeGroupMembers) > 0 {
				latestReplicaCount = len(latestShard.NodeGroupMembers) - 1
			}
		}
		latestReplicaCounts[*latestShard.NodeGroupID] = latestReplicaCount
	}
	for _, desiredShard := range desiredSpec.NodeGroupConfiguration {
		if desiredShard.NodeGroupID == nil || desiredShard.ReplicaCount == nil {
			// no specs to compare for this shard
			continue
		}
		latestShardReplicaCount, found := latestReplicaCounts[*desiredShard.NodeGroupID]
		if !found {
			// shard not present in status
			continue
		}
		if desiredShardReplicaCount := int(*desiredShard.ReplicaCount); desiredShardReplicaCount != latestShardReplicaCount {
			rm.log.V(1).Info(
				"ReplicaCount differs",
				"NodeGroup", *desiredShard.NodeGroupID,
				"desired", int(*desiredShard.ReplicaCount),
				"latest", latestShardReplicaCount,
			)
			return desiredShardReplicaCount - latestShardReplicaCount
		}
	}
	return 0
}

// shardConfigurationsDiffer returns true if shard
// configuration differs between desired, latest resource.
func (rm *resourceManager) shardConfigurationsDiffer(
	desired *resource,
	latest *resource,
) bool {
	desiredSpec := desired.ko.Spec
	latestStatus := latest.ko.Status

	// desired shards
	var desiredShardsCount *int64 = desiredSpec.NumNodeGroups
	if desiredShardsCount == nil && desiredSpec.NodeGroupConfiguration != nil {
		numShards := int64(len(desiredSpec.NodeGroupConfiguration))
		desiredShardsCount = &numShards
	}
	if desiredShardsCount == nil {
		// no shards config in desired specs
		return false
	}

	// latest shards
	var latestShardsCount *int64 = nil
	if latestStatus.NodeGroups != nil {
		numShards := int64(len(latestStatus.NodeGroups))
		latestShardsCount = &numShards
	}

	return latestShardsCount == nil || *desiredShardsCount != *latestShardsCount
}

func (rm *resourceManager) increaseReplicaCount(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (*resource, error) {
	input, err := rm.newIncreaseReplicaCountRequestPayload(desired, latest)
	if err != nil {
		return nil, err
	}
	resp, respErr := rm.sdkapi.IncreaseReplicaCountWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "IncreaseReplicaCount", respErr)
	if respErr != nil {
		rm.log.V(1).Info("Error during IncreaseReplicaCount", "error", respErr)
		return nil, respErr
	}
	return rm.provideUpdatedResource(desired, resp.ReplicationGroup)
}

func (rm *resourceManager) decreaseReplicaCount(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (*resource, error) {
	input, err := rm.newDecreaseReplicaCountRequestPayload(desired, latest)
	if err != nil {
		return nil, err
	}
	resp, respErr := rm.sdkapi.DecreaseReplicaCountWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DecreaseReplicaCount", respErr)
	if respErr != nil {
		rm.log.V(1).Info("Error during DecreaseReplicaCount", "error", respErr)
		return nil, respErr
	}
	return rm.provideUpdatedResource(desired, resp.ReplicationGroup)
}

func (rm *resourceManager) updateShardConfiguration(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (*resource, error) {
	input, err := rm.newUpdateShardConfigurationRequestPayload(desired, latest)
	if err != nil {
		return nil, err
	}
	resp, respErr := rm.sdkapi.ModifyReplicationGroupShardConfigurationWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifyReplicationGroupShardConfiguration", respErr)
	if respErr != nil {
		rm.log.V(1).Info("Error during ModifyReplicationGroupShardConfiguration", "error", respErr)
		return nil, respErr
	}
	return rm.provideUpdatedResource(desired, resp.ReplicationGroup)
}

// newIncreaseReplicaCountRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newIncreaseReplicaCountRequestPayload(
	desired *resource,
	latest *resource,
) (*svcsdk.IncreaseReplicaCountInput, error) {
	res := &svcsdk.IncreaseReplicaCountInput{}
	desiredSpec := desired.ko.Spec

	res.SetApplyImmediately(true)
	if desiredSpec.ReplicationGroupID != nil {
		res.SetReplicationGroupId(*desiredSpec.ReplicationGroupID)
	}
	if desiredSpec.ReplicasPerNodeGroup != nil {
		res.SetNewReplicaCount(*desiredSpec.ReplicasPerNodeGroup)
	}

	latestStatus := latest.ko.Status
	// each shard could have different value for replica count
	latestReplicaCounts := map[string]int{}
	for _, latestShard := range latestStatus.NodeGroups {
		if latestShard.NodeGroupID == nil {
			continue
		}
		latestReplicaCount := 0
		if latestShard.NodeGroupMembers != nil {
			if len(latestShard.NodeGroupMembers) > 0 {
				latestReplicaCount = len(latestShard.NodeGroupMembers) - 1
			}
		}
		latestReplicaCounts[*latestShard.NodeGroupID] = latestReplicaCount
	}

	if desiredSpec.NodeGroupConfiguration != nil {
		shardsConfig := []*svcsdk.ConfigureShard{}
		for _, desiredShard := range desiredSpec.NodeGroupConfiguration {
			if desiredShard.NodeGroupID == nil {
				continue
			}
			_, found := latestReplicaCounts[*desiredShard.NodeGroupID]
			if !found {
				continue
			}
			// shard has an Id and it is present on server.
			shardConfig := &svcsdk.ConfigureShard{}
			shardConfig.SetNodeGroupId(*desiredShard.NodeGroupID)
			if desiredShard.ReplicaCount != nil {
				shardConfig.SetNewReplicaCount(*desiredShard.ReplicaCount)
			}
			shardAZs := []*string{}
			if desiredShard.PrimaryAvailabilityZone != nil {
				shardAZs = append(shardAZs, desiredShard.PrimaryAvailabilityZone)
			}
			if desiredShard.ReplicaAvailabilityZones != nil {
				for _, desiredAZ := range desiredShard.ReplicaAvailabilityZones {
					shardAZs = append(shardAZs, desiredAZ)
				}
			}
			if len(shardAZs) > 0 {
				shardConfig.SetPreferredAvailabilityZones(shardAZs)
			}
			shardsConfig = append(shardsConfig, shardConfig)
		}
		res.SetReplicaConfiguration(shardsConfig)
	}

	return res, nil
}

// newDecreaseReplicaCountRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newDecreaseReplicaCountRequestPayload(
	desired *resource,
	latest *resource,
) (*svcsdk.DecreaseReplicaCountInput, error) {
	res := &svcsdk.DecreaseReplicaCountInput{}
	desiredSpec := desired.ko.Spec

	res.SetApplyImmediately(true)
	if desiredSpec.ReplicationGroupID != nil {
		res.SetReplicationGroupId(*desiredSpec.ReplicationGroupID)
	}
	if desiredSpec.ReplicasPerNodeGroup != nil {
		res.SetNewReplicaCount(*desiredSpec.ReplicasPerNodeGroup)
	}

	latestStatus := latest.ko.Status
	// each shard could have different value for replica count
	latestReplicaCounts := map[string]int{}
	for _, latestShard := range latestStatus.NodeGroups {
		if latestShard.NodeGroupID == nil {
			continue
		}
		latestReplicaCount := 0
		if latestShard.NodeGroupMembers != nil {
			if len(latestShard.NodeGroupMembers) > 0 {
				latestReplicaCount = len(latestShard.NodeGroupMembers) - 1
			}
		}
		latestReplicaCounts[*latestShard.NodeGroupID] = latestReplicaCount
	}

	if desiredSpec.NodeGroupConfiguration != nil {
		shardsConfig := []*svcsdk.ConfigureShard{}
		for _, desiredShard := range desiredSpec.NodeGroupConfiguration {
			if desiredShard.NodeGroupID == nil {
				continue
			}
			_, found := latestReplicaCounts[*desiredShard.NodeGroupID]
			if !found {
				continue
			}
			// shard has an Id and it is present on server.
			shardConfig := &svcsdk.ConfigureShard{}
			shardConfig.SetNodeGroupId(*desiredShard.NodeGroupID)
			if desiredShard.ReplicaCount != nil {
				shardConfig.SetNewReplicaCount(*desiredShard.ReplicaCount)
			}
			shardAZs := []*string{}
			if desiredShard.PrimaryAvailabilityZone != nil {
				shardAZs = append(shardAZs, desiredShard.PrimaryAvailabilityZone)
			}
			if desiredShard.ReplicaAvailabilityZones != nil {
				for _, desiredAZ := range desiredShard.ReplicaAvailabilityZones {
					shardAZs = append(shardAZs, desiredAZ)
				}
			}
			if len(shardAZs) > 0 {
				shardConfig.SetPreferredAvailabilityZones(shardAZs)
			}
			shardsConfig = append(shardsConfig, shardConfig)
		}
		res.SetReplicaConfiguration(shardsConfig)
	}

	return res, nil
}

// newUpdateShardConfigurationRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Update API call for the resource
func (rm *resourceManager) newUpdateShardConfigurationRequestPayload(
	desired *resource,
	latest *resource,
) (*svcsdk.ModifyReplicationGroupShardConfigurationInput, error) {
	res := &svcsdk.ModifyReplicationGroupShardConfigurationInput{}

	desiredSpec := desired.ko.Spec
	latestStatus := latest.ko.Status

	// Mandatory arguments
	//	- ApplyImmediately
	//	- ReplicationGroupId
	//  - NodeGroupCount
	res.SetApplyImmediately(true)
	if desiredSpec.ReplicationGroupID != nil {
		res.SetReplicationGroupId(*desiredSpec.ReplicationGroupID)
	}
	var desiredShardsCount *int64 = desiredSpec.NumNodeGroups
	if desiredShardsCount == nil && desiredSpec.NodeGroupConfiguration != nil {
		numShards := int64(len(desiredSpec.NodeGroupConfiguration))
		desiredShardsCount = &numShards
	}
	if desiredShardsCount != nil {
		res.SetNodeGroupCount(*desiredShardsCount)
	}

	// Additional arguments
	shardsConfig := []*svcsdk.ReshardingConfiguration{}
	shardsToRetain := []*string{}
	if desiredSpec.NodeGroupConfiguration != nil {
		for _, desiredShard := range desiredSpec.NodeGroupConfiguration {
			shardConfig := &svcsdk.ReshardingConfiguration{}
			if desiredShard.NodeGroupID != nil {
				shardConfig.SetNodeGroupId(*desiredShard.NodeGroupID)
				shardsToRetain = append(shardsToRetain, desiredShard.NodeGroupID)
			}
			shardAZs := []*string{}
			if desiredShard.PrimaryAvailabilityZone != nil {
				shardAZs = append(shardAZs, desiredShard.PrimaryAvailabilityZone)
			}
			if desiredShard.ReplicaAvailabilityZones != nil {
				for _, desiredAZ := range desiredShard.ReplicaAvailabilityZones {
					shardAZs = append(shardAZs, desiredAZ)
				}
				shardConfig.SetPreferredAvailabilityZones(shardAZs)
			}
			shardsConfig = append(shardsConfig, shardConfig)
		}
	}
	// If desired nodegroup count (number of shards):
	// - increases, then (optional) provide ReshardingConfiguration
	// - decreases, then (mandatory) provide
	//	 	either 	NodeGroupsToRemove
	//	 	or 		NodeGroupsToRetain
	var latestShardsCount *int64 = nil
	if latestStatus.NodeGroups != nil {
		numShards := int64(len(latestStatus.NodeGroups))
		latestShardsCount = &numShards
	}

	increase := (desiredShardsCount != nil && latestShardsCount != nil && *desiredShardsCount > *latestShardsCount) ||
		(desiredShardsCount != nil && latestShardsCount == nil)
	decrease := desiredShardsCount != nil && latestShardsCount != nil && *desiredShardsCount < *latestShardsCount

	if increase {
		if len(shardsConfig) > 0 {
			res.SetReshardingConfiguration(shardsConfig)
		}
	} else if decrease {
		if len(shardsToRetain) == 0 {
			return nil, fmt.Errorf("Could not determine NodeGroups to retain while preparing for decrease nodegroups. " +
				"Consider specifying Spec.NodeGroupConfiguration details to resolve this error.")
		}
		res.SetNodeGroupsToRetain(shardsToRetain)
	}

	return res, nil
}

// getAnyCacheClusterIDFromNodeGroups returns a cache cluster ID from supplied node groups.
// Any cache cluster Id which is not nil is returned.
func (rm *resourceManager) getAnyCacheClusterIDFromNodeGroups(
	nodeGroups []*svcapitypes.NodeGroup,
) *string {
	if nodeGroups == nil {
		return nil
	}

	var cacheClusterId *string = nil
	for _, nodeGroup := range nodeGroups {
		if nodeGroup.NodeGroupMembers == nil {
			continue
		}
		for _, nodeGroupMember := range nodeGroup.NodeGroupMembers {
			if nodeGroupMember.CacheClusterID == nil {
				continue
			}
			cacheClusterId = nodeGroupMember.CacheClusterID
			break
		}
		if cacheClusterId != nil {
			break
		}
	}
	return cacheClusterId
}

// describeCacheCluster provides CacheCluster object
// per the supplied latest Replication Group Id
// it invokes DescribeCacheClusters API to do so
func (rm *resourceManager) describeCacheCluster(
	ctx context.Context,
	latest *resource,
) (*svcsdk.CacheCluster, error) {
	input := &svcsdk.DescribeCacheClustersInput{}

	latestStatus := latest.ko.Status
	if latestStatus.NodeGroups == nil {
		return nil, nil
	}
	cacheClusterId := rm.getAnyCacheClusterIDFromNodeGroups(latestStatus.NodeGroups)
	if cacheClusterId == nil {
		return nil, nil
	}

	input.SetCacheClusterId(*cacheClusterId)
	resp, respErr := rm.sdkapi.DescribeCacheClustersWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeCacheClusters", respErr)
	if respErr != nil {
		rm.log.V(1).Info("Error during DescribeCacheClusters", "error", respErr)
		return nil, respErr
	}
	if resp.CacheClusters == nil {
		return nil, nil
	}

	for _, cc := range resp.CacheClusters {
		if cc == nil {
			continue
		}
		return cc, nil
	}
	return nil, nil
}

// securityGroupIdsDiffer return true if
// Security Group Ids differ between desired spec and latest (from cache cluster) status
func (rm *resourceManager) securityGroupIdsDiffer(
	desired *resource,
	latest *resource,
	latestCacheCluster *svcsdk.CacheCluster,
) bool {
	if desired.ko.Spec.SecurityGroupIDs == nil {
		return false
	}

	desiredIds := []*string{}
	for _, id := range desired.ko.Spec.SecurityGroupIDs {
		if id == nil {
			continue
		}
		var value string
		value = *id
		desiredIds = append(desiredIds, &value)
	}
	sort.Slice(desiredIds, func(i, j int) bool {
		return *desiredIds[i] < *desiredIds[j]
	})

	latestIds := []*string{}
	if latestCacheCluster != nil && latestCacheCluster.SecurityGroups != nil {
		for _, latestSG := range latestCacheCluster.SecurityGroups {
			if latestSG == nil {
				continue
			}
			var value string
			value = *latestSG.SecurityGroupId
			latestIds = append(latestIds, &value)
		}
	}
	sort.Slice(latestIds, func(i, j int) bool {
		return *latestIds[i] < *latestIds[j]
	})

	if len(desiredIds) != len(latestIds) {
		return true // differ
	}
	for index, desiredId := range desiredIds {
		if *desiredId != *latestIds[index] {
			return true // differ
		}
	}
	// no difference
	return false
}

// newModifyReplicationGroupRequestPayload provides request input object
func (rm *resourceManager) newModifyReplicationGroupRequestPayload(
	desired *resource,
	latest *resource,
	latestCacheCluster *svcsdk.CacheCluster,
) *svcsdk.ModifyReplicationGroupInput {
	input := &svcsdk.ModifyReplicationGroupInput{}

	input.SetApplyImmediately(true)
	if desired.ko.Spec.ReplicationGroupID != nil {
		input.SetReplicationGroupId(*desired.ko.Spec.ReplicationGroupID)
	}

	if rm.securityGroupIdsDiffer(desired, latest, latestCacheCluster) &&
		desired.ko.Spec.SecurityGroupIDs != nil {
		ids := []*string{}
		for _, id := range desired.ko.Spec.SecurityGroupIDs {
			var value string
			value = *id
			ids = append(ids, &value)
		}
		input.SetSecurityGroupIds(ids)
	}

	if rm.engineVersionDiffer(desired, latest, latestCacheCluster) &&
		desired.ko.Spec.EngineVersion != nil {
		input.SetEngineVersion(*desired.ko.Spec.EngineVersion)
	}

	return input
}

// engineVersionDiffer return true if
// Engine Version differs between desired spec and latest (from cache cluster) status
func (rm *resourceManager) engineVersionDiffer(
	desired *resource,
	latest *resource,
	latestCacheCluster *svcsdk.CacheCluster,
) bool {
	if desired.ko.Spec.EngineVersion == nil {
		return false
	}
	desiredEV := *desired.ko.Spec.EngineVersion

	var latestEV string = ""
	if latestCacheCluster != nil && latestCacheCluster.EngineVersion != nil {
		latestEV = *latestCacheCluster.EngineVersion
	}

	return desiredEV != latestEV
}

// This method copies the data from given replicationGroup by populating it into copy of supplied resource
// and returns it.
func (rm *resourceManager) provideUpdatedResource(
	desired *resource,
	replicationGroup *svcsdk.ReplicationGroup,
) (*resource, error) {
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if replicationGroup.ARN != nil {
		arn := ackv1alpha1.AWSResourceName(*replicationGroup.ARN)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if replicationGroup.AuthTokenEnabled != nil {
		ko.Status.AuthTokenEnabled = replicationGroup.AuthTokenEnabled
	}
	if replicationGroup.AuthTokenLastModifiedDate != nil {
		ko.Status.AuthTokenLastModifiedDate = &metav1.Time{*replicationGroup.AuthTokenLastModifiedDate}
	}
	if replicationGroup.AutomaticFailover != nil {
		ko.Status.AutomaticFailover = replicationGroup.AutomaticFailover
	}
	if replicationGroup.ClusterEnabled != nil {
		ko.Status.ClusterEnabled = replicationGroup.ClusterEnabled
	}
	if replicationGroup.ConfigurationEndpoint != nil {
		f7 := &svcapitypes.Endpoint{}
		if replicationGroup.ConfigurationEndpoint.Address != nil {
			f7.Address = replicationGroup.ConfigurationEndpoint.Address
		}
		if replicationGroup.ConfigurationEndpoint.Port != nil {
			f7.Port = replicationGroup.ConfigurationEndpoint.Port
		}
		ko.Status.ConfigurationEndpoint = f7
	}
	if replicationGroup.Description != nil {
		ko.Status.Description = replicationGroup.Description
	}
	if replicationGroup.GlobalReplicationGroupInfo != nil {
		f9 := &svcapitypes.GlobalReplicationGroupInfo{}
		if replicationGroup.GlobalReplicationGroupInfo.GlobalReplicationGroupId != nil {
			f9.GlobalReplicationGroupID = replicationGroup.GlobalReplicationGroupInfo.GlobalReplicationGroupId
		}
		if replicationGroup.GlobalReplicationGroupInfo.GlobalReplicationGroupMemberRole != nil {
			f9.GlobalReplicationGroupMemberRole = replicationGroup.GlobalReplicationGroupInfo.GlobalReplicationGroupMemberRole
		}
		ko.Status.GlobalReplicationGroupInfo = f9
	}
	if replicationGroup.MemberClusters != nil {
		f11 := []*string{}
		for _, f11iter := range replicationGroup.MemberClusters {
			var f11elem string
			f11elem = *f11iter
			f11 = append(f11, &f11elem)
		}
		ko.Status.MemberClusters = f11
	}
	if replicationGroup.MultiAZ != nil {
		ko.Status.MultiAZ = replicationGroup.MultiAZ
	}
	if replicationGroup.NodeGroups != nil {
		f13 := []*svcapitypes.NodeGroup{}
		for _, f13iter := range replicationGroup.NodeGroups {
			f13elem := &svcapitypes.NodeGroup{}
			if f13iter.NodeGroupId != nil {
				f13elem.NodeGroupID = f13iter.NodeGroupId
			}
			if f13iter.NodeGroupMembers != nil {
				f13elemf1 := []*svcapitypes.NodeGroupMember{}
				for _, f13elemf1iter := range f13iter.NodeGroupMembers {
					f13elemf1elem := &svcapitypes.NodeGroupMember{}
					if f13elemf1iter.CacheClusterId != nil {
						f13elemf1elem.CacheClusterID = f13elemf1iter.CacheClusterId
					}
					if f13elemf1iter.CacheNodeId != nil {
						f13elemf1elem.CacheNodeID = f13elemf1iter.CacheNodeId
					}
					if f13elemf1iter.CurrentRole != nil {
						f13elemf1elem.CurrentRole = f13elemf1iter.CurrentRole
					}
					if f13elemf1iter.PreferredAvailabilityZone != nil {
						f13elemf1elem.PreferredAvailabilityZone = f13elemf1iter.PreferredAvailabilityZone
					}
					if f13elemf1iter.ReadEndpoint != nil {
						f13elemf1elemf4 := &svcapitypes.Endpoint{}
						if f13elemf1iter.ReadEndpoint.Address != nil {
							f13elemf1elemf4.Address = f13elemf1iter.ReadEndpoint.Address
						}
						if f13elemf1iter.ReadEndpoint.Port != nil {
							f13elemf1elemf4.Port = f13elemf1iter.ReadEndpoint.Port
						}
						f13elemf1elem.ReadEndpoint = f13elemf1elemf4
					}
					f13elemf1 = append(f13elemf1, f13elemf1elem)
				}
				f13elem.NodeGroupMembers = f13elemf1
			}
			if f13iter.PrimaryEndpoint != nil {
				f13elemf2 := &svcapitypes.Endpoint{}
				if f13iter.PrimaryEndpoint.Address != nil {
					f13elemf2.Address = f13iter.PrimaryEndpoint.Address
				}
				if f13iter.PrimaryEndpoint.Port != nil {
					f13elemf2.Port = f13iter.PrimaryEndpoint.Port
				}
				f13elem.PrimaryEndpoint = f13elemf2
			}
			if f13iter.ReaderEndpoint != nil {
				f13elemf3 := &svcapitypes.Endpoint{}
				if f13iter.ReaderEndpoint.Address != nil {
					f13elemf3.Address = f13iter.ReaderEndpoint.Address
				}
				if f13iter.ReaderEndpoint.Port != nil {
					f13elemf3.Port = f13iter.ReaderEndpoint.Port
				}
				f13elem.ReaderEndpoint = f13elemf3
			}
			if f13iter.Slots != nil {
				f13elem.Slots = f13iter.Slots
			}
			if f13iter.Status != nil {
				f13elem.Status = f13iter.Status
			}
			f13 = append(f13, f13elem)
		}
		ko.Status.NodeGroups = f13
	}
	if replicationGroup.PendingModifiedValues != nil {
		f14 := &svcapitypes.ReplicationGroupPendingModifiedValues{}
		if replicationGroup.PendingModifiedValues.AuthTokenStatus != nil {
			f14.AuthTokenStatus = replicationGroup.PendingModifiedValues.AuthTokenStatus
		}
		if replicationGroup.PendingModifiedValues.AutomaticFailoverStatus != nil {
			f14.AutomaticFailoverStatus = replicationGroup.PendingModifiedValues.AutomaticFailoverStatus
		}
		if replicationGroup.PendingModifiedValues.PrimaryClusterId != nil {
			f14.PrimaryClusterID = replicationGroup.PendingModifiedValues.PrimaryClusterId
		}
		if replicationGroup.PendingModifiedValues.Resharding != nil {
			f14f3 := &svcapitypes.ReshardingStatus{}
			if replicationGroup.PendingModifiedValues.Resharding.SlotMigration != nil {
				f14f3f0 := &svcapitypes.SlotMigration{}
				if replicationGroup.PendingModifiedValues.Resharding.SlotMigration.ProgressPercentage != nil {
					f14f3f0.ProgressPercentage = replicationGroup.PendingModifiedValues.Resharding.SlotMigration.ProgressPercentage
				}
				f14f3.SlotMigration = f14f3f0
			}
			f14.Resharding = f14f3
		}
		ko.Status.PendingModifiedValues = f14
	}
	if replicationGroup.SnapshottingClusterId != nil {
		ko.Status.SnapshottingClusterID = replicationGroup.SnapshottingClusterId
	}
	if replicationGroup.Status != nil {
		ko.Status.Status = replicationGroup.Status
	}
	rm.setStatusDefaults(ko)
	// custom set output from response
	rm.customSetOutput(desired, replicationGroup, ko)
	return &resource{ko}, nil
}
