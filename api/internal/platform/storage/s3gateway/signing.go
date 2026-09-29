package s3gateway

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

func verifySignature(r *http.Request, accessKey, secretKey string) (string, error) {
	algorithm, value, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || algorithm != "AWS4-HMAC-SHA256" {
		return "", errors.New("unsupported authorization header")
	}
	attributes := make(map[string]string, 3)
	for _, entry := range strings.Split(value, ",") {
		name, item, ok := strings.Cut(strings.TrimSpace(entry), "=")
		if !ok || name == "" || item == "" {
			return "", errors.New("invalid authorization header")
		}
		attributes[name] = item
	}
	credentialParts := strings.Split(attributes["Credential"], "/")
	if len(credentialParts) != 5 || credentialParts[0] != accessKey || credentialParts[2] == "" || credentialParts[3] != "s3" || credentialParts[4] != "aws4_request" {
		return "", errors.New("invalid credential scope")
	}
	date := r.Header.Get("x-amz-date")
	signedAt, err := time.Parse("20060102T150405Z", date)
	if err != nil || signedAt.UTC().Format("20060102") != credentialParts[1] || time.Since(signedAt) > 15*time.Minute || time.Until(signedAt) > 15*time.Minute {
		return "", errors.New("invalid request timestamp")
	}
	signedHeaders := strings.Split(attributes["SignedHeaders"], ";")
	if len(signedHeaders) == 0 || !sort.StringsAreSorted(signedHeaders) {
		return "", errors.New("invalid signed headers")
	}
	canonicalHeaders := make([]string, 0, len(signedHeaders))
	seenHeaders := make(map[string]struct{}, len(signedHeaders))
	containsHost := false
	containsPayloadHash := false
	containsDate := false
	for _, name := range signedHeaders {
		if name == "" || strings.ToLower(name) != name {
			return "", errors.New("invalid signed header name")
		}
		if _, exists := seenHeaders[name]; exists {
			return "", errors.New("duplicate signed header")
		}
		seenHeaders[name] = struct{}{}
		containsHost = containsHost || name == "host"
		containsPayloadHash = containsPayloadHash || name == "x-amz-content-sha256"
		containsDate = containsDate || name == "x-amz-date"
		var values []string
		if name == "host" {
			values = []string{r.Host}
		} else {
			values = r.Header.Values(http.CanonicalHeaderKey(name))
		}
		if len(values) == 0 {
			return "", errors.New("signed header is missing")
		}
		canonicalHeaders = append(canonicalHeaders, name+":"+normalizeHeaderValue(strings.Join(values, ","))+"\n")
	}
	payloadHash := r.Header.Get("x-amz-content-sha256")
	if payloadHash == "" || !containsHost || !containsPayloadHash || !containsDate {
		return "", errors.New("required headers are not signed")
	}
	canonicalURI, err := canonicalRequestURI(r.URL)
	if err != nil {
		return "", err
	}
	canonicalQuery, err := canonicalRequestQuery(r.URL.RawQuery)
	if err != nil {
		return "", err
	}
	canonicalRequest := strings.Join([]string{
		r.Method,
		canonicalURI,
		canonicalQuery,
		strings.Join(canonicalHeaders, ""),
		strings.Join(signedHeaders, ";"),
		payloadHash,
	}, "\n")
	requestHash := sha256.Sum256([]byte(canonicalRequest))
	scope := strings.Join(credentialParts[1:], "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		date,
		scope,
		hex.EncodeToString(requestHash[:]),
	}, "\n")
	signingKey := signatureKey(secretKey, credentialParts[1], credentialParts[2])
	expected := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))
	provided, err := hex.DecodeString(attributes["Signature"])
	expectedBytes, expectedErr := hex.DecodeString(expected)
	if err != nil || expectedErr != nil || !hmac.Equal(provided, expectedBytes) {
		return "", errors.New("request signature mismatch")
	}
	return payloadHash, nil
}

func canonicalRequestURI(value *url.URL) (string, error) {
	segments := strings.Split(value.EscapedPath(), "/")
	for index, segment := range segments {
		decoded, err := url.PathUnescape(segment)
		if err != nil {
			return "", err
		}
		segments[index] = awsEncode(decoded)
	}
	return strings.Join(segments, "/"), nil
}

func canonicalRequestQuery(raw string) (string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", err
	}
	type queryEntry struct {
		name  string
		value string
	}
	entries := make([]queryEntry, 0)
	for name, items := range values {
		if len(items) == 0 {
			items = []string{""}
		}
		for _, item := range items {
			entries = append(entries, queryEntry{name: awsEncode(name), value: awsEncode(item)})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].name == entries[j].name {
			return entries[i].value < entries[j].value
		}
		return entries[i].name < entries[j].name
	})
	encoded := make([]string, 0, len(entries))
	for _, entry := range entries {
		encoded = append(encoded, entry.name+"="+entry.value)
	}
	return strings.Join(encoded, "&"), nil
}

func awsEncode(value string) string {
	const digits = "0123456789ABCDEF"
	var encoded strings.Builder
	for _, valueByte := range []byte(value) {
		if valueByte >= 'a' && valueByte <= 'z' || valueByte >= 'A' && valueByte <= 'Z' || valueByte >= '0' && valueByte <= '9' || strings.ContainsRune("-_.~", rune(valueByte)) {
			encoded.WriteByte(valueByte)
			continue
		}
		encoded.WriteByte('%')
		encoded.WriteByte(digits[valueByte>>4])
		encoded.WriteByte(digits[valueByte&15])
	}
	return encoded.String()
}

func normalizeHeaderValue(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func signatureKey(secret, date, region string) []byte {
	dateKey := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	regionKey := hmacSHA256(dateKey, []byte(region))
	serviceKey := hmacSHA256(regionKey, []byte("s3"))
	return hmacSHA256(serviceKey, []byte("aws4_request"))
}

func hmacSHA256(key, value []byte) []byte {
	hash := hmac.New(sha256.New, key)
	_, _ = hash.Write(value)
	return hash.Sum(nil)
}
