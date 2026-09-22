// Package storage hands out time-limited links to objects in a private
// S3-compatible bucket (Backblaze B2, Cloudflare R2, AWS S3).
//
// Only GET presigning is needed — uploads happen out of band with
// scripts/upload_assets.sh — so this signs AWS SigV4 query strings by hand
// rather than pulling in an SDK.
package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Bucket struct {
	Endpoint        string // e.g. https://s3.us-west-004.backblazeb2.com
	Region          string // e.g. us-west-004
	Name            string
	AccessKeyID     string
	SecretAccessKey string
}

// PresignGet returns a URL that fetches key from the bucket until expires
// has passed. It uses path-style addressing: <endpoint>/<bucket>/<key>.
func (b *Bucket) PresignGet(key string, expires time.Duration, now time.Time) (string, error) {
	u, err := url.Parse(b.Endpoint)
	if err != nil {
		return "", fmt.Errorf("storage endpoint: %w", err)
	}
	path := "/" + b.Name + "/" + strings.TrimPrefix(key, "/")
	query := presignQuery(u.Host, path, b.Region, b.AccessKeyID, b.SecretAccessKey, expires, now)
	return u.Scheme + "://" + u.Host + encodePath(path) + "?" + query, nil
}

// presignQuery computes the SigV4 query string (including X-Amz-Signature)
// for an unsigned-payload GET of path on host.
func presignQuery(host, path, region, accessKeyID, secret string, expires time.Duration, now time.Time) string {
	now = now.UTC()
	amzDate := now.Format("20060102T150405Z")
	day := now.Format("20060102")
	scope := day + "/" + region + "/s3/aws4_request"

	q := url.Values{}
	q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	q.Set("X-Amz-Credential", accessKeyID+"/"+scope)
	q.Set("X-Amz-Date", amzDate)
	q.Set("X-Amz-Expires", fmt.Sprint(int(expires.Seconds())))
	q.Set("X-Amz-SignedHeaders", "host")
	// Encode sorts by key, which is the canonical order SigV4 wants.
	canonicalQuery := strings.ReplaceAll(q.Encode(), "+", "%20")

	canonicalRequest := strings.Join([]string{
		"GET",
		encodePath(path),
		canonicalQuery,
		"host:" + host + "\n",
		"host",
		"UNSIGNED-PAYLOAD",
	}, "\n")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hexSHA256(canonicalRequest),
	}, "\n")

	key := hmacSHA256([]byte("AWS4"+secret), day)
	key = hmacSHA256(key, region)
	key = hmacSHA256(key, "s3")
	key = hmacSHA256(key, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(key, stringToSign))

	return canonicalQuery + "&X-Amz-Signature=" + signature
}

// encodePath percent-encodes each segment the way S3 canonicalises it:
// everything but unreserved characters, keeping the slashes.
func encodePath(p string) string {
	var b strings.Builder
	for _, c := range []byte(p) {
		switch {
		case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z', '0' <= c && c <= '9',
			c == '-', c == '_', c == '.', c == '~', c == '/':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func hexSHA256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
