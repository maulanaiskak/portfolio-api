package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ObjectStorage stores public files (images) and returns their public URL.
// Keys are generated server-side, never taken from the client.
type ObjectStorage interface {
	Put(ctx context.Context, key, contentType string, body io.ReadSeeker, size int64) (url string, err error)
}

const immutableCache = "public, max-age=31536000, immutable"

// --- Cloudflare R2 (S3-compatible) ---------------------------------------

type r2Storage struct {
	client        *s3.Client
	bucket        string
	publicBaseURL string
}

func (s *r2Storage) Put(ctx context.Context, key, contentType string, body io.ReadSeeker, size int64) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
		CacheControl:  aws.String(immutableCache),
	})
	if err != nil {
		return "", fmt.Errorf("r2 put %s: %w", key, err)
	}
	return s.publicBaseURL + "/" + key, nil
}

func NewR2Storage(endpoint, accessKeyID, secretAccessKey, bucket, publicBaseURL string) ObjectStorage {
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		UsePathStyle: true,
	})
	return &r2Storage{client: client, bucket: bucket, publicBaseURL: publicBaseURL}
}

// --- Local disk (development) ---------------------------------------------

// LocalStorage writes files under dir and serves them at <baseURL>/uploads/.
// For development only; config refuses it in production.
type LocalStorage struct {
	root    *os.Root
	baseURL string
}

func (s *LocalStorage) Put(_ context.Context, key, _ string, body io.ReadSeeker, _ int64) (string, error) {
	if err := s.root.MkdirAll(path.Dir(key), 0o755); err != nil {
		return "", err
	}
	f, err := s.root.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return s.baseURL + "/uploads/" + key, nil
}

// ServeHTTP serves stored files. os.Root confines lookups to the upload dir
// (no ../ traversal or symlink escapes), and directories are never listed.
func (s *LocalStorage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(r.URL.Path, "/uploads")), "/")
	f, err := s.root.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", immutableCache)
	w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

func NewLocalStorage(dir, publicBaseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, err
	}
	return &LocalStorage{root: root, baseURL: publicBaseURL}, nil
}

// ObjectKey builds a unique, unguessable key like images/2026/10/<uuid>.jpg.
func ObjectKey(prefix, id, ext string, now time.Time) string {
	return fmt.Sprintf("%s/%04d/%02d/%s.%s", prefix, now.Year(), int(now.Month()), id, ext)
}
