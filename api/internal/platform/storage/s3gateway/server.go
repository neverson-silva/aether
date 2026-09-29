package s3gateway

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"aether/internal/platform/storage"
)

const (
	healthPath       = "/healthz"
	folderMarkerName = ".aether-gdrive-folder"
	defaultMaxUpload = 5 << 30
)

type FolderProvider interface {
	EnsureFolder(context.Context, string) error
}

type Config struct {
	Provider               storage.Provider
	AccessKey              string
	SecretKey              string
	Bucket                 string
	MaxUploadSize          int64
	MaxMultipartPartSize   int64
	MaxMultipartObjectSize int64
	MultipartDir           string
	TempDir                string
}

type Server struct {
	provider      storage.Provider
	folders       FolderProvider
	accessKey     string
	secretKey     string
	bucket        string
	maxUploadSize int64
	tempDir       string
	multipart     *multipartStore
}

func New(config Config) (*Server, error) {
	if config.Provider == nil || strings.TrimSpace(config.AccessKey) == "" || strings.TrimSpace(config.SecretKey) == "" || !validBucket(config.Bucket) {
		return nil, storage.ErrInvalidConfig
	}
	maxUploadSize := config.MaxUploadSize
	if maxUploadSize <= 0 {
		maxUploadSize = defaultMaxUpload
	}
	tempDir := strings.TrimSpace(config.TempDir)
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	multipartDir := strings.TrimSpace(config.MultipartDir)
	if multipartDir == "" {
		multipartDir = filepath.Join(tempDir, "aether-gdrive-s3-multipart")
	}
	maxPartSize := config.MaxMultipartPartSize
	if maxPartSize <= 0 {
		maxPartSize = 5 << 30
	}
	multipart, err := newMultipartStore(multipartDir, tempDir, maxPartSize, config.MaxMultipartObjectSize)
	if err != nil {
		return nil, err
	}
	server := &Server{
		provider:      config.Provider,
		accessKey:     config.AccessKey,
		secretKey:     config.SecretKey,
		bucket:        config.Bucket,
		maxUploadSize: maxUploadSize,
		tempDir:       tempDir,
		multipart:     multipart,
	}
	if folders, ok := config.Provider.(FolderProvider); ok {
		server.folders = folders
	}
	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == healthPath && r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
		return
	}
	payloadHash, err := s.authenticate(r)
	if err != nil {
		s.writeError(w, http.StatusForbidden, "SignatureDoesNotMatch", "The request signature is invalid.", r.URL.Path)
		return
	}
	if r.Method == http.MethodPut || r.Method == http.MethodPost {
		maxBodySize := s.maxUploadSize
		if r.Method == http.MethodPost {
			maxBodySize = maxCompleteRequestSize
		}
		if r.Method == http.MethodPut {
			if _, isPart := r.URL.Query()["uploadId"]; isPart {
				maxBodySize = s.multipart.maxPartSize
			}
		}
		if err := s.prepareBody(r, payloadHash, maxBodySize); err != nil {
			var tooLarge *uploadTooLargeError
			if errors.As(err, &tooLarge) {
				s.writeError(w, http.StatusRequestEntityTooLarge, "EntityTooLarge", "The uploaded object exceeds the configured limit.", r.URL.Path)
				return
			}
			s.writeError(w, http.StatusBadRequest, "XAmzContentSHA256Mismatch", "The payload hash does not match the request body.", r.URL.Path)
			return
		}
		defer r.Body.Close()
	}
	s.route(w, r)
}

func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	objectPath := strings.TrimPrefix(r.URL.Path, "/")
	if objectPath == "" {
		if r.Method == http.MethodGet {
			s.writeBuckets(w)
			return
		}
		s.writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "The requested operation is not supported.", r.URL.Path)
		return
	}
	bucket, key, found := strings.Cut(objectPath, "/")
	if bucket != s.bucket {
		s.writeError(w, http.StatusNotFound, "NoSuchBucket", "The requested bucket does not exist.", r.URL.Path)
		return
	}
	if !found || key == "" {
		if r.Method == http.MethodGet {
			if _, uploads := r.URL.Query()["uploads"]; uploads {
				s.listMultipartUploads(w, r)
				return
			}
		}
		s.routeBucket(w, r)
		return
	}
	s.routeObject(w, r, key)
}

