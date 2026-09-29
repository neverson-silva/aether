package s3gateway

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"aether/internal/platform/storage"
)

func (s *Server) routeMultipartObject(w http.ResponseWriter, r *http.Request, key string) bool {
	query := r.URL.Query()
	uploadValues, hasUploadID := query["uploadId"]
	_, hasUploads := query["uploads"]
	_, hasPartNumber := query["partNumber"]
	if hasUploadID {
		if len(uploadValues) != 1 || strings.TrimSpace(uploadValues[0]) == "" {
			s.writeError(w, http.StatusBadRequest, "InvalidRequest", "Exactly one uploadId is required.", r.URL.Path)
			return true
		}
		uploadID := uploadValues[0]
		switch r.Method {
		case http.MethodPut:
			partNumbers, ok := query["partNumber"]
			if !ok || len(partNumbers) != 1 {
				s.writeError(w, http.StatusBadRequest, "InvalidArgument", "Exactly one partNumber is required.", r.URL.Path)
				return true
			}
			number, err := strconv.Atoi(partNumbers[0])
			if err != nil {
				s.writeError(w, http.StatusBadRequest, "InvalidArgument", "partNumber must be an integer.", r.URL.Path)
				return true
			}
			s.uploadMultipartPart(w, r, key, uploadID, number)
		case http.MethodGet:
			s.listMultipartParts(w, r, key, uploadID)
		case http.MethodPost:
			s.completeMultipartUpload(w, r, key, uploadID)
		case http.MethodDelete:
			s.abortMultipartUpload(w, r, key, uploadID)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "The requested multipart operation is not supported.", r.URL.Path)
		}
		return true
	}
	if hasUploads {
		if r.Method == http.MethodPost {
			s.initiateMultipartUpload(w, r, key)
			return true
		}
		s.writeError(w, http.StatusNotImplemented, "NotImplemented", "The requested multipart operation is not supported.", r.URL.Path)
		return true
	}
	if hasPartNumber {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", "partNumber requires an uploadId.", r.URL.Path)
		return true
	}
	return false
}

func (s *Server) initiateMultipartUpload(w http.ResponseWriter, r *http.Request, key string) {
	if r.ContentLength != 0 {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", "Multipart initiation does not accept a request body.", r.URL.Path)
		return
	}
	if _, err := storage.NormalizeKey(key); err != nil || strings.Contains(key, folderMarkerName) {
		s.writeError(w, http.StatusBadRequest, "InvalidArgument", "The object key is invalid.", r.URL.Path)
		return
	}
	checksumAlgorithm, err := checksumAlgorithmFromHeaders(r.Header)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", err.Error(), r.URL.Path)
		return
	}
	checksumType := strings.ToUpper(strings.TrimSpace(r.Header.Get("x-amz-checksum-type")))
	if checksumAlgorithm == "" && checksumType != "" {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", "x-amz-checksum-type requires a checksum algorithm.", r.URL.Path)
		return
	}
	if checksumAlgorithm != "" {
		if checksumType == "" {
			checksumType = checksumTypeComposite
		}
		if checksumType != checksumTypeComposite && checksumType != checksumTypeFullObject {
			s.writeError(w, http.StatusBadRequest, "InvalidArgument", "x-amz-checksum-type must be COMPOSITE or FULL_OBJECT.", r.URL.Path)
			return
		}
		if checksumType == checksumTypeFullObject && checksumAlgorithm != "CRC32" && checksumAlgorithm != "CRC32C" {
			s.writeError(w, http.StatusBadRequest, "InvalidRequest", "FULL_OBJECT checksums support CRC32 and CRC32C only.", r.URL.Path)
			return
		}
	}
	upload, err := s.multipart.initiate(s.bucket, key, r.Header.Get("Content-Type"), metadataFromRequest(r), checksumAlgorithm, checksumType)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "InternalError", "The multipart upload could not be initialized.", r.URL.Path)
		return
	}
	s.writeXML(w, http.StatusOK, initiateMultipartResult{
		XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Bucket: upload.Bucket, Key: upload.Key, UploadID: upload.ID,
		ChecksumAlgorithm: upload.ChecksumAlgorithm, ChecksumType: upload.ChecksumType,
	})
}

