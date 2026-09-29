package s3gateway

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	minMultipartPartSize   int64 = 5 << 20
	maxMultipartPartSize   int64 = 5 << 30
	maxMultipartObjectSize int64 = 5 << 40
	maxMultipartPartCount        = 10000
	maxMultipartPageSize         = 1000
	maxCompleteRequestSize       = 2 << 20
	defaultMultipartMaxAge       = 7 * 24 * time.Hour
)

type multipartError struct {
	status  int
	code    string
	message string
}

func (e *multipartError) Error() string {
	return e.message
}

type multipartUpload struct {
	ID                string                    `json:"id"`
	Bucket            string                    `json:"bucket"`
	Key               string                    `json:"key"`
	ContentType       string                    `json:"content_type"`
	Metadata          map[string]string         `json:"metadata"`
	ChecksumAlgorithm string                    `json:"checksum_algorithm,omitempty"`
	ChecksumType      string                    `json:"checksum_type,omitempty"`
	Initiated         time.Time                 `json:"initiated"`
	Parts             map[int]multipartPartFile `json:"parts"`
}

type multipartPartFile struct {
	File              string    `json:"file"`
	ETag              string    `json:"etag"`
	ChecksumAlgorithm string    `json:"checksum_algorithm,omitempty"`
	Checksum          string    `json:"checksum,omitempty"`
	Size              int64     `json:"size"`
	LastModified      time.Time `json:"last_modified"`
}

type completedMultipartPart struct {
	PartNumber        int
	ETag              string
	ChecksumAlgorithm string
	Checksum          string
}

type multipartCompletion struct {
	ETag              string
	ChecksumAlgorithm string
	Checksum          string
	ChecksumType      string
}

type multipartSessionLock struct {
	mu   sync.Mutex
	refs int
}

type multipartStore struct {
	root          string
	tempDir       string
	maxPartSize   int64
	maxObjectSize int64
	maxAge        time.Duration
	locksMu       sync.Mutex
	locks         map[string]*multipartSessionLock
	cleanupMu     sync.Mutex
	lastCleanup   time.Time
}

func newMultipartStore(root, tempDir string, maxPartSize, maxObjectSize int64) (*multipartStore, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(tempDir) == "" {
		return nil, errors.New("multipart storage paths are required")
	}
	if maxPartSize <= 0 {
		maxPartSize = maxMultipartPartSize
	} else if maxPartSize > maxMultipartPartSize {
		maxPartSize = maxMultipartPartSize
	}
	if maxObjectSize <= 0 {
		maxObjectSize = maxMultipartObjectSize
	} else if maxObjectSize > maxMultipartObjectSize {
		maxObjectSize = maxMultipartObjectSize
	}
	for _, dir := range []string{root, tempDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("create multipart storage directory: %w", err)
		}
	}
	return &multipartStore{
		root: root, tempDir: tempDir, maxPartSize: maxPartSize, maxObjectSize: maxObjectSize,
		maxAge: defaultMultipartMaxAge, locks: make(map[string]*multipartSessionLock),
	}, nil
}

func (m *multipartStore) initiate(bucket, key, contentType string, metadata map[string]string, checksumAlgorithm, checksumType string) (*multipartUpload, error) {
	m.cleanupExpired()
	id := uuid.NewString()
	dir := filepath.Join(m.root, id)
	if err := os.Mkdir(dir, 0o700); err != nil {
		return nil, err
	}
	upload := &multipartUpload{
		ID: id, Bucket: bucket, Key: key, ContentType: contentType,
		Metadata: cloneStringMap(metadata), ChecksumAlgorithm: checksumAlgorithm, ChecksumType: checksumType,
		Initiated: time.Now().UTC(), Parts: make(map[int]multipartPartFile),
	}
	if err := m.writeManifest(dir, upload); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return upload, nil
}