func (s *Server) routeBucket(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		w.WriteHeader(http.StatusOK)
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		if _, exists := r.URL.Query()["location"]; exists {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, xml.Header+"<LocationConstraint xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"></LocationConstraint>")
			return
		}
		s.listObjects(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "The requested operation is not supported.", r.URL.Path)
	}
}

func (s *Server) routeObject(w http.ResponseWriter, r *http.Request, key string) {
	if s.routeMultipartObject(w, r, key) {
		return
	}
	if r.Method == http.MethodDelete && strings.HasSuffix(key, "/") {
		key += folderMarkerName
	}
	if strings.Contains(key, folderMarkerName) {
		s.writeError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", r.URL.Path)
		return
	}
	switch r.Method {
	case http.MethodPut:
		s.putObject(w, r, key)
	case http.MethodGet:
		s.getObject(w, r, key)
	case http.MethodHead:
		s.headObject(w, r, key)
	case http.MethodDelete:
		if err := s.provider.DeleteObject(r.Context(), storage.DeleteObjectInput{Key: key}); err != nil {
			s.writeProviderError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "The requested operation is not supported.", r.URL.Path)
	}
}

func (s *Server) putObject(w http.ResponseWriter, r *http.Request, key string) {
	if strings.HasSuffix(key, "/") {
		if s.folders == nil {
			s.writeError(w, http.StatusNotImplemented, "NotImplemented", "Folder creation is not supported by this provider.", r.URL.Path)
			return
		}
		folder := strings.TrimSuffix(key, "/")
		if err := s.folders.EnsureFolder(r.Context(), folder); err != nil {
			s.writeProviderError(w, r, err)
			return
		}
		key = folder + "/" + folderMarkerName
	}
	contentType := r.Header.Get("Content-Type")
	requestChecksum, err := checksumFromHeaders(r.Header, "")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", err.Error(), r.URL.Path)
		return
	}
	if requestChecksum.Algorithm != "" && requestChecksum.Value == "" {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", "The requested checksum value is missing.", r.URL.Path)
		return
	}
	metadata := make(map[string]string)
	for name, values := range r.Header {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "x-amz-meta-") && len(values) > 0 {
			metadata[strings.TrimPrefix(lower, "x-amz-meta-")] = strings.Join(values, ",")
		}
	}
	seekableBody, ok := r.Body.(interface {
		io.Reader
		io.Seeker
	})
	if !ok {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", "The uploaded object could not be read.", r.URL.Path)
		return
	}
	md5Hash := md5.New()
	writers := []io.Writer{md5Hash}
	var checksumHash interface {
		Write([]byte) (int, error)
		Sum([]byte) []byte
	}
	if requestChecksum.Algorithm != "" {
		checksumHash, err = newChecksumHash(requestChecksum.Algorithm)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "InvalidRequest", err.Error(), r.URL.Path)
			return
		}
		writers = append(writers, checksumHash)
	}
	if _, err := io.Copy(io.MultiWriter(writers...), seekableBody); err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", "The uploaded object could not be read.", r.URL.Path)
		return
	}
	if _, err := seekableBody.Seek(0, io.SeekStart); err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", "The uploaded object could not be read.", r.URL.Path)
		return
	}
	md5Digest := md5Hash.Sum(nil)
	if contentMD5 := r.Header.Get("Content-MD5"); contentMD5 != "" {
		expected, decodeErr := base64.StdEncoding.DecodeString(contentMD5)
		if decodeErr != nil || !equalBytes(expected, md5Digest) {
			s.writeError(w, http.StatusBadRequest, "BadDigest", "The Content-MD5 value does not match the uploaded object.", r.URL.Path)
			return
		}
	}
	etag := hex.EncodeToString(md5Digest)
	metadata[storage.ObjectETagMetadataKey] = etag
	checksum := ""
	if checksumHash != nil {
		checksum = encodedChecksum(checksumHash.Sum(nil))
		if !checksumMatches(requestChecksum.Value, checksumHash.Sum(nil)) {
			s.writeError(w, http.StatusBadRequest, "BadDigest", "The checksum does not match the uploaded object.", r.URL.Path)
			return
		}
		metadata[checksumMetadataKey(requestChecksum.Algorithm)] = checksum
		metadata[storage.ObjectChecksumTypeMetadataKey] = checksumTypeFullObject
	}
	output, err := s.provider.PutObject(r.Context(), storage.PutObjectInput{Key: key, Body: r.Body, ContentType: contentType, Metadata: metadata})
	if err != nil {
		s.writeProviderError(w, r, err)
		return
	}
	if output.ETag != "" {
		etag = output.ETag
	}
	w.Header().Set("ETag", `"`+strings.Trim(etag, `"`)+`"`)
	if checksum != "" {
		w.Header().Set(checksumHeaderName(requestChecksum.Algorithm), checksum)
	}
	w.Header().Set("Content-Length", "0")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getObject(w http.ResponseWriter, r *http.Request, key string) {
	output, err := s.provider.GetObject(r.Context(), storage.GetObjectInput{Key: key})
	if err != nil {
		s.writeProviderError(w, r, err)
		return
	}
	defer output.Body.Close()
	s.writeObjectHeaders(w, output.ContentType, output.ContentLength, output.LastModified, output.ETag, output.Metadata)
	if _, err := io.Copy(w, output.Body); err != nil {
		return
	}
}

