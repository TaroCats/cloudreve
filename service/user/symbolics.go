package user

import (
	"context"
	"errors"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
)

// CreateDefaultSymbolics creates share shortcuts in the new user's root
// folder, driven by the default_symbolics admin setting (a list of numeric
// share IDs). Each shortcut is a symbolic entry carrying the share redirect
// metadata, so opening it lands on the share.
func CreateDefaultSymbolics(ctx context.Context, dep dependency.Dep, u *ent.User) {
	shareIDs := dep.SettingProvider().DefaultSymbolics(ctx)
	if len(shareIDs) == 0 {
		return
	}

	l := dep.Logger()
	hasher := dep.HashIDEncoder()
	shareClient := dep.ShareClient()

	loadCtx := context.WithValue(ctx, inventory.LoadShareFile{}, true)
	loadCtx = context.WithValue(loadCtx, inventory.LoadShareUser{}, true)

	mgr := manager.NewFileManager(dep, u)
	defer mgr.Recycle()

	root, err := fs.NewUriFromString("cloudreve://my")
	if err != nil {
		l.Warning("Failed to build root uri for default symbolics: %s", err)
		return
	}

	for _, sid := range shareIDs {
		if sid <= 0 {
			continue
		}

		share, err := shareClient.GetByID(loadCtx, sid)
		if err != nil {
			l.Warning("Default symbolics: share %d not found: %s", sid, err)
			continue
		}
		if share.Edges.File == nil || share.Edges.User == nil {
			l.Warning("Default symbolics: share %d missing file or owner edges", sid)
			continue
		}

		fileType := types.FileTypeFile
		if share.Edges.File.Type == int(types.FileTypeFolder) {
			fileType = types.FileTypeFolder
		}

		target := root.Join(share.Edges.File.Name)
		_, err = mgr.Create(ctx, target, fileType, fs.WithMetadata(map[string]string{
			dbfs.MetadataSharedRedirect: fs.NewShareUri(hashid.EncodeShareID(hasher, share.ID), ""),
			dbfs.MetadataSharedOwner:    hashid.EncodeUserID(hasher, share.Edges.User.ID),
		}))
		if err != nil {
			// Conflicts are fine (e.g. user already has an entry with the
			// same name); anything else is worth logging but must not fail
			// user creation.
			if !isFileExisted(err) {
				l.Warning("Default symbolics: failed to create shortcut for share %d: %s", sid, err)
			}
		}
	}
}

func isFileExisted(err error) bool {
	var appErr serializer.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == serializer.CodeObjectExist
	}
	return false
}