func (s *Server) uploadMultipartPart(w http.ResponseWriter, r *http.Request, key, uploadID string, number int) {
	body, ok := r.Body.(*temporaryBody)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "InternalError", "The multipart request body could not be staged.", r.URL.Path)
		return
	}
	requestChecksum, err := checksumFromHeaders(r.Header, "")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", err.Error(), r.URL.Path)
		return
	}
	part, err := s.multipart.uploadPart(uploadID, s.bucket, key, number, body, r.Header.Get("Content-MD5"), requestChecksum)
	if err != nil {
		s.writeMultipartFailure(w, r, err)
		return
	}
	w.Header().Set("ETag", quotedETag(part.ETag))
	if part.ChecksumAlgorithm != "" && part.Checksum != "" {
		w.Header().Set(checksumHeaderName(part.ChecksumAlgorithm), part.Checksum)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) completeMultipartUpload(w http.ResponseWriter, r *http.Request, key, uploadID string) {
	var request completeMultipartRequest
	decoder := xml.NewDecoder(io.LimitReader(r.Body, maxCompleteRequestSize))
	if err := decoder.Decode(&request); err != nil || request.XMLName.Local != "CompleteMultipartUpload" {
		s.writeError(w, http.StatusBadRequest, "MalformedXML", "The multipart completion request is not valid XML.", r.URL.Path)
		return
	}
	if _, err := decoder.Token(); err != io.EOF {
		s.writeError(w, http.StatusBadRequest, "MalformedXML", "The multipart completion request is not valid XML.", r.URL.Path)
		return
	}
	requested := make([]completedMultipartPart, 0, len(request.Parts))
	for _, part := range request.Parts {
		algorithm, checksum := part.checksum()
		requested = append(requested, completedMultipartPart{
			PartNumber: part.PartNumber, ETag: part.ETag,
			ChecksumAlgorithm: algorithm, Checksum: checksum,
		})
	}
	requestChecksum, err := checksumFromHeaders(r.Header, "")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", err.Error(), r.URL.Path)
		return
	}
	var providerErr error
	completion, err := s.multipart.complete(r.Context(), uploadID, s.bucket, key, requested, requestChecksum, strings.ToUpper(strings.TrimSpace(r.Header.Get("x-amz-checksum-type"))), func(reader io.Reader, upload multipartUpload, completion multipartCompletion) error {
		metadata := cloneStringMap(upload.Metadata)
		metadata[storage.ObjectETagMetadataKey] = completion.ETag
		if completion.ChecksumAlgorithm != "" && completion.Checksum != "" {
			metadata[checksumMetadataKey(completion.ChecksumAlgorithm)] = completion.Checksum
			metadata[storage.ObjectChecksumTypeMetadataKey] = completion.ChecksumType
		}
		_, providerErr = s.provider.PutObject(r.Context(), storage.PutObjectInput{
			Key: upload.Key, Body: reader, ContentType: upload.ContentType, Metadata: metadata,
		})
		return providerErr
	})
	if err != nil {
		var operationError *multipartError
		if errors.As(err, &operationError) {
			s.writeError(w, operationError.status, operationError.code, operationError.message, r.URL.Path)
			return
		}
		if providerErr != nil {
			s.writeProviderError(w, r, providerErr)
			return
		}
		s.writeError(w, http.StatusInternalServerError, "InternalError", "The multipart upload could not be completed.", r.URL.Path)
		return
	}
	location := (&url.URL{Scheme: "https", Host: r.Host, Path: "/" + s.bucket + "/" + key}).String()
	result := completeMultipartResult{
		XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Location: location,
		Bucket: s.bucket, Key: key, ETag: quotedETag(completion.ETag),
		ChecksumAlgorithm: completion.ChecksumAlgorithm,
		ChecksumType:      completion.ChecksumType,
	}
	setCompletionChecksum(&result, completion.ChecksumAlgorithm, completion.Checksum)
	s.writeXML(w, http.StatusOK, result)
}

