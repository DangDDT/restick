package minio

import (
	"context"
	"io"
	"strings"

	"github.com/eykh/restick/ftp_svc/internal/domain/objectstore"
	miniolib "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store struct {
	client *miniolib.Client
	bucket string
}

func New(endpoint string, accessKey string, secretKey string, bucket string, useSSL bool) (*Store, error) {
	client, err := miniolib.New(endpoint, &miniolib.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &Store{client: client, bucket: bucket}, nil
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, miniolib.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, int64, string, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, miniolib.GetObjectOptions{})
	if err != nil {
		return nil, 0, "", err
	}
	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, 0, "", err
	}
	contentType := info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return obj, info.Size, contentType, nil
}

func (s *Store) List(ctx context.Context, prefix string) ([]string, error) {
	keys := make([]string, 0)
	for object := range s.client.ListObjects(ctx, s.bucket, miniolib.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if object.Err != nil {
			return nil, object.Err
		}
		if object.Key != "" {
			keys = append(keys, object.Key)
		}
		if len(keys) >= 500 {
			break
		}
	}
	return keys, nil
}

func (s *Store) Stat(ctx context.Context, key string) (int64, string, string, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, miniolib.StatObjectOptions{})
	if err != nil {
		return 0, "", "", err
	}
	etag := strings.Trim(info.ETag, "\"")
	return info.Size, etag, info.ContentType, nil
}

var _ objectstore.ObjectStore = (*Store)(nil)
