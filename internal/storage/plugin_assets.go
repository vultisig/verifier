package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
)

type ObjectMetadata struct {
	ContentType   string
	ContentLength int64
}

type PluginAssetStorage interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetPublicURL(key string) string
	PresignPut(ctx context.Context, key, contentType string, expiry time.Duration) (string, error)
	HeadObject(ctx context.Context, key string) (*ObjectMetadata, error)
	GetObjectRange(ctx context.Context, key string, rangeStart, rangeEnd int64) ([]byte, error)
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	Copy(ctx context.Context, srcKey, dstKey string) error
}

type S3PluginAssetStorage struct {
	cfg      config.PluginAssetsConfig
	s3Client *s3.S3
	logger   *logrus.Logger
}

var _ PluginAssetStorage = (*S3PluginAssetStorage)(nil)

func NewS3PluginAssetStorage(cfg config.PluginAssetsConfig) (*S3PluginAssetStorage, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(cfg.Region),
		Endpoint:         aws.String(cfg.Host),
		Credentials:      credentials.NewStaticCredentials(cfg.AccessKey, cfg.Secret, ""),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 session: %w", err)
	}

	return &S3PluginAssetStorage{
		cfg:      cfg,
		s3Client: s3.New(sess),
		logger:   logrus.WithField("module", "plugin_assets").Logger,
	}, nil
}

func (s *S3PluginAssetStorage) Upload(ctx context.Context, key string, data []byte, contentType string) error {
	s.logger.Infof("uploading plugin asset: %s, bucket: %s, size: %d", key, s.cfg.Bucket, len(data))

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.cfg.Bucket),
		Key:           aws.String(key),
		Body:          aws.ReadSeekCloser(bytes.NewReader(data)),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String(contentType),
		ACL:           aws.String("public-read"),
	}

	_, err := s.s3Client.PutObjectWithContext(ctx, input)
	if err != nil {
		s.logger.Errorf("failed to upload plugin asset %s: %v", key, err)
		return fmt.Errorf("failed to upload plugin asset: %w", err)
	}

	s.logger.Infof("plugin asset uploaded: %s", key)
	return nil
}

func (s *S3PluginAssetStorage) Delete(ctx context.Context, key string) error {
	s.logger.Infof("deleting plugin asset: %s, bucket: %s", key, s.cfg.Bucket)

	_, err := s.s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		s.logger.Errorf("failed to delete plugin asset %s: %v", key, err)
		return fmt.Errorf("failed to delete plugin asset: %w", err)
	}

	s.logger.Infof("plugin asset deleted: %s", key)
	return nil
}

func (s *S3PluginAssetStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.s3Client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			if aerr.Code() == s3.ErrCodeNoSuchKey || aerr.Code() == "NotFound" {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check if plugin asset exists: %w", err)
	}
	return true, nil
}

func (s *S3PluginAssetStorage) GetPublicURL(key string) string {
	return fmt.Sprintf("%s/%s", s.cfg.EffectivePublicBaseURL(), key)
}

func (s *S3PluginAssetStorage) PresignPut(ctx context.Context, key, contentType string, expiry time.Duration) (string, error) {
	req, _ := s.s3Client.PutObjectRequest(&s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	})
	if req.Error != nil {
		return "", fmt.Errorf("failed to create PUT request: %w", req.Error)
	}
	url, err := req.Presign(expiry)
	if err != nil {
		return "", fmt.Errorf("failed to presign PUT URL: %w", err)
	}
	return url, nil
}

func (s *S3PluginAssetStorage) HeadObject(ctx context.Context, key string) (*ObjectMetadata, error) {
	out, err := s.s3Client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			if aerr.Code() == s3.ErrCodeNoSuchKey || aerr.Code() == "NotFound" {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("failed to head object: %w", err)
	}
	return &ObjectMetadata{
		ContentType:   aws.StringValue(out.ContentType),
		ContentLength: aws.Int64Value(out.ContentLength),
	}, nil
}

func (s *S3PluginAssetStorage) GetObjectRange(ctx context.Context, key string, rangeStart, rangeEnd int64) ([]byte, error) {
	rangeHeader := fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd)
	out, err := s.s3Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
		Range:  aws.String(rangeHeader),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object range: %w", err)
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (s *S3PluginAssetStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.s3Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return out.Body, nil
}

func (s *S3PluginAssetStorage) Copy(ctx context.Context, srcKey, dstKey string) error {
	s.logger.Infof("copying plugin asset: %s -> %s, bucket: %s", srcKey, dstKey, s.cfg.Bucket)

	copySource := fmt.Sprintf("%s/%s", s.cfg.Bucket, url.PathEscape(srcKey))
	_, err := s.s3Client.CopyObjectWithContext(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.cfg.Bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(dstKey),
		ACL:        aws.String("public-read"),
	})
	if err != nil {
		s.logger.Errorf("failed to copy plugin asset %s -> %s: %v", srcKey, dstKey, err)
		return fmt.Errorf("failed to copy plugin asset: %w", err)
	}

	s.logger.Infof("plugin asset copied: %s -> %s", srcKey, dstKey)
	return nil
}