func (s *Server) abortMultipartUpload(w http.ResponseWriter, r *http.Request, key, uploadID string) {
	if err := s.multipart.abort(uploadID, s.bucket, key); err != nil {
		s.writeMultipartFailure(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMultipartParts(w http.ResponseWriter, r *http.Request, key, uploadID string) {
	query := r.URL.Query()
	marker, err := parseNonNegativeQuery(query, "part-number-marker", 0, maxMultipartPartCount)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidArgument", err.Error(), r.URL.Path)
		return
	}
	limit, err := parseNonNegativeQuery(query, "max-parts", maxMultipartPageSize, maxMultipartPageSize)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidArgument", err.Error(), r.URL.Path)
		return
	}
	upload, parts, nextMarker, truncated, err := s.multipart.listParts(uploadID, s.bucket, key, marker, limit)
	if err != nil {
		s.writeMultipartFailure(w, r, err)
		return
	}
	response := listPartsResult{
		XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Bucket: upload.Bucket,
		Key: upload.Key, UploadID: upload.ID, PartNumberMarker: marker,
		MaxParts: limit, IsTruncated: truncated, Parts: make([]listedMultipartPart, 0, len(parts)),
		ChecksumAlgorithm: upload.ChecksumAlgorithm, ChecksumType: upload.ChecksumType,
	}
	if truncated {
		response.NextPartNumberMarker = nextMarker
	}
	for _, part := range parts {
		listed := listedMultipartPart{
			PartNumber: partNumberFromName(part.File), LastModified: part.LastModified,
			ETag: quotedETag(part.ETag), Size: part.Size, StorageClass: "STANDARD",
		}
		setListedPartChecksum(&listed, part.ChecksumAlgorithm, part.Checksum)
		response.Parts = append(response.Parts, listed)
	}
	s.writeXML(w, http.StatusOK, response)
}

func (s *Server) listMultipartUploads(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if _, found := query["upload-id-marker"]; found && query.Get("key-marker") == "" {
		s.writeError(w, http.StatusBadRequest, "InvalidArgument", "upload-id-marker requires key-marker.", r.URL.Path)
		return
	}
	limit, err := parseNonNegativeQuery(query, "max-uploads", maxMultipartPageSize, maxMultipartPageSize)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidArgument", err.Error(), r.URL.Path)
		return
	}
	uploads, truncated, nextKey, nextUploadID, err := s.multipart.listUploads(
		s.bucket, query.Get("prefix"), query.Get("key-marker"), query.Get("upload-id-marker"), limit,
	)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "InternalError", "Multipart uploads could not be listed.", r.URL.Path)
		return
	}
	response := listMultipartUploadsResult{
		XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Bucket: s.bucket,
		KeyMarker: query.Get("key-marker"), UploadIDMarker: query.Get("upload-id-marker"),
		Prefix: query.Get("prefix"), MaxUploads: limit, IsTruncated: truncated,
		Uploads: make([]listedMultipartUpload, 0, len(uploads)),
	}
	if truncated {
		response.NextKeyMarker = nextKey
		response.NextUploadIDMarker = nextUploadID
	}
	for _, upload := range uploads {
		response.Uploads = append(response.Uploads, listedMultipartUpload{
			Key: upload.Key, UploadID: upload.ID, Initiated: upload.Initiated, StorageClass: "STANDARD",
			ChecksumAlgorithm: upload.ChecksumAlgorithm, ChecksumType: upload.ChecksumType,
		})
	}
	s.writeXML(w, http.StatusOK, response)
}

func (s *Server) writeMultipartFailure(w http.ResponseWriter, r *http.Request, err error) {
	var operationError *multipartError
	if errors.As(err, &operationError) {
		s.writeError(w, operationError.status, operationError.code, operationError.message, r.URL.Path)
		return
	}
	s.writeError(w, http.StatusInternalServerError, "InternalError", "The multipart request could not be completed.", r.URL.Path)
}

