package restic

import (
	"strings"
	"testing"

	sshaccess "handfree-work/octo-backup/internal/plugins/access/ssh"
)

func TestRepositorySftpCommandSelectsConfiguredPrivateKey(t *testing.T) {
	config := ClientConfig{
		RepositoryAccessId: 17,
		RepositoryAccess: &sshaccess.SshAccess{
			Host:       "repo.example.com",
			Port:       2222,
			Username:   "backup",
			PrivateKey: "private-key",
		},
	}

	command := repositorySftpCommand(&config)
	if !strings.Contains(command, "-i '.ssh/key_17'") {
		t.Fatalf("SFTP command = %q, want the key for access 17", command)
	}
}

func TestRepositorySftpCommandDoesNotOverridePasswordAccessIdentity(t *testing.T) {
	config := ClientConfig{
		RepositoryAccessId: 17,
		RepositoryAccess: &sshaccess.SshAccess{
			Host:     "repo.example.com",
			Port:     22,
			Username: "backup",
			Password: "login-password",
		},
	}

	command := repositorySftpCommand(&config)
	if strings.Contains(command, " -i ") {
		t.Fatalf("password-authenticated SFTP command unexpectedly selects a private key: %q", command)
	}
}
