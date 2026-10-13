package aliyun

import "testing"

func TestCredentialsRequiresBothAccessKeyValues(t *testing.T) {
	access := AliyunAccess{AccessKeyId: "id", AccessKeySecret: "secret"}
	gotId, gotSecret, err := access.Credentials()
	if err != nil || gotId != "id" || gotSecret != "secret" {
		t.Fatalf("credentials = (%q, %q, %v)", gotId, gotSecret, err)
	}
	access.AccessKeySecret = ""
	if _, _, err := access.Credentials(); err == nil {
		t.Fatal("missing AccessKey Secret must be rejected")
	}
}

func TestProviderDefinitionEncryptsCredentials(t *testing.T) {
	definition, err := (Provider{}).Definition()
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range definition.Metadata.Fields {
		if field.Key == "accessKeyId" || field.Key == "accessKeySecret" {
			if !field.Encrypt || !field.Required {
				t.Fatalf("field %s must be required and encrypted: %#v", field.Key, field)
			}
		}
	}
}