func metadataFromRequest(r *http.Request) map[string]string {
	metadata := make(map[string]string)
	for name, values := range r.Header {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "x-amz-meta-") && len(values) > 0 {
			metadata[strings.TrimPrefix(lower, "x-amz-meta-")] = strings.Join(values, ",")
		}
	}
	return metadata
}

func setListedPartChecksum(part *listedMultipartPart, algorithm, checksum string) {
	if checksum == "" {
		return
	}
	switch algorithm {
	case "CRC32":
		part.ChecksumCRC32 = checksum
	case "CRC32C":
		part.ChecksumCRC32C = checksum
	case "SHA1":
		part.ChecksumSHA1 = checksum
	case "SHA256":
		part.ChecksumSHA256 = checksum
	case "SHA512":
		part.ChecksumSHA512 = checksum
	case "MD5":
		part.ChecksumMD5 = checksum
	}
}

func setCompletionChecksum(result *completeMultipartResult, algorithm, checksum string) {
	if checksum == "" {
		return
	}
	switch algorithm {
	case "CRC32":
		result.ChecksumCRC32 = checksum
	case "CRC32C":
		result.ChecksumCRC32C = checksum
	case "SHA1":
		result.ChecksumSHA1 = checksum
	case "SHA256":
		result.ChecksumSHA256 = checksum
	case "SHA512":
		result.ChecksumSHA512 = checksum
	case "MD5":
		result.ChecksumMD5 = checksum
	}
}

func parseNonNegativeQuery(query url.Values, name string, fallback, maximum int) (int, error) {
	values, found := query[name]
	if !found {
		return fallback, nil
	}
	if len(values) != 1 {
		return 0, fmt.Errorf("%s must be specified once", name)
	}
	parsed, err := strconv.Atoi(values[0])
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return min(parsed, maximum), nil
}

func requestUsesHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]), "https")
}

type initiateMultipartResult struct {
	XMLName           xml.Name `xml:"InitiateMultipartUploadResult"`
	XMLNS             string   `xml:"xmlns,attr"`
	Bucket            string   `xml:"Bucket"`
	Key               string   `xml:"Key"`
	UploadID          string   `xml:"UploadId"`
	ChecksumAlgorithm string   `xml:"ChecksumAlgorithm,omitempty"`
	ChecksumType      string   `xml:"ChecksumType,omitempty"`
}

type completeMultipartRequest struct {
	XMLName xml.Name                      `xml:"CompleteMultipartUpload"`
	Parts   []completedMultipartPartEntry `xml:"Part"`
}

type completedMultipartPartEntry struct {
	PartNumber     int    `xml:"PartNumber"`
	ETag           string `xml:"ETag"`
	ChecksumCRC32  string `xml:"ChecksumCRC32"`
	ChecksumCRC32C string `xml:"ChecksumCRC32C"`
	ChecksumSHA1   string `xml:"ChecksumSHA1"`
	ChecksumSHA256 string `xml:"ChecksumSHA256"`
	ChecksumSHA512 string `xml:"ChecksumSHA512"`
	ChecksumMD5    string `xml:"ChecksumMD5"`
}

func (p completedMultipartPartEntry) checksum() (string, string) {
	checksums := []struct {
		algorithm string
		value     string
	}{
		{algorithm: "CRC32", value: p.ChecksumCRC32},
		{algorithm: "CRC32C", value: p.ChecksumCRC32C},
		{algorithm: "SHA1", value: p.ChecksumSHA1},
		{algorithm: "SHA256", value: p.ChecksumSHA256},
		{algorithm: "SHA512", value: p.ChecksumSHA512},
		{algorithm: "MD5", value: p.ChecksumMD5},
	}
	for _, checksum := range checksums {
		if checksum.value != "" {
			return checksum.algorithm, checksum.value
		}
	}
	return "", ""
}