func (s *Server) headObject(w http.ResponseWriter, r *http.Request, key string) {
	output, err := s.provider.HeadObject(r.Context(), storage.HeadObjectInput{Key: key})
	if err != nil {
		s.writeProviderError(w, r, err)
		return
	}
	s.writeObjectHeaders(w, output.ContentType, output.ContentLength, output.LastModified, output.ETag, output.Metadata)
}

func (s *Server) writeObjectHeaders(w http.ResponseWriter, contentType string, contentLength int64, modified time.Time, etag string, metadata map[string]string) {
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if contentLength >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	}
	if !modified.IsZero() {
		w.Header().Set("Last-Modified", modified.UTC().Format(http.TimeFormat))
	}
	if etag != "" {
		w.Header().Set("ETag", `"`+strings.Trim(etag, `"`)+`"`)
	}
	for name, value := range metadata {
		if name == storage.ObjectETagMetadataKey {
			continue
		}
		if name == storage.ObjectChecksumTypeMetadataKey {
			w.Header().Set("x-amz-checksum-type", value)
			continue
		}
		if strings.HasPrefix(name, storage.ObjectChecksumMetadataPrefix) {
			algorithm := strings.TrimPrefix(name, storage.ObjectChecksumMetadataPrefix)
			if header := checksumHeaderName(strings.ToUpper(algorithm)); header != "" {
				w.Header().Set(header, value)
				continue
			}
		}
		w.Header().Set("x-amz-meta-"+name, value)
	}
}

