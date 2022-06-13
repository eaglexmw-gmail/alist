package operations

import (
	"context"
	"github.com/Xhofe/go-cache"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/singleflight"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/pkg/errors"
	stdpath "path"
)

// In order to facilitate adding some other things before and after file operations

// List files in storage, not contains virtual file
// TODO: cache, and prevent cache breakdown
func List(ctx context.Context, account driver.Driver, path string) ([]driver.FileInfo, error) {
	return account.List(ctx, path)
}

func Get(ctx context.Context, account driver.Driver, path string) (driver.FileInfo, error) {
	if r, ok := account.GetAddition().(driver.RootFolderId); ok && utils.PathEqual(path, "/") {
		return model.FileWithId{
			Id: r.GetRootFolderId(),
			File: model.File{
				Name:     "root",
				Size:     0,
				Modified: account.GetAccount().Modified,
				IsFolder: true,
			},
		}, nil
	}
	if r, ok := account.GetAddition().(driver.IRootFolderPath); ok && utils.PathEqual(path, r.GetRootFolderPath()) {
		return model.File{
			Name:     "root",
			Size:     0,
			Modified: account.GetAccount().Modified,
			IsFolder: true,
		}, nil
	}
	dir, name := stdpath.Split(path)
	files, err := List(ctx, account, dir)
	if err != nil {
		return nil, errors.WithMessage(err, "failed get parent list")
	}
	for _, f := range files {
		if f.GetName() == name {
			return f, nil
		}
	}
	return nil, errors.WithStack(driver.ErrorObjectNotFound)
}

var linkCache = cache.NewMemCache[*driver.Link]()
var linkG singleflight.Group[*driver.Link]

// Link get link, if is an url. should have an expiry time
func Link(ctx context.Context, account driver.Driver, path string, args driver.LinkArgs) (*driver.Link, error) {
	key := stdpath.Join(account.GetAccount().VirtualPath, path)
	if link, ok := linkCache.Get(key); ok {
		return link, nil
	}
	fn := func() (*driver.Link, error) {
		link, err := account.Link(ctx, path, args)
		if err != nil {
			return nil, errors.WithMessage(err, "failed get link")
		}
		if link.Expiration != nil {
			linkCache.Set(key, link, cache.WithEx[*driver.Link](*link.Expiration))
		}
		return link, nil
	}
	link, err, _ := linkG.Do(key, fn)
	return link, err
}

func MakeDir(ctx context.Context, account driver.Driver, path string) error {
	return account.MakeDir(ctx, path)
}

func Move(ctx context.Context, account driver.Driver, srcPath, dstPath string) error {
	return account.Move(ctx, srcPath, dstPath)
}

func Rename(ctx context.Context, account driver.Driver, srcPath, dstName string) error {
	return account.Rename(ctx, srcPath, dstName)
}

// Copy Just copy file[s] in an account
func Copy(ctx context.Context, account driver.Driver, srcPath, dstPath string) error {
	return account.Copy(ctx, srcPath, dstPath)
}

func Remove(ctx context.Context, account driver.Driver, path string) error {
	return account.Remove(ctx, path)
}

func Put(ctx context.Context, account driver.Driver, parentPath string, file driver.FileStream) error {
	return account.Put(ctx, parentPath, file)
}
