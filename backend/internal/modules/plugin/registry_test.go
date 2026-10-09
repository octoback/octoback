package plugin_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"handfree-work/octo-backup/internal/modules/plugin"
)

type testPlugin struct{}

type numericConfigPlugin struct {
	Port int
}

func (numericConfigPlugin) ExecuteAction(context.Context, string, map[string]any) (any, error) {
	return nil, nil
}

func (testPlugin) ExecuteAction(_ context.Context, action string, _ map[string]any) (any, error) {
	if action == "onTest" {
		return map[string]any{"ok": true}, nil
	}
	return nil, plugin.NewActionNotFoundError()
}

func TestGenericBuildActionDoesNotReturnPluginConfig(t *testing.T) {
	definition, err := plugin.NewDefinition([]byte("type: repository\nname: repository.example\ntitle: Example\nversion: 1.0.0\nactions:\n  - name: onBuild\n"), func(map[string]any) (plugin.PluginInstance, error) {
		return actionResultPlugin{result: map[string]any{"ok": true}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	registry := plugin.NewRegistry()
	if err := registry.Register(definition); err != nil {
		t.Fatal(err)
	}
	got, err := registry.ExecuteWithConfig(context.Background(), "repository.example", "onBuild", map[string]any{"password": "secret"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(got, map[string]any{"password": "secret"}) {
		t.Fatalf("onBuild exposed plugin config: %#v", got)
	}
}

func TestRegistrySkipsEmptyOptionalNumericConfig(t *testing.T) {
	definition, err := plugin.NewDefinition([]byte("type: access\nname: access.numeric\ntitle: Numeric\nversion: 1.0.0\n"), &numericConfigPlugin{})
	if err != nil {
		t.Fatal(err)
	}
	registry := plugin.NewRegistry()
	if err := registry.Register(definition); err != nil {
		t.Fatal(err)
	}
	instance, err := registry.NewInstance("access.numeric", map[string]any{"port": ""})
	if err != nil {
		t.Fatalf("empty numeric config should be ignored: %v", err)
	}
	if instance.(*numericConfigPlugin).Port != 0 {
		t.Fatalf("empty numeric config changed default value: %#v", instance)
	}
}

type actionResultPlugin struct{ result any }

func (p actionResultPlugin) ExecuteAction(context.Context, string, map[string]any) (any, error) {
	return p.result, nil
}

func TestRegistryParsesYAMLAndDispatchesDeclaredAction(t *testing.T) {
	definition, err := plugin.NewDefinition([]byte("type: repository\nname: repository.example\ntitle: Example\ndescription: Example repository\nversion: 1.0.0\naccessType: access.ssh\nfields:\n  - key: accessId\n    title: Access Id\n    type: number\n    mergeScript: 'return { component: { pluginName: ctx.compute(({form}) => form.config?.accessType) } }'\nactions:\n  - name: onTest\n    title: Test\n    permission: write\n"), func(map[string]any) (plugin.PluginInstance, error) { return testPlugin{}, nil })
	if err != nil {
		t.Fatalf("NewDefinition() error = %v", err)
	}
	if definition.Metadata.AccessType != "access.ssh" || definition.Metadata.Fields[0].MergeScript == "" {
		t.Fatalf("metadata access selector settings were not parsed: %#v", definition.Metadata)
	}
	registry := plugin.NewRegistry()
	if err := registry.Register(definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	got, err := registry.Execute(context.Background(), "repository.example", "onTest", map[string]any{})
	if err != nil || got == nil {
		t.Fatalf("Execute() = (%v, %v), want result", got, err)
	}
	if _, err := registry.Execute(context.Background(), "repository.example", "OnTest", nil); err == nil {
		t.Fatal("Execute() accepted non-on action")
	}
	if _, err := registry.Execute(context.Background(), "repository.example", "onDeleteEverything", nil); err == nil {
		t.Fatal("Execute() accepted undeclared action")
	}
}

func TestDefinitionRejectsInvalidActionName(t *testing.T) {
	_, err := plugin.NewDefinition([]byte("type: access\nname: access.example\ntitle: Example\ndescription: Example\nversion: 1.0.0\nactions:\n  - name: test\n"), func(map[string]any) (plugin.PluginInstance, error) { return testPlugin{}, nil })
	if err == nil {
		t.Fatal("NewDefinition() accepted action without on prefix")
	}
}

func TestRepositoryMetadataDeclaresMatchingAccessPlugin(t *testing.T) {
	wantAccessTypes := map[string]string{
		"sftp": "access.ssh",
	}
	for repository, wantAccessType := range wantAccessTypes {
		t.Run(repository, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "plugins", "repository", repository, "metadata.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			definition, err := plugin.NewDefinition(data, func(map[string]any) (plugin.PluginInstance, error) { return testPlugin{}, nil })
			if err != nil {
				t.Fatal(err)
			}
			if definition.Metadata.AccessType != wantAccessType {
				t.Fatalf("accessType = %q, want %q", definition.Metadata.AccessType, wantAccessType)
			}
			for _, field := range definition.Metadata.Fields {
				if field.Key == "accessId" {
					if field.MergeScript == "" {
						t.Fatal("accessId field has no mergeScript")
					}
					return
				}
			}
			t.Fatal("accessId field not found")
		})
	}
}
