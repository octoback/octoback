package alioss

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	_ "embed"

	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/modules/plugin"
)

//go:embed metadata.yaml
var metadata []byte

// Provider 提供阿里云 OSS Restic 仓库定义。
type Provider struct{}

// Definition 返回 AliOSS 仓库插件定义。
func (Provider) Definition() (*plugin.Definition, error) {
	return plugin.NewDefinition(metadata, &AliOSSRepository{})
}

// AliOSSRepository 保存构造 Restic S3 仓库所需的 AliOSS 配置。
type AliOSSRepository struct {
	Bucket   string
	Region   string
	Endpoint string
	Prefix   string
	Password string
}

// RepositoryConfig 返回 Restic 使用的仓库 URL、密码和 S3 凭据。
func (r AliOSSRepository) RepositoryConfig(accessKeyId, accessKeySecret string) (string, string, []string, error) {
	bucket := strings.TrimSpace(r.Bucket)
	region := strings.TrimSpace(r.Region)
	if bucket == "" || region == "" || strings.TrimSpace(accessKeyId) == "" || strings.TrimSpace(accessKeySecret) == "" || strings.TrimSpace(r.Password) == "" {
		return "", "", nil, error_.NewTextError("AliOSS 仓库配置不完整：Bucket、Region、AccessKey、Secret 和仓库密码均为必填项")
	}
	endpoint := strings.TrimSpace(r.Endpoint)
	if endpoint == "" {
		endpoint = "oss-" + region + ".aliyuncs.com"
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", nil, error_.NewTextError("AliOSS Endpoint 格式无效")
	}
	prefix := strings.Trim(strings.TrimSpace(r.Prefix), "/")
	basePath := strings.Trim(parsed.Path, "/")
	pathParts := []string{}
	if basePath != "" {
		pathParts = append(pathParts, basePath)
	}
	pathParts = append(pathParts, bucket)
	if prefix != "" {
		pathParts = append(pathParts, prefix)
	}
	repositoryURL := "s3:" + parsed.Scheme + "://" + parsed.Host + "/" + strings.Join(pathParts, "/")
	return repositoryURL, r.Password, []string{"AWS_ACCESS_KEY_ID=" + accessKeyId, "AWS_SECRET_ACCESS_KEY=" + accessKeySecret}, nil
}

// ExecuteAction 为插件管理界面提供仓库配置校验。
func (r AliOSSRepository) ExecuteAction(_ context.Context, action string, _ map[string]any) (any, error) {
	if action != "onBuild" {
		return nil, plugin.NewActionNotFoundError()
	}
	if _, _, _, err := r.RepositoryConfig("validated", "validated"); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// String 用于诊断时提供不含密钥的仓库标识。
func (r AliOSSRepository) String() string {
	return fmt.Sprintf("%s/%s", strings.TrimSpace(r.Bucket), strings.Trim(strings.TrimSpace(r.Prefix), "/"))
}
