package dbfs

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
)

// load_balance is a virtual storage policy: it holds no data itself but
// transparently forwards each upload to one of its member storage policies.
// The created entity is recorded against the selected member policy, so all
// IO afterwards (download, delete, thumb) goes straight through the member's
// driver without any load_balance involvement.

// lbCounters keeps per-policy round-robin cursors. The cursor only lives in
// memory; after a restart round-robin simply starts from the first member.
var lbCounters sync.Map // load balance policy ID -> *atomic.Uint64

// loadBalanceMembers fetches the usable member policies of a load_balance
// policy. Stale entries (deleted members, self references, nested
// load_balance members) are skipped so a partially broken configuration
// degrades instead of failing every upload.
func (f *DBFS) loadBalanceMembers(ctx context.Context, policy *ent.StoragePolicy) ([]*ent.StoragePolicy, error) {
	if policy.Settings == nil || len(policy.Settings.LoadBalancePolicyIDs) == 0 {
		return nil, serializer.NewError(serializer.CodePolicyNotAllowed,
			"Load balance policy has no member policies", nil)
	}

	members := make([]*ent.StoragePolicy, 0, len(policy.Settings.LoadBalancePolicyIDs))
	for _, id := range policy.Settings.LoadBalancePolicyIDs {
		if id <= 0 || id == policy.ID {
			continue
		}

		member, err := f.storagePolicyClient.GetPolicyByID(ctx, id)
		if err != nil || member == nil {
			continue
		}

		if member.Type == types.PolicyTypeLoadBalance {
			continue
		}

		members = append(members, member)
	}

	if len(members) == 0 {
		return nil, serializer.NewError(serializer.CodePolicyNotExist,
			"No valid member policy in load balance policy", nil)
	}

	return members, nil
}

// pickLoadBalanceMember selects one member policy according to the strategy
// configured on the load_balance policy. When advance is true the round-robin
// cursor moves forward; read-only callers (list/validation) pass false so
// they can preview the next target without skewing the distribution.
func (f *DBFS) pickLoadBalanceMember(ctx context.Context, policy *ent.StoragePolicy, advance bool) (*ent.StoragePolicy, error) {
	members, err := f.loadBalanceMembers(ctx, policy)
	if err != nil {
		return nil, err
	}

	if policy.Settings.LoadBalanceStrategy == types.LoadBalanceStrategyLeastUsed {
		return f.leastUsedMember(ctx, members), nil
	}

	// Default and round_robin: cycle through members in configured order.
	cursor, _ := lbCounters.LoadOrStore(policy.ID, new(atomic.Uint64))
	counter := cursor.(*atomic.Uint64)
	if advance {
		return members[int(counter.Add(1)-1)%len(members)], nil
	}
	return members[int(counter.Load())%len(members)], nil
}

// leastUsedMember returns the member policy with the least stored data.
// Selection errors fall back to the first member so uploads keep working.
func (f *DBFS) leastUsedMember(ctx context.Context, members []*ent.StoragePolicy) *ent.StoragePolicy {
	selected := members[0]
	minSize := int64(-1)
	for _, m := range members {
		_, size, err := f.fileClient.CountEntityByStoragePolicyID(ctx, m.ID)
		if err != nil {
			continue
		}

		if minSize < 0 || int64(size) < minSize {
			minSize = int64(size)
			selected = m
		}
	}
	return selected
}

// resolveEffectivePolicy turns a resolved policy into the concrete policy that
// will physically store an upload. Non load_balance policies pass through;
// load_balance policies are dereferenced to one member policy.
func (f *DBFS) resolveEffectivePolicy(ctx context.Context, policy *ent.StoragePolicy, advance bool) (*ent.StoragePolicy, error) {
	if policy == nil || policy.Type != types.PolicyTypeLoadBalance {
		return policy, nil
	}

	return f.pickLoadBalanceMember(ctx, policy, advance)
}

// effectivePoliciesForValidation expands a policy into every policy an upload
// under it may land on. For a load_balance policy that is the full member set,
// so pre-validation can guarantee the file satisfies the pool regardless of
// which member is ultimately selected.
func (f *DBFS) effectivePoliciesForValidation(ctx context.Context, policy *ent.StoragePolicy) ([]*ent.StoragePolicy, error) {
	if policy == nil || policy.Type != types.PolicyTypeLoadBalance {
		return []*ent.StoragePolicy{policy}, nil
	}

	members, err := f.loadBalanceMembers(ctx, policy)
	if err != nil {
		return nil, err
	}
	return members, nil
}
