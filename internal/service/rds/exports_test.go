// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package rds

// Exports for use in tests only.
var (
	ResourceCertificate             = resourceCertificate
	ResourceCluster                 = resourceCluster
	ResourceClusterSnapshot         = resourceClusterSnapshot
	ResourceEventSubscription       = resourceEventSubscription
	ResourceParameterGroup          = resourceParameterGroup
	ResourceProxy                   = resourceProxy
	ResourceProxyDefaultTargetGroup = resourceProxyDefaultTargetGroup
	ResourceProxyEndpoint           = resourceProxyEndpoint
	ResourceProxyTarget             = resourceProxyTarget
	ResourceSnapshot                = resourceSnapshot
	ResourceSnapshotCopy            = resourceSnapshotCopy
	ResourceSubnetGroup             = resourceSubnetGroup

	FindDBClusterSnapshotByID                  = findDBClusterSnapshotByID
	FindDBInstanceByID                         = findDBInstanceByIDSDKv1
	FindDBParameterGroupByName                 = findDBParameterGroupByName
	FindDBProxyByName                          = findDBProxyByName
	FindDBProxyEndpointByTwoPartKey            = findDBProxyEndpointByTwoPartKey
	FindDBProxyTargetByFourPartKey             = findDBProxyTargetByFourPartKey
	FindDBSnapshotByID                         = findDBSnapshotByID
	FindDBSubnetGroupByName                    = findDBSubnetGroupByName
	FindDefaultCertificate                     = findDefaultCertificate
	FindDefaultDBProxyTargetGroupByDBProxyName = findDefaultDBProxyTargetGroupByDBProxyName
	FindEventSubscriptionByID                  = findEventSubscriptionByID
	ListTags                                   = listTags
	NewBlueGreenOrchestrator                   = newBlueGreenOrchestrator
	ParameterGroupModifyChunk                  = parameterGroupModifyChunk
	ParseDBInstanceARN                         = parseDBInstanceARN
	ProxyTargetParseResourceID                 = proxyTargetParseResourceID
	WaitBlueGreenDeploymentDeleted             = waitBlueGreenDeploymentDeleted
	WaitBlueGreenDeploymentAvailable           = waitBlueGreenDeploymentAvailable
	WaitDBInstanceAvailable                    = waitDBInstanceAvailableSDKv2
	WaitDBInstanceDeleted                      = waitDBInstanceDeleted

	ErrCodeInvalidAction               = errCodeInvalidAction
	ErrCodeInvalidParameterCombination = errCodeInvalidParameterCombination
	ErrCodeInvalidParameterValue       = errCodeInvalidParameterValue
)
