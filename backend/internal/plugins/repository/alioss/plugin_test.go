package alioss

import (
	"reflect"
	"testing"
)

func TestRepositoryConfigBuildsDefaultAliOSSEndpoint(t *testing.T) {
	repository := AliOSSRepository{Bucket: "archive", Region: "cn-hangzhou", Prefix: "/backups/weekly/", Password: "repo-password"}
	repositoryURL, password, environment, err := repository.RepositoryConfig("access-id", "access-secret")
	if err != nil {
		t.Fatal(err)
	}
	if repositoryURL != "s3:https://oss-cn-hangzhou.aliyuncs.com/archive/backups/weekly" || password != "repo-password" {
		t.Fatalf("repository config = (%q, %q), want normalized OSS URL and repository password", repositoryURL, password)
	}
	wantEnvironment := []string{"AWS_ACCESS_KEY_ID=access-id", "AWS_SECRET_ACCESS_KEY=access-secret"}
	if !reflect.DeepEqual(environment, wantEnvironment) {
		t.Fatalf("environment = %#v, want %#v", environment, wantEnvironment)
	}
}

func TestRepositoryConfigUsesCustomEndpointAndRejectsInvalidConfig(t *testing.T) {
	repository := AliOSSRepository{Bucket: "archive", Region: "cn-hangzhou", Endpoint: "https://oss-cn-hangzhou-internal.aliyuncs.com", Password: "repo-password"}
	got, _, _, err := repository.RepositoryConfig("access-id", "access-secret")
	if err != nil || got != "s3:https://oss-cn-hangzhou-internal.aliyuncs.com/archive" {
		t.Fatalf("repository URL = %q, error = %v", got, err)
	}
	repository.Endpoint = "https://user:pass@example.com"
	if _, _, _, err := repository.RepositoryConfig("access-id", "access-secret"); err == nil {
		t.Fatal("endpoint with user info must be rejected")
	}
	repository.Endpoint = ""
	if _, _, _, err := repository.RepositoryConfig("access-id", ""); err == nil {
		t.Fatal("missing access key secret must be rejected")
	}
	repository.Endpoint = "https://oss-cn-hangzhou-internal.aliyuncs.com/custom/base"
	got, _, _, err = repository.RepositoryConfig("access-id", "access-secret")
	if err != nil || got != "s3:https://oss-cn-hangzhou-internal.aliyuncs.com/custom/base/archive" {
		t.Fatalf("endpoint base path was not preserved: %q, error = %v", got, err)
	}
}

func TestProviderDefinitionEncryptsCredentials(t *testing.T) {
	definition, err := (Provider{}).Definition()
	if err != nil {
		t.Fatal(err)
	}
	if definition.Metadata.AccessType != "access.aliyunAccess" {
		t.Fatalf("AliOSS access type = %q, want access.aliyunAccess", definition.Metadata.AccessType)
	}
	secrets := map[string]bool{"password": false}
	for _, field := range definition.Metadata.Fields {
		if _, ok := secrets[field.Key]; ok {
			secrets[field.Key] = field.Encrypt
		}
	}
	for key, encrypted := range secrets {
		if !encrypted {
			t.Errorf("%s must be encrypted", key)
		}
	}
}
