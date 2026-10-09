package ssh

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/modules/plugin"
	sshaccess "handfree-work/octo-backup/internal/plugins/access/ssh"
)

//go:embed metadata.yaml
var metadata []byte

type Provider struct{}

func (Provider) Definition() (*plugin.Definition, error) {
	return plugin.NewDefinition(metadata, &SshSource{})
}

type SshSource struct {
	PluginContext *plugin.PluginContext
	AccessId      int
	Paths         []any
	ExcludePaths  []any
}

func (e *SshSource) ExecuteAction(ctx context.Context, action string, params map[string]any) (any, error) {
	switch action {
	case "onListPaths":
		result, err := e.onListPaths(ctx, params)
		if err != nil {
			return nil, error_.NewWrapError("插件 source.ssh action onListPaths 执行失败", err)
		}
		return result, nil
	default:
		return nil, plugin.NewActionNotFoundError()
	}
}

// onListPaths 根据完整插件表单读取 SSH 主机目录。
// 目录遍历接入 SSH 执行器后，只需替换此方法，action 分发和表单参数保持不变。
func (e *SshSource) onListPaths(ctx context.Context, form map[string]any) (any, error) {
	if e.AccessId <= 0 {
		return nil, error_.NewTextError("SSH 授权 Id 不能为空")
	}
	if e.PluginContext == nil || e.PluginContext.PluginService == nil {
		return nil, error_.NewTextError("插件上下文未绑定服务")
	}
	accessId := form["accessId"]
	access, err := e.PluginContext.PluginService.ResolveAccess(accessId)
	if err != nil {
		return nil, error_.NewWrapError("加载 SSH 授权配置失败", err)
	}
	_, ok := access["config"].(map[string]any)
	if !ok {
		return nil, error_.NewTextError("SSH 授权配置不存在")
	}
	config := access["config"].(map[string]any)
	definition, err := (sshaccess.Provider{}).Definition()
	if err != nil {
		return nil, error_.NewWrapError("加载 SSH 授权插件定义失败", err)
	}
	instance, err := definition.NewPluginInstance(config)
	if err != nil {
		return nil, error_.NewWrapError("创建 SSH 授权插件实例失败", err)
	}
	client := instance.(*sshaccess.SshAccess)
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	defer client.Close()
	remotePath := strings.TrimSpace(fmt.Sprint(form["path"]))
	if remotePath == "<nil>" || remotePath == "" {
		remotePath = strings.TrimSpace(fmt.Sprint(form["cursor"]))
	}
	if remotePath == "<nil>" || remotePath == "" {
		remotePath = "/"
	}
	paths, err := client.ListDirectories(remotePath)
	if err != nil {
		return nil, err
	}
	fileCount, directoryCount, err := client.CountDirectory(ctx, remotePath)
	if err != nil {
		return nil, err
	}
	return map[string]any{"paths": paths, "fileCount": fileCount, "directoryCount": directoryCount}, nil
}
