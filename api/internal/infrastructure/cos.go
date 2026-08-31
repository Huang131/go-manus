package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	appconfig "github.com/mooc-manus/go-manus/api/config"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
	"go.uber.org/zap"
)

// Storage 对象存储接口
type Storage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetURL(ctx context.Context, key string) (string, error)
	Exists(ctx context.Context, key string) (bool, error)
	HealthCheck(ctx context.Context) error
	Close() error
}

// S3Storage AWS S3 兼容的对象存储客户端
type S3Storage struct {
	client   *s3.Client
	bucket   string
	cfg      *appconfig.COSConfig
	endpoint string
}

// NewS3Storage 创建 S3 兼容存储客户端
func NewS3Storage(cfg *appconfig.COSConfig) (*S3Storage, error) {
	logger.Info("Initializing S3-compatible storage connection...")

	if cfg.SecretID == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		logger.Warn("Storage credentials not configured, storage will be disabled")
		return &S3Storage{cfg: cfg, bucket: cfg.Bucket}, nil
	}

	// 构建端点
	endpoint := cfg.Endpoint
	if endpoint == "" {
		// 默认使用腾讯云 COS S3 兼容端点
		endpoint = fmt.Sprintf("https://cos.%s.myqcloud.com", cfg.Region)
	}

	// 创建 AWS 配置
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.SecretID,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 创建 S3 客户端
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // MinIO 需要使用 path-style 访问
	})

	logger.Info("S3-compatible storage connection established",
		zap.String("endpoint", endpoint),
		zap.String("bucket", cfg.Bucket),
	)

	return &S3Storage{
		client:   client,
		bucket:   cfg.Bucket,
		cfg:      cfg,
		endpoint: endpoint,
	}, nil
}

// Close 关闭客户端
func (s *S3Storage) Close() error {
	logger.Info("S3 storage connection closed")
	return nil
}

// HealthCheck 健康检查
func (s *S3Storage) HealthCheck(ctx context.Context) error {
	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	// 通过 HEAD Bucket 请求检查 bucket 是否可访问
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	return err
}

// Upload 上传文件
func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	logger.Debug("S3 Uploading", zap.String("key", key), zap.Int64("size", size))

	// 读取全部内容到内存
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	// 大文件使用分片上传
	if size > 100*1024*1024 { // > 100MB
		return s.uploadLarge(ctx, key, bytes.NewReader(data), size, contentType)
	}

	_, err = s.client.PutObject(ctx, input)
	if err != nil {
		logger.Error("S3 upload failed", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("S3 upload failed: %w", err)
	}

	logger.Info("S3 upload success", zap.String("key", key))
	return nil
}

// uploadLarge 分片上传大文件 (> 100MB)
func (s *S3Storage) uploadLarge(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	// 创建分片上传
	createOutput, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to create multipart upload: %w", err)
	}

	uploadID := *createOutput.UploadId
	const partSize = 10 * 1024 * 1024 // 每个分片 10MB
	var uploadedParts []s3types.CompletedPart

	partNumber := 1
	offset := int64(0)

	for offset < size {
		remaining := size - offset
		readSize := partSize
		if remaining < partSize {
			readSize = int(remaining)
		}

		// 读取分片数据
		chunk := make([]byte, readSize)
		n, err := reader.Read(chunk)
		if err != nil && err != io.EOF {
			// 出错时中止分片上传
			s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
				Bucket:   aws.String(s.bucket),
				Key:      aws.String(key),
				UploadId: aws.String(uploadID),
			})
			return fmt.Errorf("failed to read chunk: %w", err)
		}

		if n == 0 {
			break
		}

		// 上传分片
		uploadPartOutput, err := s.client.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(s.bucket),
			Key:        aws.String(key),
			UploadId:   aws.String(uploadID),
			PartNumber: aws.Int32(int32(partNumber)),
			Body:       bytes.NewReader(chunk[:n]),
		})
		if err != nil {
			s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
				Bucket:   aws.String(s.bucket),
				Key:      aws.String(key),
				UploadId: aws.String(uploadID),
			})
			return fmt.Errorf("failed to upload part %d: %w", partNumber, err)
		}

		uploadedParts = append(uploadedParts, s3types.CompletedPart{
			PartNumber: aws.Int32(int32(partNumber)),
			ETag:       uploadPartOutput.ETag,
		})

		offset += int64(n)
		partNumber++
	}

	// 完成分片上传
	_, err = s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3types.CompletedMultipartUpload{
			Parts: uploadedParts,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

// Download 下载文件
func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if s.client == nil {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	logger.Debug("S3 Downloading", zap.String("key", key))

	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		logger.Error("S3 download failed", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("S3 download failed: %w", err)
	}

	return resp.Body, nil
}

// Delete 删除文件
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	logger.Debug("S3 Deleting", zap.String("key", key))

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		logger.Error("S3 delete failed", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("S3 delete failed: %w", err)
	}

	logger.Info("S3 delete success", zap.String("key", key))
	return nil
}

// GetURL 获取文件的访问 URL
func (s *S3Storage) GetURL(ctx context.Context, key string) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("S3 client not initialized")
	}

	// 如果配置了自定义端点，使用自定义端点
	if s.cfg.Endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", s.cfg.Endpoint, s.bucket, key), nil
	}

	// 默认使用 bucket-style URL
	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", s.bucket, s.cfg.Region, key), nil
}

// Exists 检查文件是否存在
func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("S3 client not initialized")
	}

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// 404 错误表示文件不存在
		return false, nil
	}
	return true, nil
}
