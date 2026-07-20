package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/morfostech/morfos-finance/internal/config"
)

// s3Storage stores objects in an S3-compatible bucket. The AWS client supports
// endpoint path prefixes such as Supabase's required /storage/v1/s3.
type s3Storage struct {
	client        *awss3.Client
	endpoint      string
	bucket        string
	publicBaseURL string
}

func newS3(cfg config.StorageConfig) (*s3Storage, error) {
	endpoint, err := normalizeEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	awsCfg := aws.Config{
		Region:                     cfg.Region,
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	}
	client := awss3.NewFromConfig(awsCfg, func(options *awss3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})

	return &s3Storage{
		client:        client,
		endpoint:      endpoint,
		bucket:        cfg.Bucket,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

func (s *s3Storage) Put(ctx context.Context, key, contentType string, data io.Reader, size int64) (string, error) {
	_, err := s.client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          data,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}
	if s.publicBaseURL != "" {
		return s.publicBaseURL + "/" + key, nil
	}
	return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, key), nil
}

func (s *s3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func normalizeEndpoint(raw string) (string, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(raw), "/")
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("parse s3 endpoint: %q", raw)
	}
	return endpoint, nil
}
