package storage

import (
	"bytes"
	"context"
	"net/url"

	"piper_go/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	client *minio.Client
}

func NewS3(c config.S3Conf) (*S3, error) {
	u, err := url.Parse(c.EndpointURL)
	if err != nil {
		return nil, err
	}
	secure := u.Scheme == "https"
	endpoint := u.Host
	accessKey := c.AccessKey
	secretKey := c.SecretKey
	if accessKey == "" {
		accessKey = "admin"
	}
	if secretKey == "" {
		secretKey = "admin"
	}
	cl, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}
	return &S3{client: cl}, nil
}

func (s *S3) EnsureBucket(ctx context.Context, bucket string) error {
	if s == nil || s.client == nil {
		return nil
	}
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		return s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func (s *S3) Put(ctx context.Context, bucket, key string, data []byte, contentType string) error {
	if s == nil || s.client == nil || len(data) == 0 {
		return nil
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := s.client.PutObject(ctx, bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func BucketSource() string {
	return "source"
}
