// Package storage talks to the S3-compatible object store (Garage).
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config locates the bucket. Endpoint is used for uploads (inside Docker:
// http://garage:3900); PublicEndpoint is the host browsers reach, and
// presigned URLs are signed for it because SigV4 covers the Host header.
type Config struct {
	Endpoint        string
	PublicEndpoint  string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	PresignTTL      time.Duration
}

// Client uploads objects and presigns downloads.
type Client struct {
	s3      *s3.Client
	presign *s3.PresignClient
	bucket  string
	ttl     time.Duration
}

// New returns a Client. It makes no network calls.
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, errors.New("storage: endpoint, bucket and credentials are required")
	}
	public := cfg.PublicEndpoint
	if public == "" {
		public = cfg.Endpoint
	}
	return &Client{
		s3:      newS3(cfg, cfg.Endpoint),
		presign: s3.NewPresignClient(newS3(cfg, public)),
		bucket:  cfg.Bucket,
		ttl:     cfg.PresignTTL,
	}, nil
}

func newS3(cfg Config, endpoint string) *s3.Client {
	return s3.New(s3.Options{
		Region:       cfg.Region,
		BaseEndpoint: aws.String(endpoint),
		UsePathStyle: true,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		// Only send checksums when an operation requires them; keeps the
		// client compatible with Garage across SDK default changes.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
}

// Put uploads data under key.
func (c *Client) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(c.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return fmt.Errorf("storage: put %s: %w", key, err)
	}
	return nil
}

// Delete removes key; deleting a missing key is not an error.
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(c.bucket), Key: aws.String(key)})
	if err != nil {
		return fmt.Errorf("storage: delete %s: %w", key, err)
	}
	return nil
}

// PresignGet returns a time-limited download URL for key and its lifetime.
func (c *Client) PresignGet(ctx context.Context, key string) (string, time.Duration, error) {
	req, err := c.presign.PresignGetObject(ctx,
		&s3.GetObjectInput{Bucket: aws.String(c.bucket), Key: aws.String(key)},
		s3.WithPresignExpires(c.ttl),
	)
	if err != nil {
		return "", 0, fmt.Errorf("storage: presign %s: %w", key, err)
	}
	return req.URL, c.ttl, nil
}
