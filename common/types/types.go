package types

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PodStatusRequest is Message.Content which comes from edge
type PodStatusRequest struct {
	UID    types.UID
	Name   string
	Status v1.PodStatus
}

// ExtendResource is the extended resource detail that comes from edge
type ExtendResource struct {
	Name     string            `json:"name,omitempty"`
	Type     string            `json:"type,omitempty"`
	Capacity resource.Quantity `json:"capacity,omitempty"`
}

// NodeStatusRequest is Message.Content which comes from edge
type NodeStatusRequest struct {
	UID             types.UID
	Status          v1.NodeStatus
	ExtendResources map[v1.ResourceName][]ExtendResource
}

// NodeUpgradeJobRequest is upgrade msg coming from cloud to edge
type NodeUpgradeJobRequest struct {
	UpgradeID   string
	HistoryID   string
	Version     string
	UpgradeTool string
	Image       string
}

// NodeUpgradeJobResponse is used to report status msg to cloudhub https service
type NodeUpgradeJobResponse struct {
	UpgradeID   string
	HistoryID   string
	NodeName    string
	FromVersion string
	ToVersion   string
	Status      string
	Reason      string
}

// ObjectResp is the object that api-server response
type ObjectResp struct {
	Object metaV1.Object
	Err    error
}
