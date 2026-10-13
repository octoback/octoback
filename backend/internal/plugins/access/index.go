package access

import (
	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/modules/plugin"
	"handfree-work/octo-backup/internal/plugins/access/aliyun"
	"handfree-work/octo-backup/internal/plugins/access/ftp"
	"handfree-work/octo-backup/internal/plugins/access/ssh"
)

type provider interface {
	Definition() (*plugin.Definition, error)
}

func Register(r *plugin.Registry) error {
	providers := []provider{ssh.Provider{}, ftp.Provider{}, aliyun.Provider{}}
	for _, p := range providers {
		d, err := p.Definition()
		if err != nil {
			return error_.NewWrapError("加载授权插件", err)
		}
		if err := r.Register(d); err != nil {
			return err
		}
	}
	return nil
}