type completeMultipartResult struct {
	XMLName           xml.Name `xml:"CompleteMultipartUploadResult"`
	XMLNS             string   `xml:"xmlns,attr"`
	Location          string   `xml:"Location"`
	Bucket            string   `xml:"Bucket"`
	Key               string   `xml:"Key"`
	ETag              string   `xml:"ETag"`
	ChecksumCRC32     string   `xml:"ChecksumCRC32,omitempty"`
	ChecksumCRC32C    string   `xml:"ChecksumCRC32C,omitempty"`
	ChecksumSHA1      string   `xml:"ChecksumSHA1,omitempty"`
	ChecksumSHA256    string   `xml:"ChecksumSHA256,omitempty"`
	ChecksumSHA512    string   `xml:"ChecksumSHA512,omitempty"`
	ChecksumMD5       string   `xml:"ChecksumMD5,omitempty"`
	ChecksumType      string   `xml:"ChecksumType,omitempty"`
	ChecksumAlgorithm string   `xml:"ChecksumAlgorithm,omitempty"`
}

type listPartsResult struct {
	XMLName              xml.Name              `xml:"ListPartsResult"`
	XMLNS                string                `xml:"xmlns,attr"`
	Bucket               string                `xml:"Bucket"`
	Key                  string                `xml:"Key"`
	UploadID             string                `xml:"UploadId"`
	PartNumberMarker     int                   `xml:"PartNumberMarker"`
	NextPartNumberMarker int                   `xml:"NextPartNumberMarker,omitempty"`
	MaxParts             int                   `xml:"MaxParts"`
	IsTruncated          bool                  `xml:"IsTruncated"`
	ChecksumAlgorithm    string                `xml:"ChecksumAlgorithm,omitempty"`
	ChecksumType         string                `xml:"ChecksumType,omitempty"`
	Parts                []listedMultipartPart `xml:"Part"`
}

type listedMultipartPart struct {
	PartNumber     int       `xml:"PartNumber"`
	LastModified   time.Time `xml:"LastModified"`
	ETag           string    `xml:"ETag"`
	Size           int64     `xml:"Size"`
	StorageClass   string    `xml:"StorageClass"`
	ChecksumCRC32  string    `xml:"ChecksumCRC32,omitempty"`
	ChecksumCRC32C string    `xml:"ChecksumCRC32C,omitempty"`
	ChecksumSHA1   string    `xml:"ChecksumSHA1,omitempty"`
	ChecksumSHA256 string    `xml:"ChecksumSHA256,omitempty"`
	ChecksumSHA512 string    `xml:"ChecksumSHA512,omitempty"`
	ChecksumMD5    string    `xml:"ChecksumMD5,omitempty"`
}

type listMultipartUploadsResult struct {
	XMLName            xml.Name                `xml:"ListMultipartUploadsResult"`
	XMLNS              string                  `xml:"xmlns,attr"`
	Bucket             string                  `xml:"Bucket"`
	KeyMarker          string                  `xml:"KeyMarker"`
	UploadIDMarker     string                  `xml:"UploadIdMarker"`
	NextKeyMarker      string                  `xml:"NextKeyMarker,omitempty"`
	NextUploadIDMarker string                  `xml:"NextUploadIdMarker,omitempty"`
	Prefix             string                  `xml:"Prefix"`
	MaxUploads         int                     `xml:"MaxUploads"`
	IsTruncated        bool                    `xml:"IsTruncated"`
	ChecksumAlgorithm  string                  `xml:"ChecksumAlgorithm,omitempty"`
	ChecksumType       string                  `xml:"ChecksumType,omitempty"`
	Uploads            []listedMultipartUpload `xml:"Upload"`
}

type listedMultipartUpload struct {
	Key               string    `xml:"Key"`
	UploadID          string    `xml:"UploadId"`
	Initiated         time.Time `xml:"Initiated"`
	StorageClass      string    `xml:"StorageClass"`
	ChecksumAlgorithm string    `xml:"ChecksumAlgorithm,omitempty"`
	ChecksumType      string    `xml:"ChecksumType,omitempty"`
}