func (s *Server) listObjects(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit := 1000
	if raw := query.Get("max-keys"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			s.writeError(w, http.StatusBadRequest, "InvalidArgument", "max-keys must be a non-negative integer.", r.URL.Path)
			return
		}
		limit = min(parsed, 1000)
	}
	if limit == 0 {
		s.writeXML(w, http.StatusOK, listResult{
			XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Name: s.bucket,
			Prefix: query.Get("prefix"), Delimiter: query.Get("delimiter"), MaxKeys: 0,
			KeyCount: 0, Contents: []listedObject{}, CommonPrefixes: []listedPrefix{},
		})
		return
	}
	output, err := s.provider.ListObjects(r.Context(), storage.ListObjectsInput{
		Prefix: query.Get("prefix"), Limit: limit, Cursor: query.Get("continuation-token"),
	})
	if err != nil {
		s.writeProviderError(w, r, err)
		return
	}
	objects := normalizeFolderMarkers(output.Objects)
	sort.Slice(objects, func(i, j int) bool { return objects[i].Key < objects[j].Key })
	contents := make([]listedObject, 0, len(objects))
	prefixes := make([]listedPrefix, 0)
	seenPrefixes := make(map[string]struct{})
	delimiter := query.Get("delimiter")
	for _, object := range objects {
		remaining := strings.TrimPrefix(object.Key, query.Get("prefix"))
		if delimiter != "" {
			if index := strings.Index(remaining, delimiter); index >= 0 {
				commonPrefix := object.Key[:len(object.Key)-len(remaining)+index+len(delimiter)]
				if _, seen := seenPrefixes[commonPrefix]; !seen {
					seenPrefixes[commonPrefix] = struct{}{}
					prefixes = append(prefixes, listedPrefix{Prefix: commonPrefix})
				}
				continue
			}
		}
		contents = append(contents, listedObject{
			Key: object.Key, LastModified: object.LastModified.UTC(), Size: object.Size,
			ETag: quotedETag(object.ETag), StorageClass: "STANDARD",
		})
	}
	isV2 := query.Get("list-type") == "2"
	result := listResult{
		XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/",
		Name:  s.bucket, Prefix: query.Get("prefix"), Delimiter: delimiter, MaxKeys: limit,
		KeyCount: len(contents) + len(prefixes), IsTruncated: output.NextCursor != "",
		Contents: contents, CommonPrefixes: prefixes,
	}
	if isV2 {
		result.ContinuationToken = query.Get("continuation-token")
		if output.NextCursor != "" {
			result.NextContinuationToken = output.NextCursor
		}
	} else {
		result.Marker = query.Get("marker")
		if len(contents) > 0 {
			result.NextMarker = contents[len(contents)-1].Key
		} else if len(prefixes) > 0 {
			result.NextMarker = prefixes[len(prefixes)-1].Prefix
		}
	}
	s.writeXML(w, http.StatusOK, result)
}

func normalizeFolderMarkers(objects []storage.ObjectInfo) []storage.ObjectInfo {
	result := make([]storage.ObjectInfo, 0, len(objects))
	seen := make(map[string]struct{}, len(objects))
	for _, object := range objects {
		if strings.HasSuffix(object.Key, "/"+folderMarkerName) {
			object.Key = strings.TrimSuffix(object.Key, folderMarkerName)
			object.Size = 0
			object.ContentType = "application/x-directory"
		}
		if _, exists := seen[object.Key]; exists {
			continue
		}
		seen[object.Key] = struct{}{}
		result = append(result, object)
	}
	return result
}

func (s *Server) writeBuckets(w http.ResponseWriter) {
	s.writeXML(w, http.StatusOK, bucketList{
		XMLNS:   "http://s3.amazonaws.com/doc/2006-03-01/",
		Owner:   owner{ID: "aether-gdrive", DisplayName: "Aether Google Drive"},
		Buckets: []bucket{{Name: s.bucket, CreationDate: time.Now().UTC()}},
	})
}

func (s *Server) authenticate(r *http.Request) (string, error) {
	if r.Header.Get("x-amz-content-sha256") == "" || r.Header.Get("x-amz-date") == "" {
		return "", errors.New("required signature headers are missing")
	}
	if strings.HasPrefix(r.Header.Get("x-amz-content-sha256"), "STREAMING-") {
		return "", errors.New("streaming signatures are not supported")
	}
	return verifySignature(r, s.accessKey, s.secretKey)
}

func (s *Server) prepareBody(r *http.Request, payloadHash string, maxSize int64) error {
	var expected []byte
	if payloadHash == "UNSIGNED-PAYLOAD" {
		if !requestUsesHTTPS(r) {
			return errors.New("unsigned payloads require HTTPS")
		}
	} else {
		var err error
		expected, err = hex.DecodeString(payloadHash)
		if err != nil || len(expected) != sha256.Size {
			return errors.New("invalid payload hash")
		}
	}
	if r.ContentLength > maxSize {
		return &uploadTooLargeError{}
	}
	file, err := os.CreateTemp(s.tempDir, "aether-gdrive-s3-upload-*")
	if err != nil {
		return err
	}
	name := file.Name()
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(r.Body, maxSize+1))
	if copyErr != nil {
		file.Close()
		os.Remove(name)
		return copyErr
	}
	if written > maxSize {
		file.Close()
		os.Remove(name)
		return &uploadTooLargeError{}
	}
	if expected != nil && subtle.ConstantTimeCompare(expected, hash.Sum(nil)) != 1 {
		file.Close()
		os.Remove(name)
		return errors.New("payload hash mismatch")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		os.Remove(name)
		return err
	}
	r.Body = &temporaryBody{File: file, name: name}
	r.ContentLength = written
	return nil
}

