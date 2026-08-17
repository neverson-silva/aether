package git

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func Clone(ctx context.Context, url, branch, dest string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git binary not found")
	}
	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, dest)
	cmd := exec.CommandContext(ctx, "git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func CommitHEAD(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func VerifyGitHubSignature(payload []byte, signature string, secret []byte) error {
	if signature == "" {
		return errors.New("missing signature")
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(signature)) {
		return errors.New("invalid signature")
	}
	return nil
}

type PushEvent struct {
	Ref        string `json:"ref"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

func ParsePushEvent(payload []byte) (*PushEvent, error) {
	var ev PushEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	if ev.Ref == "" {
		return nil, errors.New("not a push event")
	}
	return &ev, nil
}

func (e *PushEvent) Branch() string {
	return strings.TrimPrefix(e.Ref, "refs/heads/")
}
