package vault

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/vault_config"
)

type Storage interface {
	GetVault(fileName string) ([]byte, error)
	SaveVault(fileName string, content []byte) error
	Exist(fileName string) (bool, error)
	DeleteFile(fileName string) error
}

type BlockStorageImp struct {
	cfg      vault_config.BlockStorage
	session  *session.Session
	s3Client *s3.S3
	logger   *logrus.Logger
}

var _ Storage = (*BlockStorageImp)(nil)

func NewBlockStorageImp(cfg vault_config.BlockStorage) (*BlockStorageImp, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(cfg.Region),
		Endpoint:         aws.String(cfg.Host),
		Credentials:      credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	return &BlockStorageImp{
		cfg:      cfg,
		session:  sess,
		s3Client: s3.New(sess),
		logger:   logrus.WithField("module", "block_storage").Logger,
	}, nil
}

func (bs *BlockStorageImp) GetVault(fileName string) ([]byte, error) {
	return bs.GetFile(fileName)
}

func (bs *BlockStorageImp) SaveVault(file string, content []byte) error {
	return bs.UploadFileWithRetry(content, file, 3)
}
func (bs *BlockStorageImp) Exist(fileName string) (bool, error) {
	return bs.FileExist(fileName)
}
func (bs *BlockStorageImp) FileExist(fileName string) (bool, error) {
	_, err := bs.s3Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bs.cfg.Bucket),
		Key:    aws.String(fileName),
	})
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			if aerr.Code() == s3.ErrCodeNoSuchKey || aerr.Code() == "NotFound" {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check if file exists: %w", err)
	}
	return true, nil
}
func (bs *BlockStorageImp) UploadFileWithRetry(fileContent []byte, fileName string, retry int) error {
	var err error
	for i := 0; i < retry; i++ {
		err = bs.UploadFile(fileContent, fileName)
		if err == nil {
			return nil
		}
		bs.logger.Error(err)
	}
	return err
}
func (bs *BlockStorageImp) UploadFile(fileContent []byte, fileName string) error {
	bs.logger.Infoln("upload file", fileName, "bucket", bs.cfg.Bucket, "content length", len(fileContent))
	output, err := bs.s3Client.PutObjectWithContext(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(bs.cfg.Bucket),
		Key:           aws.String(fileName),
		Body:          aws.ReadSeekCloser(bytes.NewReader(fileContent)),
		ContentLength: aws.Int64(int64(len(fileContent))),
	})
	if err != nil {
		bs.logger.Error(err)
		return err
	}
	if output != nil {
		bs.logger.Infof("upload file %s success, version id: %s", fileName, aws.StringValue(output.VersionId))
	}
	return nil
}

func (bs *BlockStorageImp) GetFile(fileName string) ([]byte, error) {
	bs.logger.Infoln("get file", fileName, "bucket", bs.cfg.Bucket)
	output, err := bs.s3Client.GetObjectWithContext(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bs.cfg.Bucket),
		Key:    aws.String(fileName),
	})
	if err != nil {
		bs.logger.Error("error getting file: ", err)
		return nil, err
	}
	defer func() {
		if err := output.Body.Close(); err != nil {
			bs.logger.Error(err)
		}
	}()
	return io.ReadAll(output.Body)
}
func (bs *BlockStorageImp) DeleteFile(fileName string) error {
	_, err := bs.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bs.cfg.Bucket),
		Key:    aws.String(fileName),
	})
	if err != nil {
		bs.logger.Error(err)
		return err
	}
	bs.logger.Infof("delete file %s success", fileName)
	return nil
}

var _ Storage = (*LocalVaultStorage)(nil)

type LocalVaultStorageConfig struct {
	VaultFilePath string `mapstructure:"vault_file_path" json:"vault_file_path"`
}

type LocalVaultStorage struct {
	cfg LocalVaultStorageConfig
}

func NewLocalVaultStorage(cfg LocalVaultStorageConfig) (*LocalVaultStorage, error) {
	return &LocalVaultStorage{
		cfg: cfg,
	}, nil
}
func (lvs *LocalVaultStorage) GetVault(fileName string) ([]byte, error) {
	filePathName := filepath.Join(lvs.cfg.VaultFilePath, fileName)
	if _, err := os.Stat(filePathName); err != nil {
		return nil, fmt.Errorf("os.Stat failed: %s: %w", err, os.ErrNotExist)
	}
	content, err := os.ReadFile(filePathName)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile failed: %s: %w", err, os.ErrPermission)
	}
	return content, nil
}
func (lvs *LocalVaultStorage) Exist(fileName string) (bool, error) {
	filePathName := filepath.Join(lvs.cfg.VaultFilePath, fileName)
	if _, err := os.Stat(filePathName); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("os.Stat failed: %s: %w", err, os.ErrPermission)
	}
	return true, nil
}
func (lvs *LocalVaultStorage) SaveVault(file string, content []byte) error {
	filePathName := filepath.Join(lvs.cfg.VaultFilePath, file)
	if err := os.MkdirAll(filepath.Dir(filePathName), 0o666); err != nil {
		return fmt.Errorf("os.MkdirAll failed: %s: %w", err, os.ErrPermission)
	}
	if err := os.WriteFile(filePathName, content, 0o666); err != nil {
		return fmt.Errorf("os.WriteFile failed: %s: %w", err, os.ErrPermission)
	}
	return nil
}
func (lvs *LocalVaultStorage) DeleteFile(fileName string) error {
	filePathName := filepath.Join(lvs.cfg.VaultFilePath, fileName)
	if err := os.Remove(filePathName); err != nil {
		return fmt.Errorf("os.Remove failed: %w", err)
	}
	return nil
}
