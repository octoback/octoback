package ssh

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"handfree-work/octo-backup/internal/base/error_"
	"handfree-work/octo-backup/internal/base/log_"
	"handfree-work/octo-backup/internal/modules/plugin"

	"golang.org/x/crypto/ssh"
)

//go:embed metadata.yaml
var metadata []byte

type Provider struct{}

func (Provider) Definition() (*plugin.Definition, error) {
	return plugin.NewDefinition(metadata, &SshAccess{})
}

type SshAccess struct {
	PluginContext      *plugin.PluginContext
	Host               string
	Port               int
	Username           string
	Password           string
	PrivateKey         string
	PrivateKeyPassword string
	Timeout            int
	JumpAccessId       int
	client             *ssh.Client
}

func (s *SshAccess) ExecuteAction(ctx context.Context, action string, _ map[string]any) (any, error) {
	if action == "onTest" {
		if err := s.Test(ctx); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	}
	return nil, plugin.NewActionNotFoundError()
}

// ListDirectoriesWithStats connects to the host and returns its child
// directories with file and directory counts for the path selector.
func (s *SshAccess) ListDirectoriesWithStats(ctx context.Context, remotePath string) (map[string]any, error) {
	if err := s.Connect(ctx); err != nil {
		return nil, err
	}
	defer s.Close()

	paths, err := s.ListDirectories(remotePath)
	if err != nil {
		return nil, err
	}
	fileCount, directoryCount, err := s.CountDirectory(ctx, remotePath)
	if err != nil {
		return nil, err
	}
	return map[string]any{"paths": paths, "fileCount": fileCount, "directoryCount": directoryCount}, nil
}

func (s *SshAccess) Test(ctx context.Context) error {
	if err := s.Connect(ctx); err != nil {
		return error_.NewWrapError("SSH 连接测试失败", err)
	}
	if err := s.Close(); err != nil {
		return error_.NewWrapError("关闭 SSH 测试连接失败", err)
	}
	return nil
}

func (s *SshAccess) Connect(ctx context.Context) error {
	host, user := s.Host, s.Username
	if host == "" || user == "" {
		return error_.NewTextError("SSH 主机和用户名不能为空")
	}
	port, timeout := s.Port, time.Duration(s.Timeout)*time.Second
	auth, err := s.authMethods()
	if err != nil {
		return err
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", address)
	if err != nil {
		return error_.NewTextError("连接 SSH 主机失败: %v", err)
	}
	clientConn, chans, reqs, err := ssh.NewClientConn(conn, address, &ssh.ClientConfig{User: user, Auth: auth, HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: timeout})
	if err != nil {
		_ = conn.Close()
		return error_.NewTextError("SSH 认证失败: %v", err)
	}
	s.client = ssh.NewClient(clientConn, chans, reqs)
	return nil
}

func (c *SshAccess) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

// Upload writes a file to the connected host, creating parent directories.
func (c *SshAccess) Upload(ctx context.Context, remotePath string, data []byte, mode uint32) error {
	if c == nil || c.client == nil {
		return error_.NewTextError("SSH 尚未连接")
	}
	session, err := c.client.NewSession()
	if err != nil {
		return error_.NewWrapError("创建 SSH 上传会话失败", err)
	}
	defer session.Close()
	command := "mkdir -p " + shellQuote(path.Dir(remotePath)) + " && base64 -d > " + shellQuote(remotePath) + " && chmod " + fmt.Sprintf("%o", mode) + " " + shellQuote(remotePath)
	session.Stdin = bytes.NewReader([]byte(base64.StdEncoding.EncodeToString(data)))
	if err := session.Run(command); err != nil {
		return error_.NewWrapError("上传远程文件失败", err)
	}
	return nil
}

func (c *SshAccess) Execute(ctx context.Context, command string) ([]byte, error) {
	if c == nil || c.client == nil {
		return nil, error_.NewTextError("SSH 尚未连接")
	}
	session, err := c.client.NewSession()
	if err != nil {
		return nil, error_.NewWrapError("创建 SSH 执行会话失败", err)
	}
	defer session.Close()
	log_.Logger.Info("发送 SSH 远程命令", zap.String("command", command))
	var stdout, stderr bytes.Buffer
	session.Stdout, session.Stderr = &stdout, &stderr
	if err := session.Run(command); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		output := append(stdout.Bytes(), stderr.Bytes()...)
		log_.Logger.Warn("SSH 远程命令执行失败", zap.String("command", command), zap.String("stderr", stderrText), zap.Error(err))
		if stderrText != "" {
			return output, error_.NewTextError("执行远程命令失败: %v，远程输出: %s", err, stderrText)
		}
		return output, error_.NewWrapError("执行远程命令失败", err)
	}
	output := append(stdout.Bytes(), stderr.Bytes()...)
	log_.Logger.Info("SSH 远程命令执行结果", zap.String("output", string(output)))
	return output, nil
}