func (s *Server) writeProviderError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, storage.ErrObjectNotFound) {
		s.writeError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", r.URL.Path)
		return
	}
	if errors.Is(err, storage.ErrInvalidObjectKey) {
		s.writeError(w, http.StatusBadRequest, "InvalidArgument", "The requested object key is invalid.", r.URL.Path)
		return
	}
	if errors.Is(err, storage.ErrPermissionDenied) || errors.Is(err, storage.ErrAuthentication) {
		s.writeError(w, http.StatusForbidden, "AccessDenied", "The storage provider rejected this request.", r.URL.Path)
		return
	}
	s.writeError(w, http.StatusBadGateway, "StorageError", "The Google Drive storage request failed.", r.URL.Path)
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message, resource string) {
	s.writeXML(w, status, errorResponse{Code: code, Message: message, Resource: resource})
}

func (s *Server) writeXML(w http.ResponseWriter, status int, value any) {
	encoded, err := xml.Marshal(value)
	if err != nil {
		http.Error(w, "response encoding failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(encoded)
}

func validBucket(value string) bool {
	if len(value) < 3 || len(value) > 63 || strings.Contains(value, "..") || strings.Contains(value, ".-") || strings.Contains(value, "-.") {
		return false
	}
	for _, character := range value {
		if !((character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '.' || character == '-') {
			return false
		}
	}
	return value[0] != '.' && value[0] != '-' && value[len(value)-1] != '.' && value[len(value)-1] != '-'
}

func quotedETag(value string) string {
	if value == "" {
		return ""
	}
	return `"` + strings.Trim(value, `"`) + `"`
}

type uploadTooLargeError struct{}

func (*uploadTooLargeError) Error() string { return "upload exceeds configured limit" }

type temporaryBody struct {
	*os.File
	name string
}

func (b *temporaryBody) Close() error {
	closeErr := b.File.Close()
	removeErr := os.Remove(b.name)
	if errors.Is(removeErr, os.ErrNotExist) {
		removeErr = nil
	}
	return errors.Join(closeErr, removeErr)
}

type errorResponse struct {
	XMLName  xml.Name `xml:"Error"`
	Code     string   `xml:"Code"`
	Message  string   `xml:"Message"`
	Resource string   `xml:"Resource"`
}

type owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type bucket struct {
	Name         string    `xml:"Name"`
	CreationDate time.Time `xml:"CreationDate"`
}

type bucketList struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	XMLNS   string   `xml:"xmlns,attr"`
	Owner   owner    `xml:"Owner"`
	Buckets []bucket `xml:"Buckets>Bucket"`
}

type listedObject struct {
	Key          string    `xml:"Key"`
	LastModified time.Time `xml:"LastModified"`
	ETag         string    `xml:"ETag,omitempty"`
	Size         int64     `xml:"Size"`
	StorageClass string    `xml:"StorageClass"`
}

type listedPrefix struct {
	Prefix string `xml:"Prefix"`
}

type listResult struct {
	XMLName               xml.Name       `xml:"ListBucketResult"`
	XMLNS                 string         `xml:"xmlns,attr"`
	Name                  string         `xml:"Name"`
	Prefix                string         `xml:"Prefix"`
	Delimiter             string         `xml:"Delimiter,omitempty"`
	Marker                string         `xml:"Marker,omitempty"`
	NextMarker            string         `xml:"NextMarker,omitempty"`
	MaxKeys               int            `xml:"MaxKeys"`
	KeyCount              int            `xml:"KeyCount"`
	IsTruncated           bool           `xml:"IsTruncated"`
	ContinuationToken     string         `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string         `xml:"NextContinuationToken,omitempty"`
	Contents              []listedObject `xml:"Contents"`
	CommonPrefixes        []listedPrefix `xml:"CommonPrefixes,omitempty"`
}
