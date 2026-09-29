package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/oauth2"

	"aether/internal/platform/storage/gdrive"
	"aether/internal/platform/storage/s3gateway"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Google Drive S3 gateway stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	accessToken := strings.TrimSpace(os.Getenv("GOOGLE_ACCESS_TOKEN"))
	refreshToken := strings.TrimSpace(os.Getenv("GOOGLE_REFRESH_TOKEN"))
	rootFolder := strings.TrimSpace(os.Getenv("GOOGLE_DRIVE_ROOT_FOLDER"))
	dataDir := strings.TrimSpace(os.Getenv("GDRIVE_S3_DATA_DIR"))
	accessKey := strings.TrimSpace(os.Getenv("S3_ACCESS_KEY_ID"))
	secretKey := strings.TrimSpace(os.Getenv("S3_SECRET_ACCESS_KEY"))
	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	if clientID == "" || clientSecret == "" || accessToken == "" || refreshToken == "" || rootFolder == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return errors.New("required Google Drive S3 gateway configuration is missing")
	}
	if dataDir == "" {
		dataDir = "/var/lib/aether-gdrive-s3"
	}
	tempDir := strings.TrimSpace(os.Getenv("GDRIVE_S3_TEMP_DIR"))
	if tempDir == "" {
		tempDir = filepath.Join(dataDir, "tmp")
	}
	multipartDir := strings.TrimSpace(os.Getenv("GDRIVE_S3_MULTIPART_DIR"))
	if multipartDir == "" {
		multipartDir = filepath.Join(dataDir, "uploads")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     oauth2.Endpoint{TokenURL: "https://oauth2.googleapis.com/token"},
	}
	transport := &googleDriveTokenTransport{
		next:         http.DefaultTransport,
		oauthConfig:  oauthConfig,
		accessToken:  accessToken,
		refreshToken: refreshToken,
	}
	provider, err := gdrive.NewProvider(gdrive.Config{
		Client:         &http.Client{Transport: transport},
		RootFolderID:   "root",
		RootFolderName: rootFolder,
	})
	if err != nil {
		return err
	}
	maxUploadSize, err := configuredMaxUpload()
	if err != nil {
		return err
	}
	maxMultipartPartSize, err := configuredMaxMultipartPart()
	if err != nil {
		return err
	}
	gateway, err := s3gateway.New(s3gateway.Config{
		Provider:               provider,
		AccessKey:              accessKey,
		SecretKey:              secretKey,
		Bucket:                 bucket,
		MaxUploadSize:          maxUploadSize,
		MaxMultipartPartSize:   maxMultipartPartSize,
		MaxMultipartObjectSize: 5 << 40,
		MultipartDir:           multipartDir,
		TempDir:                tempDir,
	})
	if err != nil {
		return err
	}
	address := strings.TrimSpace(os.Getenv("GDRIVE_S3_ADDR"))
	if address == "" {
		address = "0.0.0.0:9000"
	}
	server := &http.Server{
		Addr:              address,
		Handler:           gateway,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownContext)
	}
}

func configuredMaxUpload() (int64, error) {
	value := strings.TrimSpace(os.Getenv("GDRIVE_S3_MAX_UPLOAD_BYTES"))
	if value == "" {
		return 5 << 30, nil
	}
	limit, err := strconv.ParseInt(value, 10, 64)
	if err != nil || limit <= 0 {
		return 0, errors.New("GDRIVE_S3_MAX_UPLOAD_BYTES must be a positive integer")
	}
	return limit, nil
}

func configuredMaxMultipartPart() (int64, error) {
	value := strings.TrimSpace(os.Getenv("GDRIVE_S3_MAX_PART_BYTES"))
	if value == "" {
		return 5 << 30, nil
	}
	limit, err := strconv.ParseInt(value, 10, 64)
	if err != nil || limit <= 0 {
		return 0, errors.New("GDRIVE_S3_MAX_PART_BYTES must be a positive integer")
	}
	return limit, nil
}