func (c *SshAccess) Platform(ctx context.Context) (string, string, error) {
	output, err := c.Execute(ctx, "uname -s; uname -m")
	if err != nil {
		return "", "", error_.NewWrapError("获取 SSH 主机 platform 失败", err)
	}
	parts := strings.Fields(string(output))
	if len(parts) < 2 {
		return "", "", error_.NewTextError("SSH 主机 platform 返回无效: %s", output)
	}
	return strings.ToLower(parts[0]), strings.ToLower(parts[1]), nil
}

func (c *SshAccess) ListDirectories(path string) ([]map[string]any, error) {
	if c == nil || c.client == nil {
		return nil, error_.NewTextError("SSH 尚未连接")
	}
	if strings.TrimSpace(path) == "" {
		path = "/"
	}
	session, err := c.client.NewSession()
	if err != nil {
		return nil, error_.NewTextError("创建 SSH 会话失败: %v", err)
	}
	defer session.Close()
	command := "find " + shellQuote(path) + " -mindepth 1 -maxdepth 1 -type d -exec sh -c 'for dir do files=$(find \"$dir\" -mindepth 1 -maxdepth 1 -type f | wc -l); dirs=$(find \"$dir\" -mindepth 1 -maxdepth 1 -type d | wc -l); printf \"%s\\t%s\\t%s\\n\" \"$dir\" \"$files\" \"$dirs\"; done' sh {} +"
	output, err := session.CombinedOutput(command)
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			return nil, error_.NewWrapError(fmt.Sprintf("读取远程目录 %q 失败", path), err)
		}
		return nil, error_.NewWrapError(fmt.Sprintf("读取远程目录 %q 失败: %s", path, detail), err)
	}
	paths := make([]map[string]any, 0)
	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			parts := strings.Split(line, "\t")
			if len(parts) != 3 {
				continue
			}
			fileCount, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			directoryCount, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
			paths = append(paths, map[string]any{"title": parts[0], "key": parts[0], "isLeaf": directoryCount == 0, "fileCount": fileCount, "directoryCount": directoryCount})
		}
	}
	return paths, nil
}

// CountDirectory 返回远程目录的直接文件数和子目录数。
func (c *SshAccess) CountDirectory(ctx context.Context, path string) (int, int, error) {
	if c == nil || c.client == nil {
		return 0, 0, error_.NewTextError("SSH 尚未连接")
	}
	command := "files=$(find " + shellQuote(path) + " -mindepth 1 -maxdepth 1 -type f | wc -l); dirs=$(find " + shellQuote(path) + " -mindepth 1 -maxdepth 1 -type d | wc -l); printf '%s\\t%s\\n' \"$files\" \"$dirs\""
	output, err := c.Execute(ctx, command)
	if err != nil {
		return 0, 0, error_.NewWrapError("统计远程目录内容失败", err)
	}
	parts := strings.Split(strings.TrimSpace(string(output)), "\t")
	if len(parts) != 2 {
		return 0, 0, error_.NewTextError("远程目录统计结果无效")
	}
	files, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, error_.NewWrapError("解析远程文件数失败", err)
	}
	directories, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, error_.NewWrapError("解析远程子目录数失败", err)
	}
	return files, directories, nil
}

func (s *SshAccess) authMethods() ([]ssh.AuthMethod, error) {
	if s.Password != "" {
		return []ssh.AuthMethod{ssh.Password(s.Password)}, nil
	}
	key := strings.TrimSpace(s.PrivateKey)
	if key == "" {
		return nil, error_.NewTextError("SSH 密码或私钥不能为空")
	}
	var signer ssh.Signer
	var err error
	if passphrase := s.PrivateKeyPassword; passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(key), []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey([]byte(key))
	}
	if err != nil {
		return nil, error_.NewTextError("解析 SSH 私钥失败: %v", err)
	}
	return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
}

func intValue(value any, fallback int) int {
	if parsed, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(value))); err == nil && parsed > 0 {
		return parsed
	}
	return fallback
}
func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