func (m *multipartStore) uploadPart(uploadID, bucket, key string, number int, body *temporaryBody, contentMD5 string, requestChecksum checksumValue) (multipartPartFile, error) {
	if number < 1 || number > maxMultipartPartCount {
		return multipartPartFile{}, multipartFailure(400, "InvalidArgument", "Part number must be between 1 and 10000.")
	}
	if body == nil {
		return multipartPartFile{}, multipartFailure(400, "InvalidRequest", "A request body is required for each part.")
	}
	_, release, err := m.acquire(uploadID)
	if err != nil {
		return multipartPartFile{}, err
	}
	defer release()
	upload, dir, err := m.readUpload(uploadID)
	if err != nil {
		return multipartPartFile{}, err
	}
	if upload.Bucket != bucket || upload.Key != key {
		return multipartPartFile{}, multipartFailure(404, "NoSuchUpload", "The specified multipart upload does not exist.")
	}
	if requestChecksum.Algorithm != "" && upload.ChecksumAlgorithm != "" && requestChecksum.Algorithm != upload.ChecksumAlgorithm {
		return multipartPartFile{}, multipartFailure(400, "InvalidRequest", "The part checksum algorithm does not match the multipart upload.")
	}
	if upload.ChecksumAlgorithm == "" && requestChecksum.Algorithm != "" {
		upload.ChecksumAlgorithm = requestChecksum.Algorithm
		upload.ChecksumType = checksumTypeComposite
	}
	if upload.ChecksumAlgorithm != "" && upload.ChecksumType == checksumTypeComposite && requestChecksum.Value == "" {
		return multipartPartFile{}, multipartFailure(400, "InvalidRequest", "A checksum value is required for every multipart part.")
	}
	info, err := body.Stat()
	if err != nil {
		return multipartPartFile{}, err
	}
	if info.Size() > m.maxPartSize {
		return multipartPartFile{}, multipartFailure(413, "EntityTooLarge", "The uploaded part exceeds the configured limit.")
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return multipartPartFile{}, err
	}
	md5Hash := md5.New()
	var checksumHash hash.Hash
	if upload.ChecksumAlgorithm != "" {
		checksumHash, err = newChecksumHash(upload.ChecksumAlgorithm)
		if err != nil {
			return multipartPartFile{}, multipartFailure(400, "InvalidRequest", err.Error())
		}
	}
	writers := []io.Writer{md5Hash}
	if checksumHash != nil {
		writers = append(writers, checksumHash)
	}
	if _, err := io.Copy(io.MultiWriter(writers...), body); err != nil {
		return multipartPartFile{}, err
	}
	digest := md5Hash.Sum(nil)
	if contentMD5 != "" {
		expected, err := base64.StdEncoding.DecodeString(contentMD5)
		if err != nil || !equalBytes(digest, expected) {
			return multipartPartFile{}, multipartFailure(400, "BadDigest", "The Content-MD5 value does not match the uploaded part.")
		}
	}
	checksum := ""
	if checksumHash != nil {
		checksumDigest := checksumHash.Sum(nil)
		checksum = encodedChecksum(checksumDigest)
		if requestChecksum.Value != "" && !checksumMatches(requestChecksum.Value, checksumDigest) {
			return multipartPartFile{}, multipartFailure(400, "BadDigest", "The part checksum does not match the uploaded data.")
		}
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return multipartPartFile{}, err
	}
	fileName := fmt.Sprintf("part-%05d-%s", number, uuid.NewString())
	filePath := filepath.Join(dir, fileName)
	if err := body.Sync(); err != nil {
		return multipartPartFile{}, err
	}
	if err := os.Rename(body.name, filePath); err != nil {
		if _, seekErr := body.Seek(0, io.SeekStart); seekErr != nil {
			return multipartPartFile{}, seekErr
		}
		staged, createErr := os.CreateTemp(dir, "part-stage-*")
		if createErr != nil {
			return multipartPartFile{}, createErr
		}
		stagedName := staged.Name()
		written, copyErr := io.Copy(staged, body)
		if copyErr == nil && written != info.Size() {
			copyErr = io.ErrUnexpectedEOF
		}
		if copyErr == nil {
			copyErr = staged.Sync()
		}
		closeErr := staged.Close()
		if copyErr == nil {
			copyErr = closeErr
		}
		if copyErr == nil {
			copyErr = os.Rename(stagedName, filePath)
		}
		if copyErr != nil {
			_ = os.Remove(stagedName)
			return multipartPartFile{}, copyErr
		}
	}
	part := multipartPartFile{
		File: fileName, ETag: hex.EncodeToString(digest),
		ChecksumAlgorithm: upload.ChecksumAlgorithm, Checksum: checksum,
		Size: info.Size(), LastModified: time.Now().UTC(),
	}
	previous := upload.Parts[number]
	upload.Parts[number] = part
	if err := m.writeManifest(dir, upload); err != nil {
		_ = os.Remove(filePath)
		return multipartPartFile{}, err
	}
	if previous.File != "" {
		_ = os.Remove(filepath.Join(dir, previous.File))
	}
	return part, nil
}

