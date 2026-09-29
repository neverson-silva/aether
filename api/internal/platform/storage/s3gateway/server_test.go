package s3gateway

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"aether/internal/platform/storage"
)

const (
	gatewayTestAccessKey = "AKIATESTACCESSKEY123"
	gatewayTestSecretKey = "gateway-test-secret-key"
)

type gatewayTestObject struct {
	Body        []byte
	ContentType string
	Metadata    map[string]string
	Modified    time.Time
}

type gatewayTestProvider struct {
	mu      sync.Mutex
	objects map[string]gatewayTestObject
}

func newGatewayTestProvider() *gatewayTestProvider {
	return &gatewayTestProvider{objects: make(map[string]gatewayTestObject)}
}

func (p *gatewayTestProvider) Capabilities() storage.Capabilities {
	return storage.Capabilities{Streaming: true, ResumableUpload: true, Metadata: true}
}

func (p *gatewayTestProvider) PutObject(_ context.Context, input storage.PutObjectInput) (*storage.PutObjectOutput, error) {
	body, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	p.objects[input.Key] = gatewayTestObject{
		Body: body, ContentType: input.ContentType, Metadata: cloneStringMap(input.Metadata), Modified: time.Now().UTC(),
	}
	p.mu.Unlock()
	return &storage.PutObjectOutput{Key: input.Key, Size: int64(len(body)), ETag: input.Metadata[storage.ObjectETagMetadataKey]}, nil
}

func (p *gatewayTestProvider) GetObject(_ context.Context, input storage.GetObjectInput) (*storage.GetObjectOutput, error) {
	p.mu.Lock()
	object, found := p.objects[input.Key]
	p.mu.Unlock()
	if !found {
		return nil, storage.ErrObjectNotFound
	}
	return &storage.GetObjectOutput{
		Key: input.Key, Body: io.NopCloser(bytes.NewReader(object.Body)), ContentType: object.ContentType,
		ContentLength: int64(len(object.Body)), ETag: object.Metadata[storage.ObjectETagMetadataKey],
		LastModified: object.Modified, Metadata: cloneStringMap(object.Metadata),
	}, nil
}

func (p *gatewayTestProvider) HeadObject(_ context.Context, input storage.HeadObjectInput) (*storage.HeadObjectOutput, error) {
	p.mu.Lock()
	object, found := p.objects[input.Key]
	p.mu.Unlock()
	if !found {
		return nil, storage.ErrObjectNotFound
	}
	return &storage.HeadObjectOutput{
		Key: input.Key, ContentLength: int64(len(object.Body)), ContentType: object.ContentType,
		ETag: object.Metadata[storage.ObjectETagMetadataKey], LastModified: object.Modified,
		Metadata: cloneStringMap(object.Metadata),
	}, nil
}

func (p *gatewayTestProvider) DeleteObject(_ context.Context, input storage.DeleteObjectInput) error {
	p.mu.Lock()
	delete(p.objects, input.Key)
	p.mu.Unlock()
	return nil
}

func (p *gatewayTestProvider) ListObjects(_ context.Context, input storage.ListObjectsInput) (*storage.ListObjectsOutput, error) {
	p.mu.Lock()
	keys := make([]string, 0, len(p.objects))
	for key := range p.objects {
		if strings.HasPrefix(key, input.Prefix) {
			keys = append(keys, key)
		}
	}
	p.mu.Unlock()
	sort.Strings(keys)
	offset := 0
	if input.Cursor != "" {
		if _, err := fmt.Sscanf(input.Cursor, "%d", &offset); err != nil || offset < 0 || offset > len(keys) {
			return nil, storage.ErrInvalidObjectKey
		}
	}
	limit := input.Limit
	if limit <= 0 || limit > len(keys) {
		limit = len(keys)
	}
	end := min(offset+limit, len(keys))
	result := &storage.ListObjectsOutput{Objects: make([]storage.ObjectInfo, 0, end-offset)}
	p.mu.Lock()
	for _, key := range keys[offset:end] {
		object := p.objects[key]
		result.Objects = append(result.Objects, storage.ObjectInfo{
			Key: key, Size: int64(len(object.Body)), ContentType: object.ContentType,
			ETag: object.Metadata[storage.ObjectETagMetadataKey], LastModified: object.Modified,
		})
	}
	p.mu.Unlock()
	if end < len(keys) {
		result.NextCursor = fmt.Sprintf("%d", end)
	}
	return result, nil
}

