package user

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// UserPolicy is the caller-visible view of a storage policy bound to the
// caller's group. No credentials are exposed.
type UserPolicy struct {
	ID     int    `json:"id"`
	HashID string `json:"hash_id"`
	Name   string `json:"name"`
}

type UserPoliciesResponse struct {
	Policies []UserPolicy `json:"policies"`
}

// UserNode is the caller-visible view of a processing node.
type UserNode struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Capabilities *boolset.BooleanSet `json:"capabilities"`
}

type UserNodesResponse struct {
	Nodes []UserNode `json:"nodes"`
}

// GetUserNodes lists active task processing nodes, filtered by the caller's
// group AllowedNodes when set. Used for task node selection.
func GetUserNodes(c *gin.Context) (*UserNodesResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)

	nodes, err := dep.NodeClient().ListActiveNodes(c, nil)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list nodes", err)
	}

	var allowed []int
	if u.Edges.Group != nil {
		allowed = u.Edges.Group.Settings.AllowedNodes
	}

	res := make([]UserNode, 0, len(nodes))
	for _, n := range nodes {
		if len(allowed) > 0 && !lo.Contains(allowed, n.ID) {
			continue
		}
		res = append(res, UserNode{ID: n.ID, Name: n.Name, Capabilities: n.Capabilities})
	}

	return &UserNodesResponse{Nodes: res}, nil
}

// GetUserPolicies lists the storage policies available to the caller's
// user group. Used by the file manager for policy binding and migration.
func GetUserPolicies(c *gin.Context) (*UserPoliciesResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()

	group := u.Edges.Group
	if group == nil {
		return &UserPoliciesResponse{Policies: []UserPolicy{}}, nil
	}

	sc := dep.StoragePolicyClient()
	policies, err := sc.ListByGroup(c, group)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list storage policies", err)
	}

	return &UserPoliciesResponse{
		Policies: lo.Map(policies, func(p *ent.StoragePolicy, _ int) UserPolicy {
			return UserPolicy{
				ID:     p.ID,
				HashID: hashid.EncodePolicyID(hasher, p.ID),
				Name:   p.Name,
			}
		}),
	}, nil
}
