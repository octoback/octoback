package logic

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/models"
	"handfree-work/octo-backup/internal/modules/plugin"
	"handfree-work/octo-backup/internal/svc"
)

func TestAttachSnapshotAssociationsIncludesBackupSummary(t *testing.T) {
	database, err := db_.OpenSQLite(filepath.Join(t.TempDir(), "snapshot-summary.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.AutoMigrate(&models.BackupLog{}); err != nil {
		t.Fatal(err)
	}
	repositoryId := int64(7)
	snapshotId := "snapshot-summary"
	if err := database.Create(&models.BackupLog{PlanId: 3, RepositoryId: &repositoryId, SnapshotId: &snapshotId, Summary: `{"snapshot_id":"snapshot-summary","total_files_processed":12,"total_bytes_processed":4096}`}).Error; err != nil {
		t.Fatal(err)
	}
	service := NewPluginInstanceService(context.Background(), &svc.ServiceContext{Db: database})
	attached := service.attachSnapshotAssociations(repositoryId, []any{map[string]any{"id": snapshotId}}).([]any)
	entry := attached[0].(map[string]any)
	if entry["summary"] == nil {
		t.Fatalf("summary not attached: %#v", entry)
	}
	encoded, err := json.Marshal(entry["summary"])
	if err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(encoded, &summary); err != nil {
		t.Fatal(err)
	}
	if summary["snapshot_id"] != snapshotId || summary["total_files_processed"] != float64(12) {
		t.Fatalf("summary = %#v", summary)
	}
}

type configCaptureExecutor struct {
	config map[string]any
	params map[string]any
}

func (e *configCaptureExecutor) ExecuteAction(_ context.Context, _ string, params map[string]any) (any, error) {
	e.params = params
	return e.config, nil
}

func TestMergePluginConfigClearsBlankSecretsAndUpdatesOtherFields(t *testing.T) {
	existing := map[string]any{"secretId": "old-id", "secretKey": "old-key", "region": "ap-shanghai"}
	incoming := map[string]any{"secretId": "new-id", "secretKey": "", "region": ""}
	fields := []plugin.FieldSpec{
		{Key: "secretId", Encrypt: true},
		{Key: "secretKey", Encrypt: true},
		{Key: "region"},
	}

	got := mergePluginConfig(existing, incoming, fields)
	want := map[string]any{"secretId": "new-id", "secretKey": "", "region": ""}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("config[%q] = %#v, want %#v", key, got[key], value)
		}
	}
}

func TestMergePluginConfigClearsBlankValuesForEveryFieldType(t *testing.T) {
	got := mergePluginConfig(
		map[string]any{"text": "old", "secret": "old-secret", "number": 22},
		map[string]any{"text": "", "secret": "", "number": ""},
		[]plugin.FieldSpec{{Key: "text"}, {Key: "secret", Encrypt: true}, {Key: "number"}},
	)
	for key, value := range map[string]any{"text": "", "secret": "", "number": ""} {
		if got[key] != value {
			t.Errorf("config[%q] = %#v, want empty string", key, got[key])
		}
	}
}

func TestMergePluginConfigDropsConfiguredMarkerAndUnknownFields(t *testing.T) {
	got := mergePluginConfig(map[string]any{"region": "cn"}, map[string]any{"configured": true, "unexpected": "value"}, []plugin.FieldSpec{{Key: "region"}})
	if len(got) != 1 || got["region"] != "cn" {
		t.Fatalf("merged config = %#v, want only preserved region", got)
	}
}

func TestRepositoryReferencesAccessMatchesExactId(t *testing.T) {
	referenced, err := repositoryReferencesAccess("accessId: 42\npath: backup\n", 42)
	if err != nil || !referenced {
		t.Fatalf("reference = %v, error = %v; want referenced", referenced, err)
	}

	referenced, err = repositoryReferencesAccess("accessId: 142\n", 42)
	if err != nil || referenced {
		t.Fatalf("reference = %v, error = %v; want not referenced", referenced, err)
	}
}

