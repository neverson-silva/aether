package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aether/internal/domain"
	"aether/internal/runtime"
)

func (c *Core) CreateS3Destination(orgID, name, endpoint, bucket, region, accessKey, secretKey string) (*domain.S3Destination, error) {
	ak, err := c.Secrets.EncryptString(accessKey)
	if err != nil {
		return nil, err
	}
	sk, err := c.Secrets.EncryptString(secretKey)
	if err != nil {
		return nil, err
	}
	d := &domain.S3Destination{
		ID:           domain.NewID(),
		OrgID:        orgID,
		Name:         name,
		Endpoint:     strings.TrimSuffix(endpoint, "/"),
		Bucket:       bucket,
		Region:       region,
		AccessKeyEnc: ak,
		SecretKeyEnc: sk,
		CreatedAt:    time.Now().UTC(),
	}
	if err := c.Store.CreateS3Destination(d); err != nil {
		return nil, err
	}
	return d, nil
}

func (c *Core) s3creds(d *domain.S3Destination) (string, string, error) {
	ak, err := c.Secrets.DecryptString(d.AccessKeyEnc)
	if err != nil {
		return "", "", err
	}
	sk, err := c.Secrets.DecryptString(d.SecretKeyEnc)
	if err != nil {
		return "", "", err
	}
	return ak, sk, nil
}

func (c *Core) UploadToS3(dest *domain.S3Destination, key string, data []byte) error {
	ak, sk, err := c.s3creds(dest)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	body := strings.NewReader(string(data))
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/%s/%s", dest.Endpoint, dest.Bucket, key), body)
	if err != nil {
		return err
	}
	req.Header.Set("x-amz-content-sha256", sha256hex(data))
	req.Header.Set("x-amz-date", now.Format("20060102T150405Z"))
	req.Header.Set("Content-Type", "application/octet-stream")
	signS3(req, ak, sk, dest.Region, now)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload falhou: status %d", resp.StatusCode)
	}
	return nil
}

func sha256hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func signS3(req *http.Request, ak, sk, region string, now time.Time) {
	amzDate := now.Format("20060102T150405Z")
	shortDate := now.Format("20060102")
	payloadHash := req.Header.Get("x-amz-content-sha256")
	canonicalHeaders := "host:" + req.Host + "\n" + "x-amz-content-sha256:" + payloadHash + "\n" + "x-amz-date:" + amzDate + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	scope := shortDate + "/" + region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256hex([]byte(canonicalRequest))
	kDate := hmacSHA([]byte("AWS4"+sk), shortDate)
	kRegion := hmacSHA(kDate, region)
	kService := hmacSHA(kRegion, "s3")
	kSigning := hmacSHA(kService, "aws4_request")
	signature := hex.EncodeToString(hmacSHA(kSigning, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", ak, scope, signedHeaders, signature))
}

func hmacSHA(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func (c *Core) BackupVolumeToDestination(appID, volumeName, destID string) (*domain.Backup, error) {
	app, err := c.Store.GetApp(appID)
	if err != nil {
		return nil, err
	}
	dest, err := c.Store.GetS3Destination(destID)
	if err != nil {
		return nil, err
	}
	volName := volumeName
	if !strings.Contains(volumeName, "aether-") {
		volName = "aether-" + app.Name + "-" + volumeName
	}
	ctx, cancel := timeoutCtx(30 * time.Minute)
	defer cancel()
	containerName := "aether-volbackup-" + app.Name
	c.Driver.Remove(ctx, containerName, true)
	spec := runtime.RunSpec{
		Name:    containerName,
		Image:   "alpine:3.20",
		Cmd:     []string{"sh", "-c", "tar czf - -C /data ."},
		Volumes: []runtime.VolumeMount{{Source: volName, Target: "/data"}},
		Restart: "no",
	}
	id, err := c.Driver.Run(ctx, spec)
	if err != nil {
		return nil, err
	}
	defer c.Driver.Remove(ctx, id, true)
	stream, err := c.Driver.Logs(ctx, id, false)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("volumes/%s/%s-%s.tar.gz", app.Name, volumeName, time.Now().UTC().Format("20060102T150405"))
	if err := c.UploadToS3(dest, key, data); err != nil {
		return nil, err
	}
	dir := filepath.Join(c.Cfg.StateDir, "backups")
	os.MkdirAll(dir, 0o750)
	localPath := filepath.Join(dir, strings.ReplaceAll(key, "/", "_"))
	if err := os.WriteFile(localPath, data, 0o640); err != nil {
		return nil, err
	}
	info, _ := os.Stat(localPath)
	b := &domain.Backup{
		ID:        domain.NewID(),
		Path:      localPath,
		Size:      info.Size(),
		CreatedAt: time.Now().UTC(),
	}
	if err := c.createBackupRecord(b, "volume", dest.Name, appID); err != nil {
		return nil, err
	}
	return b, nil
}

func timeoutCtx(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