func (p *gatewayTestProvider) CopyObject(_ context.Context, input storage.CopyObjectInput) (*storage.CopyObjectOutput, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	object, found := p.objects[input.SourceKey]
	if !found {
		return nil, storage.ErrObjectNotFound
	}
	p.objects[input.DestinationKey] = object
	return &storage.CopyObjectOutput{
		SourceKey: input.SourceKey, DestinationKey: input.DestinationKey,
		ETag: object.Metadata[storage.ObjectETagMetadataKey],
	}, nil
}

func (p *gatewayTestProvider) EnsureFolder(context.Context, string) error {
	return nil
}

func TestMultipartUploadLifecycleWithCompositeChecksum(t *testing.T) {
	provider := newGatewayTestProvider()
	dataDir := t.TempDir()
	server, err := New(Config{
		Provider: provider, AccessKey: gatewayTestAccessKey, SecretKey: gatewayTestSecretKey,
		Bucket: "gdrive", TempDir: dataDir + "/tmp", MultipartDir: dataDir + "/parts",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	response := serveGatewayRequest(t, server, http.MethodPost, "/gdrive/archive.bin?uploads=", nil, http.Header{
		"Content-Type":                 []string{"application/octet-stream"},
		"X-Amz-Sdk-Checksum-Algorithm": []string{"CRC32"},
		"X-Amz-Checksum-Type":          []string{checksumTypeComposite},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("initiate status = %d, body = %s", response.Code, response.Body.String())
	}
	var initiated initiateMultipartResult
	if err := xml.Unmarshal(response.Body.Bytes(), &initiated); err != nil {
		t.Fatalf("decode initiate response: %v", err)
	}
	if initiated.UploadID == "" || initiated.ChecksumAlgorithm != "CRC32" || initiated.ChecksumType != checksumTypeComposite {
		t.Fatalf("unexpected initiate response: %+v", initiated)
	}

	partOne := bytes.Repeat([]byte("a"), int(minMultipartPartSize))
	partTwo := []byte("tail")
	partOneETag, partOneChecksum := gatewayTestPart(t, server, "archive.bin", initiated.UploadID, 1, partOne, "CRC32")
	partTwoETag, partTwoChecksum := gatewayTestPart(t, server, "archive.bin", initiated.UploadID, 2, partTwo, "CRC32")

	listTarget := "/gdrive/archive.bin?uploadId=" + url.QueryEscape(initiated.UploadID)
	response = serveGatewayRequest(t, server, http.MethodGet, listTarget, nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("list parts status = %d, body = %s", response.Code, response.Body.String())
	}
	var listed listPartsResult
	if err := xml.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list parts response: %v", err)
	}
	if len(listed.Parts) != 2 || listed.Parts[0].ChecksumCRC32 != partOneChecksum || listed.Parts[1].ChecksumCRC32 != partTwoChecksum {
		t.Fatalf("unexpected listed parts: %+v", listed.Parts)
	}

	completionBody := []byte(fmt.Sprintf(
		"<CompleteMultipartUpload><Part><PartNumber>1</PartNumber><ETag>\"%s\"</ETag><ChecksumCRC32>%s</ChecksumCRC32></Part><Part><PartNumber>2</PartNumber><ETag>\"%s\"</ETag><ChecksumCRC32>%s</ChecksumCRC32></Part></CompleteMultipartUpload>",
		partOneETag, partOneChecksum, partTwoETag, partTwoChecksum,
	))
	completionHeaders := http.Header{
		"X-Amz-Checksum-Algorithm": []string{"CRC32"},
		"X-Amz-Checksum-Type":      []string{checksumTypeComposite},
	}
	response = serveGatewayRequest(t, server, http.MethodPost, listTarget, completionBody, completionHeaders)
	if response.Code != http.StatusOK {
		t.Fatalf("complete status = %d, body = %s", response.Code, response.Body.String())
	}
	var completed completeMultipartResult
	if err := xml.Unmarshal(response.Body.Bytes(), &completed); err != nil {
		t.Fatalf("decode completion response: %v", err)
	}
	if completed.ChecksumCRC32 == "" || completed.ChecksumType != checksumTypeComposite || completed.ETag == "" {
		t.Fatalf("unexpected completion response: %+v", completed)
	}

	getResponse := serveGatewayRequest(t, server, http.MethodGet, "/gdrive/archive.bin", nil, nil)
	if getResponse.Code != http.StatusOK || !bytes.Equal(getResponse.Body.Bytes(), append(partOne, partTwo...)) {
		t.Fatalf("download did not return the completed multipart object: status=%d size=%d", getResponse.Code, getResponse.Body.Len())
	}
	if getResponse.Header().Get("x-amz-checksum-crc32") != completed.ChecksumCRC32 || getResponse.Header().Get("x-amz-checksum-type") != checksumTypeComposite {
		t.Fatalf("download checksum headers = %v", getResponse.Header())
	}
}

func TestMultipartAbortAndSmallNonFinalPart(t *testing.T) {
	provider := newGatewayTestProvider()
	dataDir := t.TempDir()
	server, err := New(Config{
		Provider: provider, AccessKey: gatewayTestAccessKey, SecretKey: gatewayTestSecretKey,
		Bucket: "gdrive", TempDir: dataDir + "/tmp", MultipartDir: dataDir + "/parts",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	response := serveGatewayRequest(t, server, http.MethodPost, "/gdrive/bad.bin?uploads=", nil, nil)
	var initiated initiateMultipartResult
	if err := xml.Unmarshal(response.Body.Bytes(), &initiated); err != nil {
		t.Fatalf("decode initiate response: %v", err)
	}
	otherResponse := serveGatewayRequest(t, server, http.MethodPost, "/gdrive/other.bin?uploads=", nil, nil)
	var otherUpload initiateMultipartResult
	if err := xml.Unmarshal(otherResponse.Body.Bytes(), &otherUpload); err != nil {
		t.Fatalf("decode second initiate response: %v", err)
	}
	listUploadsResponse := serveGatewayRequest(t, server, http.MethodGet, "/gdrive?uploads=&max-uploads=1", nil, nil)
	var listedUploads listMultipartUploadsResult
	if err := xml.Unmarshal(listUploadsResponse.Body.Bytes(), &listedUploads); err != nil {
		t.Fatalf("decode multipart uploads response: %v", err)
	}
	if listUploadsResponse.Code != http.StatusOK || !listedUploads.IsTruncated || len(listedUploads.Uploads) != 1 {
		t.Fatalf("first multipart upload page = %d %+v", listUploadsResponse.Code, listedUploads)
	}
	nextUploadsTarget := "/gdrive?uploads=&max-uploads=1&key-marker=" + url.QueryEscape(listedUploads.NextKeyMarker) + "&upload-id-marker=" + url.QueryEscape(listedUploads.NextUploadIDMarker)
	listUploadsResponse = serveGatewayRequest(t, server, http.MethodGet, nextUploadsTarget, nil, nil)
	listedUploads = listMultipartUploadsResult{}
	if err := xml.Unmarshal(listUploadsResponse.Body.Bytes(), &listedUploads); err != nil {
		t.Fatalf("decode second multipart uploads page: %v", err)
	}
	if listUploadsResponse.Code != http.StatusOK || len(listedUploads.Uploads) != 1 || listedUploads.Uploads[0].Key != "other.bin" {
		t.Fatalf("second multipart upload page = %d %+v", listUploadsResponse.Code, listedUploads)
	}
	partETag, _ := gatewayTestPart(t, server, "bad.bin", initiated.UploadID, 1, []byte("first"), "")
	_, _ = gatewayTestPart(t, server, "bad.bin", initiated.UploadID, 2, []byte("last"), "")
	completeXML := []byte(fmt.Sprintf(
		"<CompleteMultipartUpload><Part><PartNumber>1</PartNumber><ETag>\"%s\"</ETag></Part><Part><PartNumber>2</PartNumber><ETag>\"%s\"</ETag></Part></CompleteMultipartUpload>",
		partETag, md5Hex([]byte("last")),
	))
	response = serveGatewayRequest(t, server, http.MethodPost, "/gdrive/bad.bin?uploadId="+initiated.UploadID, completeXML, nil)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "EntityTooSmall") {
		t.Fatalf("small non-final part response = %d %s", response.Code, response.Body.String())
	}

	abortResponse := serveGatewayRequest(t, server, http.MethodDelete, "/gdrive/bad.bin?uploadId="+initiated.UploadID, nil, nil)
	if abortResponse.Code != http.StatusNoContent {
		t.Fatalf("abort status = %d, body = %s", abortResponse.Code, abortResponse.Body.String())
	}
	listResponse := serveGatewayRequest(t, server, http.MethodGet, "/gdrive/bad.bin?uploadId="+initiated.UploadID, nil, nil)
	if listResponse.Code != http.StatusNotFound || !strings.Contains(listResponse.Body.String(), "NoSuchUpload") {
		t.Fatalf("list after abort response = %d %s", listResponse.Code, listResponse.Body.String())
	}
	otherAbort := serveGatewayRequest(t, server, http.MethodDelete, "/gdrive/other.bin?uploadId="+otherUpload.UploadID, nil, nil)
	if otherAbort.Code != http.StatusNoContent {
		t.Fatalf("second abort status = %d, body = %s", otherAbort.Code, otherAbort.Body.String())
	}
}

func TestMultipartFullObjectChecksumIsVerifiedBeforeCommit(t *testing.T) {
	provider := newGatewayTestProvider()
	dataDir := t.TempDir()
	server, err := New(Config{
		Provider: provider, AccessKey: gatewayTestAccessKey, SecretKey: gatewayTestSecretKey,
		Bucket: "gdrive", TempDir: dataDir + "/tmp", MultipartDir: dataDir + "/parts",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	response := serveGatewayRequest(t, server, http.MethodPost, "/gdrive/full.bin?uploads=", nil, http.Header{
		"X-Amz-Checksum-Algorithm": []string{"CRC32"},
		"X-Amz-Checksum-Type":      []string{checksumTypeFullObject},
	})
	var initiated initiateMultipartResult
	if err := xml.Unmarshal(response.Body.Bytes(), &initiated); err != nil {
		t.Fatalf("decode initiate response: %v", err)
	}
	body := bytes.Repeat([]byte("z"), int(minMultipartPartSize))
	partETag, partChecksum := gatewayTestPart(t, server, "full.bin", initiated.UploadID, 1, body, "CRC32")
	completionXML := []byte(fmt.Sprintf(
		"<CompleteMultipartUpload><Part><PartNumber>1</PartNumber><ETag>\"%s\"</ETag><ChecksumCRC32>%s</ChecksumCRC32></Part></CompleteMultipartUpload>",
		partETag, partChecksum,
	))
	target := "/gdrive/full.bin?uploadId=" + initiated.UploadID
	wrongChecksum := gatewayTestChecksum(t, "CRC32", []byte("different content"))
	response = serveGatewayRequest(t, server, http.MethodPost, target, completionXML, http.Header{
		"X-Amz-Checksum-Crc32": []string{wrongChecksum},
		"X-Amz-Checksum-Type":  []string{checksumTypeFullObject},
	})
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "BadDigest") {
		t.Fatalf("bad full-object checksum response = %d %s", response.Code, response.Body.String())
	}
	provider.mu.Lock()
	_, prematurelyCommitted := provider.objects["full.bin"]
	provider.mu.Unlock()
	if prematurelyCommitted {
		t.Fatal("object was committed before the checksum was verified")
	}

	fullChecksum := gatewayTestChecksum(t, "CRC32", body)
	response = serveGatewayRequest(t, server, http.MethodPost, target, completionXML, http.Header{
		"X-Amz-Checksum-Crc32": []string{fullChecksum},
		"X-Amz-Checksum-Type":  []string{checksumTypeFullObject},
	})
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("<ChecksumType>FULL_OBJECT</ChecksumType>")) {
		t.Fatalf("valid full-object checksum response = %d %s", response.Code, response.Body.String())
	}
	getResponse := serveGatewayRequest(t, server, http.MethodGet, "/gdrive/full.bin", nil, nil)
	if getResponse.Code != http.StatusOK || getResponse.Header().Get("x-amz-checksum-crc32") != fullChecksum {
		t.Fatalf("full-object checksum headers = %d %v", getResponse.Code, getResponse.Header())
	}
}

func TestPutObjectChecksumRoundTrip(t *testing.T) {
	provider := newGatewayTestProvider()
	dataDir := t.TempDir()
	server, err := New(Config{
		Provider: provider, AccessKey: gatewayTestAccessKey, SecretKey: gatewayTestSecretKey,
		Bucket: "gdrive", TempDir: dataDir + "/tmp", MultipartDir: dataDir + "/parts",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	body := []byte("document bytes")
	checksum := gatewayTestChecksum(t, "CRC32", body)
	md5Digest := md5.Sum(body)
	response := serveGatewayRequest(t, server, http.MethodPut, "/gdrive/docs/readme.txt", body, http.Header{
		"X-Amz-Checksum-Crc32": []string{checksum},
		"Content-Md5":          []string{base64.StdEncoding.EncodeToString(md5Digest[:])},
	})
	if response.Code != http.StatusOK || response.Header().Get("x-amz-checksum-crc32") != checksum {
		t.Fatalf("put response = %d %v %s", response.Code, response.Header(), response.Body.String())
	}
	response = serveGatewayRequest(t, server, http.MethodGet, "/gdrive/docs/readme.txt", nil, nil)
	if response.Code != http.StatusOK || response.Header().Get("x-amz-checksum-crc32") != checksum || !bytes.Equal(response.Body.Bytes(), body) {
		t.Fatalf("get response = %d %v %s", response.Code, response.Header(), response.Body.String())
	}
}

func gatewayTestPart(t *testing.T, server http.Handler, key, uploadID string, number int, body []byte, algorithm string) (string, string) {
	t.Helper()
	target := fmt.Sprintf("/gdrive/%s?partNumber=%d&uploadId=%s", key, number, url.QueryEscape(uploadID))
	headers := make(http.Header)
	checksum := ""
	if algorithm != "" {
		checksum = gatewayTestChecksum(t, algorithm, body)
		headers.Set(checksumHeaderName(algorithm), checksum)
	}
	response := serveGatewayRequest(t, server, http.MethodPut, target, body, headers)
	if response.Code != http.StatusOK {
		t.Fatalf("upload part %d status = %d, body = %s", number, response.Code, response.Body.String())
	}
	return strings.Trim(response.Header().Get("ETag"), `"`), checksum
}

func gatewayTestChecksum(t *testing.T, algorithm string, body []byte) string {
	t.Helper()
	hash, err := newChecksumHash(algorithm)
	if err != nil {
		t.Fatalf("new checksum: %v", err)
	}
	if _, err := hash.Write(body); err != nil {
		t.Fatalf("write checksum: %v", err)
	}
	return encodedChecksum(hash.Sum(nil))
}

func md5Hex(body []byte) string {
	digest := md5.Sum(body)
	return hex.EncodeToString(digest[:])
}

func serveGatewayRequest(t *testing.T, server http.Handler, method, target string, body []byte, additionalHeaders http.Header) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, "https://gateway.test"+target, bytes.NewReader(body))
	for name, values := range additionalHeaders {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	date := time.Now().UTC().Format("20060102T150405Z")
	payloadDigest := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(payloadDigest[:])
	request.Header.Set("x-amz-date", date)
	request.Header.Set("x-amz-content-sha256", payloadHash)
	headerNames := []string{"host"}
	for name := range request.Header {
		if strings.EqualFold(name, "Authorization") {
			continue
		}
		headerNames = append(headerNames, strings.ToLower(name))
	}
	sort.Strings(headerNames)
	canonicalHeaders := make([]string, 0, len(headerNames))
	for _, name := range headerNames {
		value := request.Host
		if name != "host" {
			value = strings.Join(request.Header.Values(http.CanonicalHeaderKey(name)), ",")
		}
		canonicalHeaders = append(canonicalHeaders, name+":"+normalizeHeaderValue(value)+"\n")
	}
	canonicalURI, err := canonicalRequestURI(request.URL)
	if err != nil {
		t.Fatalf("canonical URI: %v", err)
	}
	canonicalQuery, err := canonicalRequestQuery(request.URL.RawQuery)
	if err != nil {
		t.Fatalf("canonical query: %v", err)
	}
	signedHeaders := strings.Join(headerNames, ";")
	canonicalRequest := strings.Join([]string{
		method, canonicalURI, canonicalQuery, strings.Join(canonicalHeaders, ""), signedHeaders, payloadHash,
	}, "\n")
	requestDigest := sha256.Sum256([]byte(canonicalRequest))
	dateScope := date[:8]
	credentialScope := dateScope + "/us-east-1/s3/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256", date, credentialScope, hex.EncodeToString(requestDigest[:]),
	}, "\n")
	signature := hex.EncodeToString(hmacSHA256(signatureKey(gatewayTestSecretKey, dateScope, "us-east-1"), []byte(stringToSign)))
	request.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		gatewayTestAccessKey, credentialScope, signedHeaders, signature,
	))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}
