package workflows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
)

type (
	MigratePolicyTask struct {
		*queue.DBTask

		l        logging.Logger
		state    *MigratePolicyTaskState
		progress queue.Progresses
	}

	// MigratePolicyTaskState is persisted between runs so an interrupted
	// migration resumes from the last successfully moved entity.
	MigratePolicyTaskState struct {
		SrcPolicyID  int  `json:"src_policy_id"`
		DstPolicyID  int  `json:"dst_policy_id"`
		OwnerID      int  `json:"owner_id,omitempty"` // 0 = all owners (admin)
		Total        int  `json:"total"`
		Migrated     int  `json:"migrated"`
		Failed       int  `json:"failed"`
		BytesMoved   int64 `json:"bytes_moved"`
		LastEntityID int  `json:"last_entity_id"`
		DeleteSource bool `json:"delete_source"`
		LastError    string `json:"last_error,omitempty"`
	}
)

const (
	MigratePolicyBatchSize = 100

	ProgressTypeMigratePolicy = "migrate_policy"
)

func init() {
	queue.RegisterResumableTaskFactory(queue.PolicyMigrateTaskType, NewMigratePolicyTaskFromModel)
}

func NewMigratePolicyTask(ctx context.Context, u *ent.User, srcPolicyID, dstPolicyID int, deleteSource bool, ownerID int) (queue.Task, error) {
	if srcPolicyID <= 0 || dstPolicyID <= 0 || srcPolicyID == dstPolicyID {
		return nil, fmt.Errorf("invalid source/destination storage policy: %w", queue.CriticalErr)
	}

	state := &MigratePolicyTaskState{
		SrcPolicyID:  srcPolicyID,
		DstPolicyID:  dstPolicyID,
		OwnerID:      ownerID,
		DeleteSource: deleteSource,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	return &MigratePolicyTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.PolicyMigrateTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
			DirectOwner: u,
		},
	}, nil
}

func NewMigratePolicyTaskFromModel(t *ent.Task) queue.Task {
	return &MigratePolicyTask{
		DBTask: &queue.DBTask{
			Task: t,
		},
	}
}

func (m *MigratePolicyTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()

	m.Lock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	if m.progress[ProgressTypeMigratePolicy] == nil {
		m.progress[ProgressTypeMigratePolicy] = &queue.Progress{}
	}
	m.Unlock()

	state := &MigratePolicyTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %s (%w)", err, queue.CriticalErr)
	}
	m.state = state

	next, err := m.migrateBatch(ctx, dep)

	newStateStr, marshalErr := json.Marshal(m.state)
	if marshalErr != nil {
		return task.StatusError, fmt.Errorf("failed to marshal state: %w", marshalErr)
	}

	m.Lock()
	m.Task.PrivateState = string(newStateStr)
	m.Unlock()
	return next, err
}

