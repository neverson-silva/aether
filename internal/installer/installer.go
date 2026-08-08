package installer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"aether/internal/config"
	"aether/internal/core"
)

type InstallResult struct {
	StateDir string
	Email    string
	Password string
	APIAddr  string
}

func Install(email, name, password string) (*InstallResult, error) {
	if password == "" {
		password = randomPassword()
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := ensureRuntime(cfg); err != nil {
		return nil, err
	}
	c, err := core.New(cfg)
	if err != nil {
		return nil, err
	}
	defer c.Stop(context.Background())
	user, org, err := c.CreateUserAndOrg(email, name, password)
	if err != nil {
		return nil, fmt.Errorf("criação do admin falhou: %w", err)
	}
	_ = user
	_ = org
	if err := writeSystemdUnit(cfg); err != nil {
		log.Printf("avisos: %v", err)
	}
	return &InstallResult{
		StateDir: cfg.StateDir,
		Email:    email,
		Password: password,
		APIAddr:  cfg.APIAddr,
	}, nil
}

func ensureRuntime(cfg *config.Config) error {
	if _, err := exec.LookPath("podman"); err == nil {
		return nil
	}
	if os.Geteuid() != 0 {
		return errors.New("nenhum runtime de containers (podman) encontrado; execute como root para instalação automática")
	}
	if _, err := os.Stat("/etc/os-release"); err != nil {
		return errors.New("sistema não suportado para instalação automática de runtime")
	}
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return err
	}
	var cmds [][]string
	if strings.Contains(string(data), "debian") || strings.Contains(string(data), "ubuntu") {
		cmds = [][]string{
			{"apt-get", "update", "-qq"},
			{"apt-get", "install", "-y", "-qq", "podman", "crun", "conmon", "buildah", "skopeo", "fuse-overlayfs"},
		}
	} else if strings.Contains(string(data), "fedora") || strings.Contains(string(data), "rhel") || strings.Contains(string(data), "centos") {
		cmds = [][]string{
			{"dnf", "install", "-y", "podman", "crun", "conmon", "buildah", "skopeo", "fuse-overlayfs"},
		}
	} else {
		return errors.New("distribuição não suportada; instale podman manualmente")
	}
	for _, args := range cmds {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("instalação do runtime falhou (%s): %s", args[0], strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func writeSystemdUnit(cfg *config.Config) error {
	if os.Geteuid() != 0 {
		return nil
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return nil
	}
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	bin, err = filepath.EvalSymlinks(bin)
	if err != nil {
		bin, _ = os.Executable()
	}
	unit := `[Unit]
Description=Aether platform core
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=` + bin + ` serve
Environment=AETHER_STATE=` + cfg.StateDir + `
Restart=on-failure
RestartSec=3
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`
	path := "/etc/systemd/system/aether-core.service"
	if err := os.WriteFile(path, []byte(unit), 0o644); err != nil {
		return err
	}
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "aether-core.service").Run()
	return nil
}

func Uninstall(purge bool) error {
	if os.Geteuid() == 0 {
		exec.Command("systemctl", "disable", "--now", "aether-core.service").Run()
		os.Remove("/etc/systemd/system/aether-core.service")
		exec.Command("systemctl", "daemon-reload").Run()
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if purge {
		if err := os.RemoveAll(cfg.StateDir); err != nil {
			return err
		}
	}
	return nil
}

func Update(updateURL, expectedSHA256 string) error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	bin, err = filepath.EvalSymlinks(bin)
	if err != nil {
		bin, _ = os.Executable()
	}
	dir := filepath.Join(filepath.Dir(bin), "..", "updates")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	prev := filepath.Join(dir, "aether.previous")
	if err := copyFile(bin, prev); err != nil {
		return err
	}
	tmp := bin + ".new"
	if err := download(updateURL, tmp); err != nil {
		return err
	}
	if expectedSHA256 != "" {
		data, err := os.ReadFile(tmp)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != strings.ToLower(expectedSHA256) {
			os.Remove(tmp)
			return errors.New("checksum do binário não confere")
		}
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, bin); err != nil {
		return err
	}
	if os.Geteuid() == 0 {
		exec.Command("systemctl", "restart", "aether-core.service").Run()
	}
	return nil
}

func Rollback() error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	bin, err = filepath.EvalSymlinks(bin)
	if err != nil {
		return err
	}
	prev := filepath.Join(filepath.Dir(bin), "..", "updates", "aether.previous")
	if _, err := os.Stat(prev); err != nil {
		return errors.New("nenhum binário anterior encontrado")
	}
	if err := copyFile(prev, bin); err != nil {
		return err
	}
	if err := os.Chmod(bin, 0o755); err != nil {
		return err
	}
	if os.Geteuid() == 0 {
		exec.Command("systemctl", "restart", "aether-core.service").Run()
	}
	return nil
}

func download(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download falhou: status %d", resp.StatusCode)
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o700)
}

func randomPassword() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	b := make([]byte, 20)
	for i := range b {
		b[i] = chars[int(raw[i])%len(chars)]
	}
	return string(b)
}
