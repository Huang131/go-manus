package infrastructure

import (
	"context"
	"testing"

	appconfig "github.com/Huang131/go-manus/api/config"
)

func TestGenerateURL(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		bucket   string
		region   string
		key      string
		want     string
	}{
		{
			name:     "AWS S3 with region",
			provider: "aws",
			bucket:   "my-bucket",
			region:   "us-east-1",
			key:      "files/test.txt",
			want:     "https://my-bucket.s3.us-east-1.amazonaws.com/files/test.txt",
		},
		{
			name:     "AWS S3 without region",
			provider: "aws",
			bucket:   "my-bucket",
			region:   "",
			key:      "files/test.txt",
			want:     "https://my-bucket.s3.amazonaws.com/files/test.txt",
		},
		{
			name:     "Tencent COS",
			provider: "tencent",
			bucket:   "my-bucket",
			region:   "ap-guangzhou",
			key:      "files/test.txt",
			want:     "https://my-bucket.cos.ap-guangzhou.myqcloud.com/files/test.txt",
		},
		{
			name:     "Tencent COS without region",
			provider: "tencent",
			bucket:   "my-bucket",
			region:   "",
			key:      "files/test.txt",
			want:     "https://my-bucket.cos.ap-guangzhou.myqcloud.com/files/test.txt",
		},
		{
			name:     "Volcengine TOS",
			provider: "volcengine",
			bucket:   "my-bucket",
			region:   "cn-beijing",
			key:      "files/test.txt",
			want:     "https://my-bucket.s3.cn-beijing.volces.com/files/test.txt",
		},
		{
			name:     "Volcengine TOS without region",
			provider: "volcengine",
			bucket:   "my-bucket",
			region:   "",
			key:      "files/test.txt",
			want:     "https://my-bucket.s3.cn-beijing.volces.com/files/test.txt",
		},
		{
			name:     "MinIO",
			provider: "minio",
			bucket:   "my-bucket",
			region:   "",
			key:      "files/test.txt",
			want:     "minio://my-bucket/files/test.txt",
		},
		{
			name:     "Custom provider fallback to AWS format",
			provider: "custom",
			bucket:   "my-bucket",
			region:   "eu-west-1",
			key:      "files/test.txt",
			want:     "https://my-bucket.s3.eu-west-1.amazonaws.com/files/test.txt",
		},
		{
			name:     "Unknown provider fallback to AWS format",
			provider: "",
			bucket:   "my-bucket",
			region:   "us-west-2",
			key:      "files/test.txt",
			want:     "https://my-bucket.s3.us-west-2.amazonaws.com/files/test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateURL(tt.provider, tt.bucket, tt.region, tt.key)
			if got != tt.want {
				t.Errorf("generateURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderDefaultEndpoint(t *testing.T) {
	tests := []struct {
		provider string
		region   string
		want     string
	}{
		{
			provider: "aws",
			region:   "us-east-1",
			want:     "https://s3.us-east-1.amazonaws.com",
		},
		{
			provider: "aws",
			region:   "",
			want:     "https://s3.amazonaws.com",
		},
		{
			provider: "tencent",
			region:   "ap-shanghai",
			want:     "https://cos.ap-shanghai.myqcloud.com",
		},
		{
			provider: "tencent",
			region:   "",
			want:     "https://cos.ap-guangzhou.myqcloud.com",
		},
		{
			provider: "volcengine",
			region:   "cn-beijing",
			want:     "https://s3.cn-beijing.volces.com",
		},
		{
			provider: "volcengine",
			region:   "",
			want:     "https://s3.ap-beijing.volces.com",
		},
		{
			provider: "minio",
			region:   "",
			want:     "",
		},
		{
			provider: "custom",
			region:   "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"-"+tt.region, func(t *testing.T) {
			got := providerDefaultEndpoint(tt.provider, tt.region)
			if got != tt.want {
				t.Errorf("providerDefaultEndpoint(%q, %q) = %v, want %v",
					tt.provider, tt.region, got, tt.want)
			}
		})
	}
}

func TestOSS_GetURL(t *testing.T) {
	tests := []struct {
		name string
		cfg  *appconfig.ObjectStorageConfig
		key  string
		want string
	}{
		{
			name: "Custom endpoint takes precedence",
			cfg: &appconfig.ObjectStorageConfig{
				Provider: "tencent",
				Endpoint: "https://my-custom-endpoint.com",
				Region:   "ap-guangzhou",
				Bucket:   "my-bucket",
			},
			key:  "test.txt",
			want: "https://my-custom-endpoint.com/my-bucket/test.txt",
		},
		{
			name: "Tencent provider generates correct URL",
			cfg: &appconfig.ObjectStorageConfig{
				Provider: "tencent",
				Region:   "ap-guangzhou",
				Bucket:   "my-bucket",
			},
			key:  "test.txt",
			want: "https://my-bucket.cos.ap-guangzhou.myqcloud.com/test.txt",
		},
		{
			name: "AWS provider generates correct URL",
			cfg: &appconfig.ObjectStorageConfig{
				Provider: "aws",
				Region:   "us-east-1",
				Bucket:   "my-bucket",
			},
			key:  "test.txt",
			want: "https://my-bucket.s3.us-east-1.amazonaws.com/test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oss := &OSS{
				cfg:    tt.cfg,
				bucket: tt.cfg.Bucket,
			}
			got, err := oss.GetURL(context.Background(), tt.key)
			if err != nil {
				t.Fatalf("GetURL() error = %v", err)
			}
			if got == "" {
				t.Fatal("GetURL() returned empty string")
			}
			if got != tt.want {
				t.Errorf("GetURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewOSS_WithoutCredentials(t *testing.T) {
	// 测试没有凭证时的行为
	cfg := &appconfig.ObjectStorageConfig{
		Provider:  "tencent",
		SecretID:  "",
		SecretKey: "",
		Bucket:    "test-bucket",
	}

	oss, err := NewOSS(cfg)
	if err != nil {
		t.Fatalf("NewOSS() error = %v", err)
	}

	// 应该返回一个禁用状态的 OSS 实例
	if oss == nil {
		t.Fatal("NewOSS() returned nil")
	}

	// 应该能获取 URL（即使 client 未初始化）
	url, err := oss.GetURL(context.Background(), "test.txt")
	if err != nil {
		t.Fatalf("GetURL() error = %v", err)
	}
	t.Logf("URL without credentials: %s", url)
}
