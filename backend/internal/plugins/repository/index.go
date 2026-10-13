package repository

import (
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/modules/plugin"
	"handfree-work/octo-backup/internal/plugins/repository/alioss"
	"handfree-work/octo-backup/internal/plugins/repository/sftp"
)

type provider interface {
	Definition() (*plugin.Definition, error)
}

func Register(r *plugin.Registry) error {
	providers := []provider{sftp.Provider{}, alioss.Provider{}}
	for _, p := range providers {
		d, err := p.Definition()
		if err != nil {
			return error_.NewWrapError("加载仓库插件", err)
		}
		if err := r.Register(d); err != nil {
			return err
		}
	}
	return nil
}