// migrateBatch moves up to MigratePolicyBatchSize entities and returns
// StatusResumable while more remain. Counting runs once on the first pass.
func (m *MigratePolicyTask) migrateBatch(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	fc := dep.FileClient()
	sc := dep.StoragePolicyClient()

	if m.state.Total == 0 {
		countRes, err := fc.ListEntities(ctx, &inventory.ListEntityParameters{
			UserID:          m.state.OwnerID,
			StoragePolicyID: m.state.SrcPolicyID,
			PaginationArgs: &inventory.PaginationArgs{
				Page:     0,
				PageSize: 1,
			},
		})
		if err != nil {
			return task.StatusError, fmt.Errorf("failed to count entities: %w", queue.CriticalErr)
		}
		m.state.Total = countRes.PaginationResults.TotalItems
		if m.state.Total == 0 {
			return task.StatusCompleted, nil
		}
	}

	srcPolicy, err := sc.GetPolicyByID(ctx, m.state.SrcPolicyID)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to load source policy: %w", queue.CriticalErr)
	}
	dstPolicy, err := sc.GetPolicyByID(ctx, m.state.DstPolicyID)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to load destination policy: %w", queue.CriticalErr)
	}

	mgr := manager.NewFileManager(dep, m.Owner())
	defer mgr.Recycle()

	entities, err := fc.ListEntities(ctx, &inventory.ListEntityParameters{
		UserID:          m.state.OwnerID,
		StoragePolicyID: m.state.SrcPolicyID,
		IDAfter:         m.state.LastEntityID,
		PaginationArgs: &inventory.PaginationArgs{
			Page:     0,
			PageSize: MigratePolicyBatchSize,
			OrderBy:  "id",
			Order:    "asc",
		},
	})
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to list entities: %w", err)
	}

	if len(entities.Entities) == 0 {
		return task.StatusCompleted, nil
	}

	for _, e := range entities.Entities {
		if e.ID <= m.state.LastEntityID {
			continue
		}

		if err := m.migrateOne(ctx, dep, mgr, e, srcPolicy, dstPolicy); err != nil {
			m.state.Failed++
			m.state.LastError = fmt.Sprintf("entity %d: %s", e.ID, err)
			m.l.Warning("Failed to migrate entity %d: %s", e.ID, err)
			// Advance the checkpoint past the failure so the task can finish;
			// the source object is left untouched and remains recoverable.
			m.state.LastEntityID = e.ID
			m.saveProgress()
			continue
		}

		m.state.LastEntityID = e.ID
		m.state.Migrated++
		m.state.BytesMoved += e.Size
		m.saveProgress()
	}

	// More to do if this batch was full: suspend and resume for the next batch.
	if len(entities.Entities) >= MigratePolicyBatchSize {
		m.ResumeAfter(0)
		return task.StatusSuspending, nil
	}
	return task.StatusCompleted, nil
}

// migrateOne copies a single entity to the destination policy and repoints
// the entity row at it. When DeleteSource is set the origin object is removed.
func (m *MigratePolicyTask) migrateOne(ctx context.Context, dep dependency.Dep, mgr manager.FileManager,
	e *ent.Entity, srcPolicy, dstPolicy *ent.StoragePolicy) error {

	src, err := mgr.GetEntitySource(ctx, e.ID)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	// Destination save path mirrors the source layout so the object naming
	// rules of the destination policy still apply.
	req := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Size:     e.Size,
			SavePath: e.Source,
		},
		File: src,
	}

	dstHandler, err := mgr.GetStorageDriver(ctx, dstPolicy)
	if err != nil {
		return fmt.Errorf("get destination driver: %w", err)
	}
	if err := dstHandler.Put(ctx, req); err != nil {
		return fmt.Errorf("write to destination: %w", err)
	}

	if err := dep.FileClient().UpdateEntityPolicy(ctx, e, dstPolicy.ID, e.Source); err != nil {
		return fmt.Errorf("repoint entity: %w", err)
	}

	if m.state.DeleteSource {
		srcHandler, err := mgr.GetStorageDriver(ctx, srcPolicy)
		if err == nil {
			if failed, err := srcHandler.Delete(ctx, e.Source); err != nil {
				m.l.Warning("Migrated entity %d but failed to delete source object %v: %s", e.ID, failed, err)
			}
		}
	}

	return nil
}

func (m *MigratePolicyTask) saveProgress() {
	m.Lock()
	defer m.Unlock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	m.progress[ProgressTypeMigratePolicy] = &queue.Progress{
		Current: int64(m.state.Migrated + m.state.Failed),
		Total:   int64(m.state.Total),
	}
}

func (m *MigratePolicyTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()
	return m.progress
}

func (m *MigratePolicyTask) Summarize(hasher hashid.Encoder) *queue.Summary {
	m.Lock()
	defer m.Unlock()
	if m.state == nil {
		return &queue.Summary{}
	}
	return &queue.Summary{
		Props: map[string]any{
			"src_policy":  m.state.SrcPolicyID,
			"dst_policy":  m.state.DstPolicyID,
			"migrated":    m.state.Migrated,
			"failed":      m.state.Failed,
			"bytes_moved": m.state.BytesMoved,
		},
	}
}
