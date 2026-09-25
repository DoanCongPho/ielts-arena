package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound is returned by Get for a key that holds no object.
var ErrNotFound = errors.New("object not found")

// ObjectStore holds user recordings and generated examiner audio. The
// browser never sees credentials: it uploads and downloads through
// short-lived links, and the server reads and writes objects directly.
type ObjectStore interface {
	// UploadURL is a link the browser PUTs one object's bytes to.
	UploadURL(key string, expires time.Duration) (string, error)
	// DownloadURL is a link anyone holding it can GET the object from.
	DownloadURL(key string, expires time.Duration) (string, error)
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key, contentType string, body []byte) error
	Exists(ctx context.Context, key string) (bool, error)
}

// maxObjectBytes caps one object read or written through the store. A
// 2-minute Opus recording is well under 1 MB; examiner lines are smaller.
const maxObjectBytes = 25 << 20

// --- bucket ---------------------------------------------------------------

type bucketStore struct {
	b    *Bucket
	http *http.Client
}

// NewBucketStore stores objects in an S3-compatible bucket through
// presigned links, so it needs no SDK.
func NewBucketStore(b *Bucket) ObjectStore {
	return &bucketStore{b: b, http: &http.Client{Timeout: 60 * time.Second}}
}

func (s *bucketStore) UploadURL(key string, expires time.Duration) (string, error) {
	return s.b.PresignPut(key, expires, time.Now())
}

func (s *bucketStore) DownloadURL(key string, expires time.Duration) (string, error) {
	return s.b.PresignGet(key, expires, time.Now())
}

func (s *bucketStore) Get(ctx context.Context, key string) ([]byte, error) {
	resp, err := s.do(ctx, http.MethodGet, key, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, maxObjectBytes))
}

func (s *bucketStore) Put(ctx context.Context, key, contentType string, body []byte) error {
	resp, err := s.do(ctx, http.MethodPut, key, contentType, body)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (s *bucketStore) Exists(ctx context.Context, key string) (bool, error) {
	resp, err := s.do(ctx, http.MethodHead, key, "", nil)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, resp.Body.Close()
}

// do sends one presigned request and turns a non-2xx status into an error,
// closing the body in that case.
func (s *bucketStore) do(ctx context.Context, method, key, contentType string, body []byte) (*http.Response, error) {
	link, err := s.b.presign(method, key, 15*time.Minute, time.Now())
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, link, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, key, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("%s %s: %w", method, key, ErrNotFound)
	}
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, fmt.Errorf("%s %s: status %d: %s", method, key, resp.StatusCode, msg)
	}
	return resp, nil
}

// --- disk -----------------------------------------------------------------

// DiskStore keeps objects under a directory, for local development without
// a bucket. Its links point back at this server (see Handler) and carry an
// HMAC over method, key and expiry, so they behave like presigned links.
type DiskStore struct {
	root   string
	prefix string // URL path Handler is mounted at, e.g. "/media/"
	secret []byte
}

func NewDiskStore(root, prefix string, secret []byte) *DiskStore {
	return &DiskStore{root: root, prefix: prefix, secret: secret}
}

func (d *DiskStore) UploadURL(key string, expires time.Duration) (string, error) {
	return d.link(http.MethodPut, key, expires)
}

func (d *DiskStore) DownloadURL(key string, expires time.Duration) (string, error) {
	return d.link(http.MethodGet, key, expires)
}

func (d *DiskStore) Get(_ context.Context, key string) ([]byte, error) {
	p, err := d.path(key)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("get %s: %w", key, ErrNotFound)
	}
	return b, err
}

func (d *DiskStore) Put(_ context.Context, key, _ string, body []byte) error {
	p, err := d.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, body, 0o644)
}

func (d *DiskStore) Exists(_ context.Context, key string) (bool, error) {
	p, err := d.path(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Handler serves GET and PUT on links made by UploadURL/DownloadURL. It
// expects the prefix already stripped from the path.
func (d *DiskStore) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/")
		exp, err := strconv.ParseInt(r.URL.Query().Get("exp"), 10, 64)
		if err != nil || time.Now().Unix() > exp ||
			!hmac.Equal([]byte(r.URL.Query().Get("sig")), []byte(d.sign(r.Method, key, exp))) {
			http.Error(w, "link invalid or expired", http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodGet:
			b, err := d.Get(r.Context(), key)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.ServeContent(w, r, filepath.Base(key), time.Time{}, bytes.NewReader(b))
		case http.MethodPut:
			body, err := io.ReadAll(io.LimitReader(r.Body, maxObjectBytes+1))
			if err != nil || len(body) > maxObjectBytes {
				http.Error(w, "upload too large", http.StatusRequestEntityTooLarge)
				return
			}
			if err := d.Put(r.Context(), key, r.Header.Get("Content-Type"), body); err != nil {
				http.Error(w, "store failed", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func (d *DiskStore) link(method, key string, expires time.Duration) (string, error) {
	if _, err := d.path(key); err != nil {
		return "", err
	}
	exp := time.Now().Add(expires).Unix()
	return fmt.Sprintf("%s%s?exp=%d&sig=%s", d.prefix, encodePath(key), exp, d.sign(method, key, exp)), nil
}

func (d *DiskStore) sign(method, key string, exp int64) string {
	h := hmac.New(sha256.New, d.secret)
	fmt.Fprintf(h, "%s\n%s\n%d", method, key, exp)
	return hex.EncodeToString(h.Sum(nil))
}

// path resolves key inside root, refusing anything that would escape it.
func (d *DiskStore) path(key string) (string, error) {
	clean := filepath.Clean("/" + key)
	if clean == "/" || strings.Contains(key, "..") {
		return "", fmt.Errorf("invalid object key %q", key)
	}
	return filepath.Join(d.root, clean), nil
}
