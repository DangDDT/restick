package application

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"path"

	"github.com/eykh/restick/ftp_svc/internal/domain/objectkey"
)

func (s *ObjectService) BuildBundle(ctx context.Context, packID string, version int, keys []string) (string, error) {
	if len(keys) == 0 || len(keys) > 200 {
		return "", objectkey.ErrInvalidKey
	}
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	for _, key := range keys {
		if err := objectkey.ValidateProcessed(key); err != nil {
			_ = writer.Close()
			return "", err
		}
		body, _, _, err := s.store.Get(ctx, key)
		if err != nil {
			_ = writer.Close()
			return "", err
		}
		name := zipName(key)
		entry, err := writer.Create(name)
		if err != nil {
			_ = body.Close()
			_ = writer.Close()
			return "", err
		}
		_, err = io.Copy(entry, io.LimitReader(body, objectkey.MaxObjectBytes))
		_ = body.Close()
		if err != nil {
			_ = writer.Close()
			return "", err
		}
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	if buffer.Len() > objectkey.MaxZipBytes {
		return "", objectkey.ErrTooLarge
	}
	bundleKey := objectkey.BundleKey(packID, version)
	err := s.store.Put(ctx, bundleKey, bytes.NewReader(buffer.Bytes()), int64(buffer.Len()), "application/zip")
	return bundleKey, err
}

func zipName(key string) string {
	base := path.Base(key)
	if len(base) >= 6 && base[:6] == "thumb_" {
		return path.Join("thumbs", base)
	}
	return base
}
