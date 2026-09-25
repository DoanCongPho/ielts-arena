package storage

import (
	"strings"
	"testing"
	"time"
)

// The presigned-URL example from the AWS SigV4 docs ("Authenticating
// Requests: Using Query Parameters"), which publishes the expected signature.
func TestPresignQuery_AWSExample(t *testing.T) {
	now := time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC)
	got := presignQuery(
		"GET", "examplebucket.s3.amazonaws.com", "/test.txt", "us-east-1",
		"AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		24*time.Hour, now,
	)
	const wantSig = "X-Amz-Signature=aeeed9bbccd4d02ee5c0109b86d86835f995330da4c265957d157751f604d404"
	if !strings.HasSuffix(got, wantSig) {
		t.Errorf("presignQuery() = %s\nwant suffix %s", got, wantSig)
	}
}

func TestPresignGet_PathStyle(t *testing.T) {
	b := &Bucket{
		Endpoint: "https://s3.us-west-004.backblazeb2.com", Region: "us-west-004",
		Name: "ielts-assets", AccessKeyID: "id", SecretAccessKey: "secret",
	}
	got, err := b.PresignGet("/audio/a b.mp3", time.Hour, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	const wantPrefix = "https://s3.us-west-004.backblazeb2.com/ielts-assets/audio/a%20b.mp3?X-Amz-Algorithm=AWS4-HMAC-SHA256&"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("PresignGet() = %s\nwant prefix %s", got, wantPrefix)
	}
	if !strings.Contains(got, "X-Amz-Expires=3600&") {
		t.Errorf("PresignGet() = %s, want X-Amz-Expires=3600", got)
	}
}