func (m *multipartStore) complete(ctx context.Context, uploadID, bucket, key string, requested []completedMultipartPart, requestChecksum checksumValue, requestChecksumType string, commit func(io.Reader, multipartUpload, multipartCompletion) error) (multipartCompletion, error) {
	_, release, err := m.acquire(uploadID)
	if err != nil {
		return multipartCompletion{}, err
	}
	defer release()
	upload, dir, err := m.readUpload(uploadID)
	if err != nil {
		return multipartCompletion{}, err
	}
	if upload.Bucket != bucket || upload.Key != key {
		return multipartCompletion{}, multipartFailure(404, "NoSuchUpload", "The specified multipart upload does not exist.")
	}
	if len(requested) == 0 || len(requested) > maxMultipartPartCount {
		return multipartCompletion{}, multipartFailure(400, "InvalidPart", "The completion request must contain between 1 and 10000 parts.")
	}
	if requestChecksum.Algorithm != "" && upload.ChecksumAlgorithm != "" && requestChecksum.Algorithm != upload.ChecksumAlgorithm {
		return multipartCompletion{}, multipartFailure(400, "InvalidRequest", "The completion checksum algorithm does not match the multipart upload.")
	}
	if requestChecksumType != "" && requestChecksumType != upload.ChecksumType {
		return multipartCompletion{}, multipartFailure(400, "InvalidRequest", "The completion checksum type does not match the multipart upload.")
	}
	var total int64
	partDigests := make([]byte, 0, len(requested)*md5.Size)
	sequence := make([]multipartPartFile, 0, len(requested))
	previousNumber := 0
	for index, requestedPart := range requested {
		if requestedPart.PartNumber <= previousNumber || requestedPart.PartNumber > maxMultipartPartCount {
			return multipartCompletion{}, multipartFailure(400, "InvalidPartOrder", "Parts must be listed once in strictly increasing part-number order.")
		}
		if upload.ChecksumAlgorithm != "" && upload.ChecksumType == checksumTypeComposite && requestedPart.PartNumber != index+1 {
			return multipartCompletion{}, multipartFailure(400, "InvalidPartOrder", "Multipart uploads with composite checksums require consecutive part numbers starting at 1.")
		}
		previousNumber = requestedPart.PartNumber
		stored, exists := upload.Parts[requestedPart.PartNumber]
		if !exists || !strings.EqualFold(strings.Trim(requestedPart.ETag, `"`), stored.ETag) {
			return multipartCompletion{}, multipartFailure(400, "InvalidPart", "A requested part is missing or has an invalid ETag.")
		}
		if requestedPart.Checksum != "" && (requestedPart.ChecksumAlgorithm != stored.ChecksumAlgorithm || !checksumMatches(requestedPart.Checksum, mustDecodeChecksum(stored.Checksum))) {
			return multipartCompletion{}, multipartFailure(400, "InvalidPart", "A requested part checksum is missing or invalid.")
		}
		if upload.ChecksumAlgorithm != "" && upload.ChecksumType == checksumTypeComposite && (stored.Checksum == "" || requestedPart.Checksum == "") {
			return multipartCompletion{}, multipartFailure(400, "InvalidPart", "Every completed part must include its checksum.")
		}
		if index < len(requested)-1 && stored.Size < minMultipartPartSize {
			return multipartCompletion{}, multipartFailure(400, "EntityTooSmall", "Every part except the last must be at least 5 MiB.")
		}
		digest, decodeErr := hex.DecodeString(stored.ETag)
		if decodeErr != nil || len(digest) != md5.Size {
			return multipartCompletion{}, multipartFailure(400, "InvalidPart", "A stored part has an invalid checksum.")
		}
		partDigests = append(partDigests, digest...)
		total += stored.Size
		if total > m.maxObjectSize {
			return multipartCompletion{}, multipartFailure(413, "EntityTooLarge", "The completed object exceeds the Google Drive file-size limit.")
		}
		if filepath.Base(stored.File) != stored.File || !strings.HasPrefix(stored.File, "part-") {
			return multipartCompletion{}, multipartFailure(500, "InternalError", "Multipart upload state is invalid.")
		}
		sequence = append(sequence, stored)
	}
	checksum := md5.Sum(partDigests)
	completion := multipartCompletion{
		ETag:              fmt.Sprintf("%x-%d", checksum, len(sequence)),
		ChecksumAlgorithm: upload.ChecksumAlgorithm,
		ChecksumType:      upload.ChecksumType,
	}
	if upload.ChecksumAlgorithm != "" {
		if upload.ChecksumType == checksumTypeFullObject {
			completion.Checksum, err = multipartContentChecksum(ctx, dir, sequence, upload.ChecksumAlgorithm)
		} else {
			completion.Checksum, err = multipartCompositeChecksum(sequence, upload.ChecksumAlgorithm)
		}
		if err != nil {
			return multipartCompletion{}, err
		}
		if upload.ChecksumType == checksumTypeFullObject && requestChecksum.Value == "" {
			return multipartCompletion{}, multipartFailure(400, "InvalidRequest", "A full-object checksum is required to complete this multipart upload.")
		}
		if requestChecksum.Value != "" && !checksumMatches(requestChecksum.Value, mustDecodeChecksum(completion.Checksum)) {
			return multipartCompletion{}, multipartFailure(400, "BadDigest", "The completed object checksum does not match the uploaded data.")
		}
	}
	reader := &multipartSequenceReader{ctx: ctx, dir: dir, parts: sequence}
	err = commit(reader, *upload, completion)
	closeErr := reader.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return multipartCompletion{}, err
	}
	_ = os.RemoveAll(dir)
	return completion, nil
}

