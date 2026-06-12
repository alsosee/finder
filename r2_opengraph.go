package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type R2OpenGraphUploader struct {
	accountID       string
	accessKeyID     string
	accessKeySecret string
	bucket          string
	client          *http.Client
	now             func() time.Time
}

func (u R2OpenGraphUploader) Upload(key string, body []byte, contentType string) error {
	client := u.client
	if client == nil {
		client = http.DefaultClient
	}
	now := time.Now().UTC()
	if u.now != nil {
		now = u.now().UTC()
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s/%s", u.accountID, escapePathSegment(u.bucket), escapeObjectKey(key))
	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating R2 request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-amz-content-sha256", sha256Hex(body))
	req.Header.Set("x-amz-date", now.Format("20060102T150405Z"))

	u.sign(req, body, now)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("uploading to R2: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("uploading to R2: status %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func (u R2OpenGraphUploader) sign(req *http.Request, body []byte, now time.Time) {
	date := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	scope := date + "/auto/s3/aws4_request"

	canonicalHeaders := "host:" + req.URL.Host + "\n" +
		"x-amz-content-sha256:" + sha256Hex(body) + "\n" +
		"x-amz-date:" + amzDate + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		"",
		canonicalHeaders,
		signedHeaders,
		sha256Hex(body),
	}, "\n")

	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := awsSigningKey(u.accessKeySecret, date, "auto", "s3")
	signature := hmacSHA256Hex(signingKey, stringToSign)
	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		u.accessKeyID,
		scope,
		signedHeaders,
		signature,
	))
}

func awsSigningKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func hmacSHA256Hex(key []byte, data string) string {
	return hex.EncodeToString(hmacSHA256(key, data))
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func escapeObjectKey(key string) string {
	parts := strings.Split(path.Clean(filepathSlash(key)), "/")
	for i := range parts {
		parts[i] = escapePathSegment(parts[i])
	}
	return strings.Join(parts, "/")
}

func escapePathSegment(segment string) string {
	return strings.ReplaceAll(url.PathEscape(segment), "+", "%20")
}

func filepathSlash(key string) string {
	return strings.ReplaceAll(key, "\\", "/")
}
