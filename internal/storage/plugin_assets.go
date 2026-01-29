package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
)

type PluginAssetStorage interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetPublicURL(key string) string
}

type S3PluginAssetStorage struct {
	cfg      config.PluginAssetsConfig
	s3Client *s3.S3
	logger   *logrus.Logger
}

var _ PluginAssetStorage = (*S3PluginAssetStorage)(nil)

func NewS3PluginAssetStorage(cfg config.PluginAssetsConfig) (*S3PluginAssetStorage, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("plugin assets bucket not configured")
	}

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
