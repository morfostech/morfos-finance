package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/morfostech/morfos-finance/internal/config"
)

func TestS3PreservesEndpointPathForPutAndDelete(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") == "" {
			t.Error("request is not signed")
		}
		if r.Header.Get("X-Amz-Sdk-Checksum-Algorithm") != "" {
			t.Error("optional AWS checksum is incompatible with generic S3 providers")
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store, err := newS3(config.StorageConfig{
		Endpoint:      server.URL + "/storage/v1/s3",
		Bucket:        "morfos-finance",
		AccessKey:     "access-key",
		SecretKey:     "secret-key",
		Region:        "ca-central-1",
		PublicBaseURL: "https://files.example/morfos-finance",
	})
	if err != nil {
		t.Fatal(err)
	}

	key := "propostas/1/proposal.docx"
	objectURL, err := store.Put(context.Background(), key, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", strings.NewReader("document"), 8)
	if err != nil {
		t.Fatal(err)
	}
	if objectURL != "https://files.example/morfos-finance/"+key {
		t.Fatalf("object URL = %q", objectURL)
	}
	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"PUT /storage/v1/s3/morfos-finance/" + key,
		"DELETE /storage/v1/s3/morfos-finance/" + key,
	}
	if len(requests) != len(want) {
		t.Fatalf("requests = %v", requests)
	}
	for i := range want {
		if requests[i] != want[i] {
			t.Fatalf("request %d = %q, want %q", i, requests[i], want[i])
		}
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	endpoint, err := normalizeEndpoint("storage.example.com/storage/v1/s3/")
	if err != nil {
		t.Fatal(err)
	}
	if endpoint != "https://storage.example.com/storage/v1/s3" {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

func TestS3Integration(t *testing.T) {
	if os.Getenv("S3_INTEGRATION") != "1" {
		t.Skip("set S3_INTEGRATION=1 to test configured object storage")
	}

	store, err := newS3(config.StorageConfig{
		Endpoint:      os.Getenv("S3_ENDPOINT"),
		Bucket:        os.Getenv("S3_BUCKET"),
		AccessKey:     os.Getenv("S3_ACCESS_KEY_ID"),
		SecretKey:     os.Getenv("S3_SECRET_ACCESS_KEY"),
		Region:        os.Getenv("S3_REGION"),
		PublicBaseURL: os.Getenv("S3_PUBLIC_BASE_URL"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.client.ListBuckets(context.Background(), &awss3.ListBucketsInput{}); err != nil {
		t.Fatalf("list buckets: %v", err)
	}
	if _, err := store.client.HeadBucket(context.Background(), &awss3.HeadBucketInput{Bucket: aws.String(store.bucket)}); err != nil {
		t.Fatalf("head bucket: %v", err)
	}

	key := fmt.Sprintf("healthchecks/storage-%d.txt", time.Now().UnixNano())
	content := "morfos-finance-storage-check"
	if _, err := store.Put(context.Background(), key, "text/plain", strings.NewReader(content), int64(len(content))); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}
}
