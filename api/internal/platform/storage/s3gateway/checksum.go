package s3gateway

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"hash"
	"hash/crc32"
	"net/http"
	"strings"
)

const (
	checksumTypeComposite  = "COMPOSITE"
	checksumTypeFullObject = "FULL_OBJECT"
)

type checksumValue struct {
	Algorithm string
	Value     string
}

func checksumAlgorithmFromHeaders(headers http.Header) (string, error) {
	algorithm := ""
	for _, name := range []string{"x-amz-checksum-algorithm", "x-amz-sdk-checksum-algorithm"} {
		value := strings.ToUpper(strings.TrimSpace(headers.Get(name)))
		if value == "" {
			continue
		}
		if _, err := newChecksumHash(value); err != nil {
			return "", err
		}
		if algorithm != "" && algorithm != value {
			return "", errors.New("checksum algorithm headers do not match")
		}
		algorithm = value
	}
	return algorithm, nil
}

func checksumFromHeaders(headers http.Header, requiredAlgorithm string) (checksumValue, error) {
	algorithm, err := checksumAlgorithmFromHeaders(headers)
	if err != nil {
		return checksumValue{}, err
	}
	if requiredAlgorithm != "" {
		requiredAlgorithm = strings.ToUpper(strings.TrimSpace(requiredAlgorithm))
		if algorithm != "" && algorithm != requiredAlgorithm {
			return checksumValue{}, errors.New("checksum algorithm does not match the multipart upload")
		}
		algorithm = requiredAlgorithm
	}
	value := ""
	valueAlgorithm := ""
	for _, candidate := range []string{"CRC32", "CRC32C", "SHA1", "SHA256", "SHA512", "MD5"} {
		header := checksumHeaderName(candidate)
		values := headers.Values(header)
		if len(values) == 0 {
			continue
		}
		if len(values) != 1 || valueAlgorithm != "" {
			return checksumValue{}, errors.New("exactly one checksum header is supported")
		}
		valueAlgorithm = candidate
		value = strings.TrimSpace(values[0])
	}
	for name, values := range headers {
		lower := strings.ToLower(name)
		if !strings.HasPrefix(lower, "x-amz-checksum-") || lower == "x-amz-checksum-algorithm" || lower == "x-amz-checksum-type" || lower == "x-amz-checksum-mode" {
			continue
		}
		if checksumAlgorithmForHeader(lower) == "" && len(values) > 0 {
			return checksumValue{}, errors.New("the requested checksum algorithm is not supported")
		}
	}
	if valueAlgorithm != "" {
		if algorithm != "" && algorithm != valueAlgorithm {
			return checksumValue{}, errors.New("checksum header does not match the requested algorithm")
		}
		algorithm = valueAlgorithm
	}
	if algorithm != "" {
		if _, err := newChecksumHash(algorithm); err != nil {
			return checksumValue{}, err
		}
	}
	return checksumValue{Algorithm: algorithm, Value: value}, nil
}

func newChecksumHash(algorithm string) (hash.Hash, error) {
	switch strings.ToUpper(strings.TrimSpace(algorithm)) {
	case "CRC32":
		return crc32.NewIEEE(), nil
	case "CRC32C":
		return crc32.New(crc32.MakeTable(crc32.Castagnoli)), nil
	case "SHA1":
		return sha1.New(), nil
	case "SHA256":
		return sha256.New(), nil
	case "SHA512":
		return sha512.New(), nil
	case "MD5":
		return md5.New(), nil
	default:
		return nil, errors.New("the requested checksum algorithm is not supported")
	}
}

func checksumHeaderName(algorithm string) string {
	switch strings.ToUpper(strings.TrimSpace(algorithm)) {
	case "CRC32":
		return "x-amz-checksum-crc32"
	case "CRC32C":
		return "x-amz-checksum-crc32c"
	case "SHA1":
		return "x-amz-checksum-sha1"
	case "SHA256":
		return "x-amz-checksum-sha256"
	case "SHA512":
		return "x-amz-checksum-sha512"
	case "MD5":
		return "x-amz-checksum-md5"
	default:
		return ""
	}
}

func checksumXMLName(algorithm string) string {
	switch strings.ToUpper(strings.TrimSpace(algorithm)) {
	case "CRC32":
		return "ChecksumCRC32"
	case "CRC32C":
		return "ChecksumCRC32C"
	case "SHA1":
		return "ChecksumSHA1"
	case "SHA256":
		return "ChecksumSHA256"
	case "SHA512":
		return "ChecksumSHA512"
	case "MD5":
		return "ChecksumMD5"
	default:
		return ""
	}
}

func checksumAlgorithmForHeader(name string) string {
	for _, algorithm := range []string{"CRC32", "CRC32C", "SHA1", "SHA256", "SHA512", "MD5"} {
		if strings.EqualFold(name, checksumHeaderName(algorithm)) {
			return algorithm
		}
	}
	return ""
}

func checksumMetadataKey(algorithm string) string {
	return "aether_s3_checksum_" + strings.ToLower(strings.TrimSpace(algorithm))
}

func checksumMatches(encoded string, sum []byte) bool {
	expected, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(expected) != len(sum) {
		return false
	}
	return equalBytes(expected, sum)
}

func encodedChecksum(sum []byte) string {
	return base64.StdEncoding.EncodeToString(sum)
}
