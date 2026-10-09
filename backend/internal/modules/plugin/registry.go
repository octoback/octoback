package plugin

import (
	"context"
	"fmt"
	"handfree-work/octo-backup/internal/base/error_"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type PluginType string

const (
	TypeAccess       PluginType = "access"
	TypeRepository   PluginType = "repository"
	TypeSource       PluginType = "source"
	TypeNotification PluginType = "notification"
)

type FieldSpec struct {
	Key         string         `yaml:"key" json:"key"`
	Title       string         `yaml:"title" json:"title"`
	Type        string         `yaml:"type" json:"type"`
	Required    bool           `yaml:"required" json:"required"`
	Encrypt     bool           `yaml:"encrypt" json:"encrypt"`
	Default     any            `yaml:"default,omitempty" json:"default,omitempty"`
	Placeholder string         `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
	Helper      string         `yaml:"helper,omitempty" json:"helper,omitempty"`
	MergeScript string         `yaml:"mergeScript,omitempty" json:"mergeScript,omitempty"`
	Component   map[string]any `yaml:"component,omitempty" json:"component,omitempty"`
	Options     []FieldOption  `yaml:"options,omitempty" json:"options,omitempty"`
}

type FieldOption struct {
	Label string `yaml:"label" json:"label"`
	Value string `yaml:"value" json:"value"`
}
type ActionSpec struct {
	Name       string `yaml:"name" json:"name"`
	Title      string `yaml:"title,omitempty" json:"title,omitempty"`
	Permission string `yaml:"permission,omitempty" json:"permission,omitempty"`
}
type Metadata struct {
	Type        PluginType   `yaml:"type" json:"type"`
	Name        string       `yaml:"name" json:"name"`
	Title       string       `yaml:"title" json:"title"`
	Description string       `yaml:"description" json:"description"`
	Version     string       `yaml:"version" json:"version"`
	Method      string       `yaml:"method,omitempty" json:"method,omitempty"`
	AccessType  string       `yaml:"accessType,omitempty" json:"accessType,omitempty"`
	Icon        string       `yaml:"icon,omitempty" json:"icon,omitempty"`
	Group       string       `yaml:"group,omitempty" json:"group,omitempty"`
	Fields      []FieldSpec  `yaml:"fields,omitempty" json:"fields,omitempty"`
	Actions     []ActionSpec `yaml:"actions,omitempty" json:"actions,omitempty"`
}
type PluginInstance interface {
	ExecuteAction(context.Context, string, map[string]any) (any, error)
}
type NewPluginInstance func(map[string]any) (PluginInstance, error)
type Definition struct {
	Metadata          Metadata
	NewPluginInstance NewPluginInstance
	newWithContext    func(*PluginContext, map[string]any) (PluginInstance, error)
}

func NewActionNotFoundError() *error_.CodedError {
	return error_.NewTextError("插件 action 不存在")
}
func NewDefinition(data []byte, instanceFactory any) (*Definition, error) {
	var metadata Metadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, error_.NewWrapError("解析插件元数据", err)
	}
	newPluginInstance, err := pluginFactory(instanceFactory)
	if err != nil {
		return nil, err
	}
	if metadata.Type == "" || metadata.Name == "" || metadata.Title == "" || metadata.Version == "" {
		return nil, error_.NewTextError("插件元数据缺少必填字段")
	}
	seen := map[string]bool{}
	for _, field := range metadata.Fields {
		if field.Key == "" || seen[field.Key] {
			return nil, error_.NewTextError("插件字段无效: %s", field.Key)
		}
		seen[field.Key] = true
	}
	seen = map[string]bool{}
	for _, action := range metadata.Actions {
		if !strings.HasPrefix(action.Name, "on") || len(action.Name) <= 2 || seen[action.Name] {
			return nil, error_.NewTextError("插件 action 必须以 on 开头且不可重复: %s", action.Name)
		}
		seen[action.Name] = true
	}
	withContext := func(ctx *PluginContext, config map[string]any) (PluginInstance, error) {
		return newPluginInstance(config)
	}
	if callback, ok := instanceFactory.(func(*PluginContext, map[string]any) (PluginInstance, error)); ok {
		withContext = callback
	}
	if typ := reflect.TypeOf(instanceFactory); typ != nil && typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Struct {
		withContext = func(ctx *PluginContext, config map[string]any) (PluginInstance, error) {
			return newReflectedPlugin(typ, ctx, config)
		}
	}
	return &Definition{Metadata: metadata, NewPluginInstance: newPluginInstance, newWithContext: withContext}, nil
}

func newReflectedPlugin(typ reflect.Type, ctx *PluginContext, config map[string]any) (PluginInstance, error) {
	value := reflect.New(typ.Elem())
	for key, raw := range config {
		if raw == nil || (reflect.ValueOf(raw).Kind() == reflect.String && strings.TrimSpace(reflect.ValueOf(raw).String()) == "") {
			continue
		}
		for i := 0; i < value.Elem().NumField(); i++ {
			field := value.Elem().Type().Field(i)
			if strings.EqualFold(field.Name, key) && value.Elem().Field(i).CanSet() {
				converted, err := convertConfigValue(raw, field.Type)
				if err != nil {
					return nil, err
				}
				value.Elem().Field(i).Set(converted)
			}
		}
	}
	for i := 0; i < value.Elem().NumField(); i++ {
		field := value.Elem().Type().Field(i)
		if field.Name == "PluginContext" && value.Elem().Field(i).CanSet() && value.Elem().Field(i).Type() == reflect.TypeOf((*PluginContext)(nil)) {
			value.Elem().Field(i).Set(reflect.ValueOf(ctx))
		}
	}
	return value.Interface().(PluginInstance), nil
}
func pluginFactory(factory any) (NewPluginInstance, error) {
	if callback, ok := factory.(func(*PluginContext, map[string]any) (PluginInstance, error)); ok {
		return func(config map[string]any) (PluginInstance, error) { return callback(nil, config) }, nil
	}
	if callback, ok := factory.(NewPluginInstance); ok {
		return callback, nil
	}
	if callback, ok := factory.(func(map[string]any) (PluginInstance, error)); ok {
		return callback, nil
	}
	typ := reflect.TypeOf(factory)
	if typ == nil || typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		return nil, error_.NewTextError("插件实例工厂无效")
	}
	return func(config map[string]any) (PluginInstance, error) {
		value := reflect.New(typ.Elem())
		for key, raw := range config {
			if raw == nil || (reflect.ValueOf(raw).Kind() == reflect.String && strings.TrimSpace(reflect.ValueOf(raw).String()) == "") {
				continue
			}
			for fieldIndex := 0; fieldIndex < value.Elem().NumField(); fieldIndex++ {
				field := value.Elem().Type().Field(fieldIndex)
				if !strings.EqualFold(field.Name, key) || !value.Elem().Field(fieldIndex).CanSet() {
					continue
				}
				converted, err := convertConfigValue(raw, field.Type)
				if err != nil {
					return nil, error_.NewWrapError(fmt.Sprintf("插件字段 %s 配置无效", key), err)
				}
				value.Elem().Field(fieldIndex).Set(converted)
				break
			}
		}
		return value.Interface().(PluginInstance), nil
	}, nil
}

func convertConfigValue(raw any, target reflect.Type) (reflect.Value, error) {
	value := reflect.ValueOf(raw)
	if value.IsValid() && value.Type().AssignableTo(target) {
		return value, nil
	}
	if target.Kind() == reflect.String {
		return reflect.ValueOf(fmt.Sprint(raw)).Convert(target), nil
	}
	if target.Kind() >= reflect.Int && target.Kind() <= reflect.Int64 {
		var number int64
		if value.IsValid() && value.Kind() >= reflect.Int && value.Kind() <= reflect.Int64 {
			number = value.Int()
		} else if _, err := fmt.Sscan(fmt.Sprint(raw), &number); err != nil {
			return reflect.Value{}, err
		}
		result := reflect.New(target).Elem()
		result.SetInt(number)
		return result, nil
	}
	return reflect.Value{}, error_.NewTextError("无法转换为 %s", target)
}

type Registry struct{ definitions map[string]*Definition }

func NewRegistry() *Registry { return &Registry{definitions: map[string]*Definition{}} }
func (r *Registry) Register(d *Definition) error {
	if d == nil || d.Metadata.Name == "" {
		return error_.NewTextError("插件定义不能为空")
	}
	if _, ok := r.definitions[d.Metadata.Name]; ok {
		return error_.NewTextError("插件已注册: %s", d.Metadata.Name)
	}
	r.definitions[d.Metadata.Name] = d
	return nil
}
func (r *Registry) Get(name string) (*Definition, bool) { d, ok := r.definitions[name]; return d, ok }
func (r *Registry) NewInstance(name string, config map[string]any, contexts ...*PluginContext) (PluginInstance, error) {
	d, ok := r.Get(name)
	if !ok {
		return nil, error_.NewTextError("插件不存在: %s", name)
	}
	if len(contexts) > 0 && d.newWithContext != nil {
		return d.newWithContext(contexts[0], config)
	}
	return d.NewPluginInstance(config)
}
func (r *Registry) Metadata(t PluginType, name string) []Metadata {
	out := []Metadata{}
	for _, d := range r.definitions {
		if (t == "" || d.Metadata.Type == t) && (name == "" || d.Metadata.Name == name) {
			out = append(out, d.Metadata)
		}
	}
	return out
}
func (r *Registry) Execute(ctx context.Context, name, action string, params map[string]any) (any, error) {
	return r.ExecuteWithConfig(ctx, name, action, params, params)
}
func (r *Registry) ExecuteWithConfig(ctx context.Context, name, action string, config, params map[string]any) (any, error) {
	d, ok := r.Get(name)
	if !ok {
		return nil, error_.NewTextError("插件不存在: %s", name)
	}
	allowed := false
	for _, a := range d.Metadata.Actions {
		if a.Name == action {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, NewActionNotFoundError()
	}
	p, err := r.NewInstance(name, config)
	if err != nil {
		return nil, err
	}
	return p.ExecuteAction(ctx, action, params)
}
