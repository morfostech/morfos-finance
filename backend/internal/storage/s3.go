package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/morfostech/morfos-finance/internal/config"
)

// s3Storage stores objects in an S3-compatible bucket (Cloudflare R2 by
// default). Uses path-style addressing for maximum compatibility.
type s3Storage struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

func newS3(cfg config.StorageConfig) (*s3Storage, error) {
	endpoint, secure, err := splitEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: secure,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("init s3 client: %w", err)
	}
	return &s3Storage{
		client:        client,
		bucket:        cfg.Bucket,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

func (s *s3Storage) Put(ctx context.Context, key, contentType string, data io.Reader, size int64) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}
	if s.publicBaseURL != "" {
		return s.publicBaseURL + "/" + key, nil
	}
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(s.client.EndpointURL().String(), "/"), s.bucket, key), nil
}

func (s *s3Storage) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

// splitEndpoint turns a URL like "https://acc.r2.cloudflarestorage.com" into the
// host:port and TLS flag minio expects.
func splitEndpoint(raw string) (host string, secure bool, err error) {
	if !strings.Contains(raw, "://") {
		return raw, true, nil // bare host, assume TLS
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false, fmt.Errorf("parse s3 endpoint: %w", err)
	}
	return u.Host, u.Scheme == "https", nil
}
