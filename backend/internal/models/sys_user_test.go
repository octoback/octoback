package models_test

import (
	"path/filepath"
	"testing"

	"handfree-work/octo-backup/internal/base/db_"
	"handfree-work/octo-backup/internal/models"
)

func TestRenamedModelsUseCanonicalTables(t *testing.T) {
	if got := (models.SysUser{}).TableName(); got != "sys_user" {
		t.Fatalf("SysUser.TableName() = %q, want %q", got, "sys_user")
	}
	if got := (models.RepositoryInfo{}).TableName(); got != "repository_info" {
		t.Fatalf("RepositoryInfo.TableName() = %q, want %q", got, "repository_info")
	}

	database, err := db_.OpenSQLite(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatalf("OpenSQLite() error = %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db_.Migrate(database, &models.SysUser{}, &models.RepositoryInfo{}); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	for _, table := range []string{"sys_user", "repository_info"} {
		if !database.Migrator().HasTable(table) {
			t.Errorf("canonical table %q was not created", table)
		}
	}
}