func (m *multipartStore) abort(uploadID, bucket, key string) error {
	_, release, err := m.acquire(uploadID)
	if err != nil {
		return err
	}
	defer release()
	upload, dir, err := m.readUpload(uploadID)
	if err != nil {
		return err
	}
	if upload.Bucket != bucket || upload.Key != key {
		return multipartFailure(404, "NoSuchUpload", "The specified multipart upload does not exist.")
	}
	return os.RemoveAll(dir)
}

func (m *multipartStore) listParts(uploadID, bucket, key string, marker, limit int) (multipartUpload, []multipartPartFile, int, bool, error) {
	_, release, err := m.acquire(uploadID)
	if err != nil {
		return multipartUpload{}, nil, 0, false, err
	}
	defer release()
	upload, _, err := m.readUpload(uploadID)
	if err != nil {
		return multipartUpload{}, nil, 0, false, err
	}
	if upload.Bucket != bucket || upload.Key != key {
		return multipartUpload{}, nil, 0, false, multipartFailure(404, "NoSuchUpload", "The specified multipart upload does not exist.")
	}
	parts := make([]multipartPartFile, 0, len(upload.Parts))
	for number, part := range upload.Parts {
		if number > marker {
			parts = append(parts, part)
		}
	}
	sort.Slice(parts, func(i, j int) bool {
		return partNumberFromName(parts[i].File) < partNumberFromName(parts[j].File)
	})
	truncated := len(parts) > limit
	nextMarker := marker
	if truncated && limit == 0 {
		nextMarker = partNumberFromName(parts[0].File)
	}
	if truncated {
		parts = parts[:limit]
	}
	if len(parts) > 0 {
		nextMarker = partNumberFromName(parts[len(parts)-1].File)
	}
	return *upload, parts, nextMarker, truncated, nil
}

