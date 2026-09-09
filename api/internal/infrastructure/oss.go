package infrastructure

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appconfig "github.com/Huang131/go-manus/api/config"
	"github.com/Huang131/go-manus/api/pkg/logger"
)

// Storage 对象存储接口，抽象底层存储实现。
type Storage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetURL(ctx context.Context, key string) (string, error)
	Exists(ctx context.Context, key string) (bool, error)
	HealthCheck(ctx context.Context) error
	Close() error
}

// Provider 对象存储服务商枚举。
type Provider string

const (
	ProviderAWS        Provider = "aws"
	ProviderTencent    Provider = "tencent"
	ProviderVolcengine Provider = "volcengine"
	ProviderMinio      Provider = "minio"
	ProviderCustom     Provider = "custom"
)

// providerDefaultEndpoint 根据 Provider 和 Region 返回默认 S3 兼容端点。
func providerDefaultEndpoint(provider, region string) string {
	switch Provider(provider) {
	case ProviderAWS:
		if region == "" {
			return "https://s3.amazonaws.com"
		}
		return fmt.Sprintf("https://s3.%s.amazonaws.com", region)
	case ProviderTencent:
		if region == "" {
			return "https://cos.ap-guangzhou.myqcloud.com"
		}
		return fmt.Sprintf("https://cos.%s.myqcloud.com", region)
	case ProviderVolcengine:
		if region == "" {
			return "https://s3.ap-beijing.volces.com"
		}
		return fmt.Sprintf("https://s3.%s.volces.com", region)
	case ProviderMinio:
		// MinIO 通常是本地或自建，端点必须显式配置
		return ""
	default:
		return ""
	}
}

// generateURL 根据 Provider 生成对象访问 URL。
func generateURL(provider, bucket, region, key string) string {
	switch Provider(provider) {
	case ProviderAWS:
		if region == "" {
			return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key)
		}
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, key)
	case ProviderTencent:
		if region == "" {
			region = "ap-guangzhou"
		}
		return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", bucket, region, key)
	case ProviderVolcengine:
		if region == "" {
			region = "cn-beijing"
		}
		return fmt.Sprintf("https://%s.s3.%s.volces.com/%s", bucket, region, key)
	case ProviderMinio:
		// MinIO 使用 path-style，由 endpoint 决定 URL 格式
		return fmt.Sprintf("minio://%s/%s", bucket, key)
	default:
		// 自定义 Provider 或未识别，按 AWS 格式
		if region != "" {
			return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, key)
		}
		return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key)
	}
}

// OSS AWS S3 兼容的对象存储客户端。
type OSS struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
	cfg      *appconfig.ObjectStorageConfig
	endpoint string
}

// IsReady 表示存储客户端是否已完成可用初始化。
func (s *OSS) IsReady() bool {
	return s != nil && s.client != nil
}

// NewOSS 创建 S3 兼容存储客户端。
// 支持 AWS S3、腾讯云 COS、火山云 TOS、MinIO 等 S3 兼容存储。
func NewOSS(cfg *appconfig.ObjectStorageConfig) (*OSS, error) {
	logger.Info("Initializing S3-compatible storage connection...")

	if cfg.SecretID == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		logger.Warn("Storage credentials not configured, storage will be disabled")
		return &OSS{cfg: cfg, bucket: cfg.Bucket}, nil
	}

	// 构建端点
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = providerDefaultEndpoint(cfg.Provider, cfg.Region)
	}

	if endpoint == "" {
		return nil, fmt.Errorf("endpoint is required: please set endpoint or configure provider")
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
		// MinIO 和部分兼容存储需要 path-style
		o.UsePathStyle = Provider(cfg.Provider) == ProviderMinio || strings.Contains(endpoint, "minio")
	})

	logger.Info("S3-compatible storage connection established",
		logger.String("provider", cfg.Provider),
		logger.String("endpoint", endpoint),
		logger.String("bucket", cfg.Bucket),
	)

	return &OSS{
		client:   client,
		uploader: newUploader(client),
		bucket:   cfg.Bucket,
		cfg:      cfg,
		endpoint: endpoint,
	}, nil
}

// newUploader 创建流式分片上传器。
// 小文件自动走单次 PutObject，大文件自动分片并发上传，避免全量读入内存。
func newUploader(client *s3.Client) *manager.Uploader {
	return manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024 // 每个分片 10MB
		u.Concurrency = 4
	})
}

// Close 关闭客户端。
// aws-sdk-go-v2 的 s3.Client 基于 net/http 连接池，无需也无法显式关闭底层连接；
// 此方法仅为满足 Storage 生命周期接口。
func (s *OSS) Close() error {
	logger.Info("S3 storage connection closed")
	return nil
}

// HealthCheck 健康检查。
func (s *OSS) HealthCheck(ctx context.Context) error {
	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	return err
}

// Upload 上传文件。
// 通过 manager.Uploader 流式上传：数据不整块读入内存，
// 小文件单次 PutObject，大文件自动 10MB 分片并发上传。
func (s *OSS) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	logger.Debug("S3 Uploading", logger.String("key", key), logger.Int64("size", size))

	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
		ContentLength: func() *int64 {
			if size > 0 {
				return aws.Int64(size)
			}
			return nil
		}(),
	})
	if err != nil {
		logger.Error("S3 upload failed", logger.String("key", key), logger.Err(err))
		return fmt.Errorf("S3 upload failed: %w", err)
	}

	logger.Debug("S3 upload success", logger.String("key", key))
	return nil
}

// Download 下载文件。
func (s *OSS) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if s.client == nil {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	logger.Debug("S3 Downloading", logger.String("key", key))

	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		logger.Error("S3 download failed", logger.String("key", key), logger.Err(err))
		return nil, fmt.Errorf("S3 download failed: %w", err)
	}

	return resp.Body, nil
}

// Delete 删除文件。
func (s *OSS) Delete(ctx context.Context, key string) error {
	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	logger.Debug("S3 Deleting", logger.String("key", key))

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		logger.Error("S3 delete failed", logger.String("key", key), logger.Err(err))
		return fmt.Errorf("S3 delete failed: %w", err)
	}

	logger.Info("S3 delete success", logger.String("key", key))
	return nil
}

// GetURL 获取对象的访问 URL。
// URL 格式由配置的 Provider 决定：
//   - aws: https://{bucket}.s3.{region}.amazonaws.com/{key}
//   - tencent: https://{bucket}.cos.{region}.myqcloud.com/{key}
//   - volcengine: https://{bucket}.s3.{region}.volces.com/{key}
//   - minio: 返回 minio://{bucket}/{key}（MinIO 使用 path-style）
func (s *OSS) GetURL(ctx context.Context, key string) (string, error) {
	if s.cfg.Endpoint != "" {
		// 自定义端点直接使用
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(s.cfg.Endpoint, "/"), s.bucket, key), nil
	}

	return generateURL(s.cfg.Provider, s.bucket, s.cfg.Region, key), nil
}

// Exists 检查文件是否存在。
func (s *OSS) Exists(ctx context.Context, key string) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("S3 client not initialized")
	}

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// 404 错误表示文件不存在
		logger.Error("S3 head object failed", logger.String("key", key), logger.Err(err))
		return false, nil
	}
	return true, nil
}
