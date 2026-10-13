package aliyun

import (
	"context"
	_ "embed"

	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/modules/plugin"
)

//go:embed metadata.yaml
var metadata []byte

// Provider 提供阿里云 AccessKey 授权定义。
type Provider struct{}

// Definition 返回阿里云授权插件定义。
func (Provider) Definition() (*plugin.Definition, error) {
	return plugin.NewDefinition(metadata, &AliyunAccess{})
}

// AliyunAccess 保存阿里云对象存储访问凭据。
type AliyunAccess struct {
	AccessKeyId     string
	AccessKeySecret string
}

// Credentials 返回供对象存储使用的 AccessKey 凭据。
func (a AliyunAccess) Credentials() (string, string, error) {
	if a.AccessKeyId == "" || a.AccessKeySecret == "" {
		return "", "", error_.NewTextError("阿里云授权配置不完整：AccessKey ID 和 AccessKey Secret 均为必填项")
	}
	return a.AccessKeyId, a.AccessKeySecret, nil
}

// ExecuteAction 校验阿里云授权配置。
func (a AliyunAccess) ExecuteAction(_ context.Context, action string, _ map[string]any) (any, error) {
	if action != "onTest" {
		return nil, plugin.NewActionNotFoundError()
	}
	if _, _, err := a.Credentials(); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}