func (m *multipartStore) listUploads(bucket, prefix, keyMarker, uploadIDMarker string, limit int) ([]multipartUpload, bool, string, string, error) {
	m.cleanupExpired()
	entries, err := os.ReadDir(m.root)
	if err != nil {
		return nil, false, "", "", err
	}
	uploads := make([]multipartUpload, 0)
	for _, entry := range entries {
		if !entry.IsDir() || !validMultipartUploadID(entry.Name()) {
			continue
		}
		_, release, err := m.acquire(entry.Name())
		if err != nil {
			continue
		}
		upload, _, readErr := m.readUpload(entry.Name())
		release()
		if readErr != nil || upload.Bucket != bucket || !strings.HasPrefix(upload.Key, prefix) {
			continue
		}
		if keyMarker != "" && (upload.Key < keyMarker || upload.Key == keyMarker && (uploadIDMarker == "" || upload.ID <= uploadIDMarker)) {
			continue
		}
		uploads = append(uploads, cloneMultipartUpload(*upload))
	}
	sort.Slice(uploads, func(i, j int) bool {
		if uploads[i].Key == uploads[j].Key {
			return uploads[i].ID < uploads[j].ID
		}
		return uploads[i].Key < uploads[j].Key
	})
	truncated := len(uploads) > limit
	nextKeyMarker, nextUploadIDMarker := "", ""
	if truncated && limit == 0 {
		nextKeyMarker, nextUploadIDMarker = uploads[0].Key, uploads[0].ID
	}
	if truncated {
		uploads = uploads[:limit]
	}
	if truncated && len(uploads) > 0 {
		last := uploads[len(uploads)-1]
		nextKeyMarker, nextUploadIDMarker = last.Key, last.ID
	}
	return uploads, truncated, nextKeyMarker, nextUploadIDMarker, nil
}

func (m *multipartStore) cleanupExpired() {
	now := time.Now().UTC()
	m.cleanupMu.Lock()
	if !m.lastCleanup.IsZero() && now.Sub(m.lastCleanup) < time.Hour {
		m.cleanupMu.Unlock()
		return
	}
	m.lastCleanup = now
	m.cleanupMu.Unlock()
	entries, err := os.ReadDir(m.root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !validMultipartUploadID(entry.Name()) {
			continue
		}
		_, release, err := m.acquire(entry.Name())
		if err != nil {
			continue
		}
		upload, dir, readErr := m.readUpload(entry.Name())
		if readErr == nil && now.Sub(upload.Initiated) > m.maxAge {
			_ = os.RemoveAll(dir)
		}
		release()
	}
}

func (m *multipartStore) acquire(uploadID string) (*sync.Mutex, func(), error) {
	if !validMultipartUploadID(uploadID) {
		return nil, nil, multipartFailure(404, "NoSuchUpload", "The specified multipart upload does not exist.")
	}
	m.locksMu.Lock()
	entry := m.locks[uploadID]
	if entry == nil {
		entry = &multipartSessionLock{}
		m.locks[uploadID] = entry
	}
	entry.refs++
	m.locksMu.Unlock()
	entry.mu.Lock()
	return &entry.mu, func() {
		entry.mu.Unlock()
		m.locksMu.Lock()
		entry.refs--
		if entry.refs == 0 && m.locks[uploadID] == entry {
			delete(m.locks, uploadID)
		}
		m.locksMu.Unlock()
	}, nil
}

