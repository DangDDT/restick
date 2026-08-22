package application

import (
	"bytes"
	"context"
	"io"

	"github.com/eykh/restick/ftp_svc/internal/domain/objectkey"
	"github.com/eykh/restick/ftp_svc/internal/domain/objectstore"
)

type ObjectService struct {
	store objectstore.ObjectStore
}

func NewObjectService(store objectstore.ObjectStore) *ObjectService {
	return &ObjectService{store: store}
}

func (s *ObjectService) Put(ctx context.Context, key string, data []byte, contentType string) error {
	if int64(len(data)) > objectkey.MaxObjectBytes {
		return objectkey.ErrTooLarge
	}
	sniffed, err := objectkey.Sniff(data)
	if err != nil {
		return err
	}
	if contentType == "" {
		contentType = sniffed
	}
	return s.store.Put(ctx, key, bytes.NewReader(data), int64(len(data)), contentType)
}

func (s *ObjectService) Get(ctx context.Context, key string) ([]byte, string, error) {
	body, _, contentType, err := s.store.Get(ctx, key)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(io.LimitReader(body, objectkey.MaxZipBytes+1))
	if err != nil {
		return nil, "", err
	}
	return data, contentType, nil
}

func (s *ObjectService) List(ctx context.Context, prefix string) ([]string, error) {
	return s.store.List(ctx, prefix)
}

func (s *ObjectService) Store() objectstore.ObjectStore {
	return s.store
}
