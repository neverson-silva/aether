package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/runtime"
)

const snapshotChunkSize = 1 << 20

func (c *Core) chunkDir() string {
	return filepath.Join(c.Cfg.DataDir, "snapshots", "chunks")
}

func (c *Core) CreateVolumeSnapshot(ctx context.Context, orgID, appID, volume, name string) (*domain.Snapshot, error) {
	if err := os.MkdirAll(c.chunkDir(), 0o750); err != nil {
		return nil, err
	}
	helper := "aether-snap-" + strings.ReplaceAll(volume, "/", "-")
	c.Driver.Remove(ctx, helper, true)
	id, err := c.Driver.Run(ctx, runtime.RunSpec{
		Name:    helper,
		Image:   "busybox:latest",
		Cmd:     []string{"sleep", "3600"},
		Volumes: []runtime.VolumeMount{{Source: volume, Target: "/data"}},
		Restart: "no",
		Labels:  map[string]string{"aether.role": "snapshot"},
	})
	if err != nil {
		return nil, fmt.Errorf("helper: %w", err)
	}
	defer c.Driver.Remove(ctx, id, true)
	stream, err := c.Driver.ExecStream(ctx, id, runtime.ExecRequest{
		Command: []string{"tar", "-cf", "-", "-C", "/data", "."},
	})
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	manifest, total, dedupSaved, err := c.chunkStream(stream.Stdout())
	if err != nil {
		return nil, err
	}
	sn := &domain.Snapshot{
		ID:         "snap-" + domain.NewID(),
		OrgID:      orgID,
		AppID:      appID,
		Volume:     volume,
		Name:       name,
		Size:       total,
		Chunks:     len(manifest),
		DedupSaved: dedupSaved,
		CreatedAt:  time.Now().UTC(),
	}
	manifestPath := filepath.Join(c.chunkDir(), sn.ID+".json")
	raw, _ := json.Marshal(manifest)
	if err := os.WriteFile(manifestPath, raw, 0o640); err != nil {
		return nil, err
	}
	if err := c.Store.CreateSnapshot(sn); err != nil {
		return nil, err
	}
	return sn, nil
}

func (c *Core) chunkStream(r io.Reader) ([]string, int64, int64, error) {
	var (
		manifest   []string
		total      int64
		dedupSaved int64
	)
	buf := &bytes.Buffer{}
	enc, err := zstd.NewWriter(buf)
	if err != nil {
		return nil, 0, 0, err
	}
	flush := func() error {
		if err := enc.Close(); err != nil {
			return err
		}
		if buf.Len() > 0 {
			chunk := append([]byte{}, buf.Bytes()...)
			sum := sha256.Sum256(chunk)
			hash := hex.EncodeToString(sum[:])
			path := filepath.Join(c.chunkDir(), hash+".zst")
			if _, err := os.Stat(path); err == nil {
				dedupSaved += int64(len(chunk))
			} else if err := os.WriteFile(path, chunk, 0o640); err != nil {
				return err
			}
			manifest = append(manifest, hash)
		}
		buf.Reset()
		enc, err = zstd.NewWriter(buf)
		return err
	}
	readBuf := make([]byte, snapshotChunkSize)
	for {
		n, err := r.Read(readBuf)
		if n > 0 {
			if _, werr := enc.Write(readBuf[:n]); werr != nil {
				return nil, 0, 0, werr
			}
			total += int64(n)
		}
		if buf.Len() >= snapshotChunkSize {
			if err := flush(); err != nil {
				return nil, 0, 0, err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, 0, err
		}
	}
	if err := flush(); err != nil {
		return nil, 0, 0, err
	}
	return manifest, total, dedupSaved, nil
}

func (c *Core) RestoreVolumeSnapshot(ctx context.Context, snapID, volume string) error {
	snap := c.findSnapshot(snapID)
	if snap == nil {
		return fmt.Errorf("snapshot não encontrado")
	}
	manifestPath := filepath.Join(c.chunkDir(), snap.ID+".json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	var hashes []string
	if err := json.Unmarshal(raw, &hashes); err != nil {
		return err
	}
	var tarData bytes.Buffer
	for _, h := range hashes {
		data, err := os.ReadFile(filepath.Join(c.chunkDir(), h+".zst"))
		if err != nil {
			return err
		}
		zr, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			return err
		}
		if _, err := io.Copy(&tarData, zr); err != nil {
			return err
		}
		zr.Close()
	}
	helper := "aether-restore-" + strings.ReplaceAll(volume, "/", "-")
	c.Driver.Remove(ctx, helper, true)
	id, err := c.Driver.Run(ctx, runtime.RunSpec{
		Name:    helper,
		Image:   "busybox:latest",
		Cmd:     []string{"sleep", "3600"},
		Volumes: []runtime.VolumeMount{{Source: volume, Target: "/data"}},
		Restart: "no",
		Labels:  map[string]string{"aether.role": "snapshot"},
	})
	if err != nil {
		return err
	}
	defer c.Driver.Remove(ctx, id, true)
	stream, err := c.Driver.ExecStream(ctx, id, runtime.ExecRequest{
		Command: []string{"tar", "-xf", "-", "-C", "/data"},
	})
	if err != nil {
		return err
	}
	if _, err := stream.Write(tarData.Bytes()); err != nil {
		return err
	}
	stream.Close()
	if code, err := stream.Wait(); err != nil || code != 0 {
		return fmt.Errorf("restore exit %d: %v", code, err)
	}
	return nil
}

func (c *Core) findSnapshot(id string) *domain.Snapshot {
	orgs, _ := c.Store.ListOrgs()
	for _, o := range orgs {
		list, _ := c.Store.ListSnapshots(o.ID)
		for _, sn := range list {
			if sn.ID == id {
				return &sn
			}
		}
	}
	return nil
}
