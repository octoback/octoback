package plugin

import "handfree-work/octo-backup/internal/base/error_"

// PluginContext 为插件提供共享服务和实例状态访问能力。
type PluginContext struct {
	PluginService PluginVarsService
}

// PluginVarsService 定义插件实例状态的读取和持久化能力。
type PluginVarsService interface {
	GetVar(string) any
	SetVar(string, any) error
	ResolveAccess(any) (map[string]any, error)
}

// SetVar 设置插件实例状态并持久化。
func (c *PluginContext) SetVar(key string, value any) error {
	if c == nil || c.PluginService == nil {
		return error_.NewTextError("插件上下文未绑定服务")
	}
	return c.PluginService.SetVar(key, value)
}

// GetVar 读取插件实例状态。
func (c *PluginContext) GetVar(key string) any {
	if c == nil || c.PluginService == nil {
		return nil
	}
	return c.PluginService.GetVar(key)
}