func (m *multipartStore) readUpload(uploadID string) (*multipartUpload, string, error) {
	dir := filepath.Join(m.root, uploadID)
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", multipartFailure(404, "NoSuchUpload", "The specified multipart upload does not exist.")
		}
		return nil, "", err
	}
	var upload multipartUpload
	if err := json.Unmarshal(data, &upload); err != nil || upload.ID != uploadID || upload.Parts == nil {
		return nil, "", multipartFailure(500, "InternalError", "Multipart upload state is invalid.")
	}
	return &upload, dir, nil
}

func (m *multipartStore) writeManifest(dir string, upload *multipartUpload) error {
	file, err := os.CreateTemp(dir, "manifest-stage-*")
	if err != nil {
		return err
	}
	name := file.Name()
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		os.Remove(name)
		return err
	}
	if err := json.NewEncoder(file).Encode(upload); err != nil {
		file.Close()
		os.Remove(name)
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(name)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, filepath.Join(dir, "manifest.json")); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

func validMultipartUploadID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id.String() == value
}

func multipartFailure(status int, code, message string) error {
	return &multipartError{status: status, code: code, message: message}
}

func partNumberFromName(value string) int {
	name := strings.TrimPrefix(value, "part-")
	partNumber, _ := strconv.Atoi(strings.SplitN(name, "-", 2)[0])
	return partNumber
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}

func cloneMultipartUpload(source multipartUpload) multipartUpload {
	copy := source
	copy.Metadata = cloneStringMap(source.Metadata)
	copy.Parts = make(map[int]multipartPartFile, len(source.Parts))
	for number, part := range source.Parts {
		copy.Parts[number] = part
	}
	return copy
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func multipartContentChecksum(ctx context.Context, dir string, parts []multipartPartFile, algorithm string) (string, error) {
	checksum, err := newChecksumHash(algorithm)
	if err != nil {
		return "", err
	}
	reader := &multipartSequenceReader{ctx: ctx, dir: dir, parts: parts}
	_, copyErr := io.Copy(checksum, reader)
	closeErr := reader.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return encodedChecksum(checksum.Sum(nil)), nil
}

func multipartCompositeChecksum(parts []multipartPartFile, algorithm string) (string, error) {
	checksum, err := newChecksumHash(algorithm)
	if err != nil {
		return "", err
	}
	for _, part := range parts {
		if part.ChecksumAlgorithm != algorithm || part.Checksum == "" {
			return "", multipartFailure(400, "InvalidPart", "Every completed part must contain the selected checksum.")
		}
		value, err := base64.StdEncoding.DecodeString(part.Checksum)
		if err != nil {
			return "", multipartFailure(400, "InvalidPart", "A stored part checksum is invalid.")
		}
		if _, err := checksum.Write(value); err != nil {
			return "", err
		}
	}
	return encodedChecksum(checksum.Sum(nil)), nil
}

func mustDecodeChecksum(value string) []byte {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil
	}
	return decoded
}

type multipartSequenceReader struct {
	ctx     context.Context
	dir     string
	parts   []multipartPartFile
	index   int
	current *os.File
}

func (r *multipartSequenceReader) Read(buffer []byte) (int, error) {
	for {
		if err := r.ctx.Err(); err != nil {
			return 0, err
		}
		if r.current == nil {
			if r.index >= len(r.parts) {
				return 0, io.EOF
			}
			name := r.parts[r.index].File
			if filepath.Base(name) != name || !strings.HasPrefix(name, "part-") {
				return 0, errors.New("multipart part path is invalid")
			}
			file, err := os.Open(filepath.Join(r.dir, name))
			if err != nil {
				return 0, err
			}
			r.current = file
			r.index++
		}
		count, err := r.current.Read(buffer)
		if errors.Is(err, io.EOF) {
			closeErr := r.current.Close()
			r.current = nil
			if closeErr != nil {
				return count, closeErr
			}
			if count > 0 {
				return count, nil
			}
			continue
		}
		return count, err
	}
}

func (r *multipartSequenceReader) Close() error {
	if r.current == nil {
		return nil
	}
	err := r.current.Close()
	r.current = nil
	return err
}
