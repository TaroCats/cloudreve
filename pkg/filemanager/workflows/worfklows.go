package workflows

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
)

const (
	TaskTempPath                 = "fm_workflows"
	slaveProgressRefreshInterval = 5 * time.Second
)

type NodeState struct {
	NodeID int `json:"node_id"`

	progress queue.Progresses
}

// allocateNode allocates a node for the task. When the owner's group
// restricts AllowedNodes, selection is limited to that subset.
func allocateNode(ctx context.Context, dep dependency.Dep, state *NodeState, capability types.NodeCapability, owner *ent.User) (cluster.Node, error) {
	np, err := dep.NodePool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get node pool: %w", err)
	}

	var allowed []int
	if owner != nil && owner.GroupUsers > 0 {
		if g, err := dep.GroupClient().GetByID(ctx, owner.GroupUsers); err == nil {
			allowed = g.Settings.AllowedNodes
		}
	}

	node, err := np.Get(ctx, capability, state.NodeID, allowed)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	state.NodeID = node.ID()
	return node, nil
}

// prepareSlaveTaskCtx prepares the context for the slave task.
func prepareSlaveTaskCtx(ctx context.Context, props *types.SlaveTaskProps) context.Context {
	ctx = context.WithValue(ctx, cluster.SlaveNodeIDCtx{}, strconv.Itoa(props.NodeID))
	ctx = context.WithValue(ctx, cluster.MasterSiteUrlCtx{}, props.MasterSiteURl)
	ctx = context.WithValue(ctx, cluster.MasterSiteVersionCtx{}, props.MasterSiteVersion)
	ctx = context.WithValue(ctx, cluster.MasterSiteIDCtx{}, props.MasterSiteID)
	return ctx
}

func prepareTempFolder(ctx context.Context, dep dependency.Dep, t queue.Task) (string, error) {
	settings := dep.SettingProvider()
	tempPath := util.DataPath(path.Join(settings.TempPath(ctx), TaskTempPath, strconv.Itoa(t.ID())))
	if err := util.CreatNestedFolder(tempPath); err != nil {
		return "", fmt.Errorf("failed to create temp folder: %w", err)
	}

	dep.Logger().Info("Temp folder created: %s", tempPath)
	return tempPath, nil
}