func TestRepositoryReferencesAccessRejectsMalformedConfig(t *testing.T) {
	if _, err := repositoryReferencesAccess("accessId: [", 42); err == nil {
		t.Fatal("malformed config should return an error")
	}
}

func TestPresentPluginConfigRedactsSecretsAndReturnsPlainFields(t *testing.T) {
	config := presentPluginConfig("secretId: id-1\nsecretKey: key-1\nregion: ap-shanghai\n", []plugin.FieldSpec{
		{Key: "secretId", Encrypt: true},
		{Key: "secretKey", Encrypt: true},
		{Key: "region"},
	}, nil)
	if config["secretId"] != "****" || config["secretKey"] != "ke*-1" {
		t.Fatalf("secret values should be masked: %#v", config)
	}
	if config["region"] != "ap-shanghai" || config["secretIdConfigured"] != nil || config["secretKeyConfigured"] != nil {
		t.Fatalf("present config = %#v", config)
	}
}

func TestPluginCreateUpdateAndConcurrentMasterKeyInitialization(t *testing.T) {
	database, err := db_.OpenSQLite(filepath.Join(t.TempDir(), "plugins.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db_.Migrate(database, &models.PluginInstance{}, &models.SysSetting{}); err != nil {
		t.Fatal(err)
	}
	if !database.Migrator().HasTable("plugin_instance") {
		t.Fatal("plugin instances should use the plugin_instance table")
	}

	registry := plugin.NewRegistry()
	definition, err := plugin.NewDefinition([]byte("type: access\nname: access.test\ntitle: Test\nversion: 1\nfields:\n  - {key: secret, title: Secret, type: password, encrypt: true}\n  - {key: account, title: Account, type: string, required: true}\n  - {key: region, title: Region, type: string}\nactions:\n  - {name: onTest}\n"), func(config map[string]any) (plugin.PluginInstance, error) {
		return &configCaptureExecutor{config: config}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(definition); err != nil {
		t.Fatal(err)
	}
	var repositoryExecutor *configCaptureExecutor
	repositoryDefinition, err := plugin.NewDefinition([]byte("type: repository\nname: repository.test\ntitle: Repository\nversion: 1\nfields:\n  - {key: accessId, title: Access Id, type: number}\n  - {key: path, title: Path, type: string, required: true}\n  - {key: password, title: Password, type: password, encrypt: true, required: true}\nactions:\n  - {name: onBuild}\n"), func(config map[string]any) (plugin.PluginInstance, error) {
		repositoryExecutor = &configCaptureExecutor{config: config}
		return repositoryExecutor, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(repositoryDefinition); err != nil {
		t.Fatal(err)
	}
	service := NewPluginInstanceService(context.Background(), &svc.ServiceContext{Db: database, Plugins: registry})

	const workers = 8
	start := make(chan struct{})
	var wait sync.WaitGroup
	errCh := make(chan error, workers)
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := service.secretCodec()
			errCh <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent secretCodec() error = %v", err)
		}
	}

	created, err := service.Create(&PluginInstanceInput{
		Name:       "腾讯云",
		PluginType: "access",
		PluginName: "access.test",
		Config:     map[string]any{"secret": "top-secret", "account": "account-1", "region": "cn-shanghai"},
	})
	if err != nil {
		t.Fatal(err)
	}
	idPtr, ok := created["id"].(*int64)
	if !ok || idPtr == nil {
		t.Fatalf("created plugin id = %#v, want *int64", created["id"])
	}
	id := *idPtr
	info, err := service.Info(id)
	if err != nil || info["config"].(map[string]any)["account"] != "account-1" {
		t.Fatalf("plugin info = %#v, error = %v; want complete config", info, err)
	}
	simple, err := service.GetSimpleByIDs([]int64{id, id + 100})
	if err != nil || len(simple) != 1 || simple[0]["name"] != "腾讯云" || simple[0]["pluginName"] != "access.test" || simple[0]["icon"] != "" || simple[0]["config"] != nil {
		t.Fatalf("simple plugins = %#v, error = %v", simple, err)
	}
	var stored models.PluginInstance
	if err := database.First(&stored, id).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored.ConfigYAML, "top-secret") {
		t.Fatalf("stored config contains plaintext secret: %q", stored.ConfigYAML)
	}
	if createdConfig := created["config"].(map[string]any); createdConfig["secret"] != "to******et" || createdConfig["secretConfigured"] != nil || createdConfig["account"] != "account-1" {
		t.Fatalf("created config should be masked while returning plain fields: %#v", createdConfig)
	}
	if _, err := service.Update(id, &PluginInstanceInput{Config: map[string]any{"account": ""}}); err == nil {
		t.Fatal("update should reject clearing a required field")
	}

	if _, err := service.Update(id, &PluginInstanceInput{Name: "腾讯云修改", Config: map[string]any{"secret": "", "region": ""}}); err != nil {
		t.Fatal(err)
	}
	if err := database.First(&stored, id).Error; err != nil {
		t.Fatal(err)
	}
	codec, err := service.secretCodec()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := codec.DecodeYAML([]byte(stored.ConfigYAML), secretFields(definition.Metadata.Fields))
	if err != nil {
		t.Fatal(err)
	}
	if decoded["secret"] != "" || decoded["region"] != "" {
		t.Fatalf("updated config = %#v", decoded)
	}
	if _, err := service.Action(id, "onTest", nil); err != nil {
		t.Fatalf("action should load decrypted config: %v", err)
	}

	page, err := service.Page(&PluginInstancePageQuery{Offset: -10, Limit: 10, PluginType: "access", Name: "修改"})
	if err != nil || page.Offset != 0 || page.Total != 1 || len(page.Records) != 1 {
		t.Fatalf("filtered page = %#v, error = %v", page, err)
	}
	if _, ok := page.Records[0]["config"]; ok {
		t.Fatalf("plugin page record should not include config: %#v", page.Records[0])
	}
	page, err = service.Page(&PluginInstancePageQuery{PluginType: "access", PluginName: "access.test"})
	if err != nil || page.Total != 1 || len(page.Records) != 1 || page.Records[0]["pluginName"] != "access.test" {
		t.Fatalf("pluginName-filtered page = %#v, error = %v", page, err)
	}

	repository, err := service.Create(&PluginInstanceInput{Name: "repo", PluginName: "repository.test", Config: map[string]any{"accessId": id, "path": "backup", "password": "repo-secret"}})
	if err != nil {
		t.Fatal(err)
	}
	repositoryId := *repository["id"].(*int64)
	if _, err := service.Update(repositoryId, &PluginInstanceInput{Config: map[string]any{"password": "new-secret"}}); err == nil {
		t.Fatal("repository password should not be changeable")
	}
	if _, err := service.Update(repositoryId, &PluginInstanceInput{Config: map[string]any{"password": ""}}); err != nil {
		t.Fatalf("blank repository password should preserve the configured password: %v", err)
	}
	result, err := service.Action(repositoryId, "onBuild", nil)
	if err != nil {
		t.Fatalf("repository action: %v", err)
	}
	if result == nil {
		t.Fatal("repository action should return a result")
	}
	params := map[string]any{"path": "/etc", "access": "untrusted"}
	if _, err := service.Action(repositoryId, "onBuild", params); err != nil {
		t.Fatal(err)
	}
	if repositoryExecutor.params["path"] != "/etc" || params["access"] != "untrusted" {
		t.Fatal("action must preserve form parameters and leave caller parameters unchanged")
	}
	if err := service.Delete(id); err == nil {
		t.Fatal("delete should reject an access instance referenced by a repository")
	}
	if err := service.Delete(repositoryId); err != nil {
		t.Fatalf("delete repository: %v", err)
	}
	if err := service.Delete(id); err != nil {
		t.Fatalf("delete unreferenced access: %v", err)
	}
}
