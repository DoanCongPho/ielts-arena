package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDiskStore_SignedUploadThenDownload(t *testing.T) {
	d := NewDiskStore(t.TempDir(), "/media/", []byte("secret"))
	srv := httptest.NewServer(http.StripPrefix("/media", d.Handler()))
	defer srv.Close()

	up, err := d.UploadURL("speaking/1/a.webm", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPut, srv.URL+up, strings.NewReader("audio"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT = %v, %v", resp, err)
	}

	got, err := d.Get(context.Background(), "speaking/1/a.webm")
	if err != nil || string(got) != "audio" {
		t.Fatalf("Get = %q, %v", got, err)
	}

	// An upload link is not a download link.
	resp, _ = http.Get(srv.URL + up)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("GET with a PUT signature = %d, want 403", resp.StatusCode)
	}
	down, _ := d.DownloadURL("speaking/1/a.webm", time.Minute)
	resp, _ = http.Get(srv.URL + down)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET = %d, want 200", resp.StatusCode)
	}
}

func TestDiskStore_ExpiredAndEscapingKeys(t *testing.T) {
	d := NewDiskStore(t.TempDir(), "/media/", []byte("secret"))
	srv := httptest.NewServer(http.StripPrefix("/media", d.Handler()))
	defer srv.Close()

	expired, _ := d.DownloadURL("x.mp3", -time.Minute)
	resp, _ := http.Get(srv.URL + expired)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expired link = %d, want 403", resp.StatusCode)
	}
	if _, err := d.UploadURL("../etc/passwd", time.Minute); err == nil {
		t.Error("UploadURL accepted a key escaping the root")
	}
	if ok, _ := d.Exists(context.Background(), "missing.mp3"); ok {
		t.Error("Exists(missing) = true")
	}
}
