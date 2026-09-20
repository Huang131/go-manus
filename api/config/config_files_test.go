package config

import (
	"testing"
)

// TestLoadAllConfigFiles 验证三个配置文件都能正确加载并解析 server 配置。
//
// 依赖：本测试会读取磁盘上真实的配置文件（相对 ../，即 api/ 根目录）：
//   - config.yaml
//   - config.test.yaml
//   - config.docker.yaml
//
// 删除或重命名其中任意一个都会导致本测试失败。它守护的契约是
// "三份配置文件的 server 段结构一致且关键字段有效"，因此保留磁盘依赖是必要的。
func TestLoadAllConfigFiles(t *testing.T) {
	tests := []struct {
		file    string
		env     string
		timeout int
	}{
		{"../config.yaml", "development", 30},
		{"../config.test.yaml", "test", 10},
		{"../config.docker.yaml", "development", 30},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			cfg, err := LoadWithValidation(tt.file)
			if err != nil {
				t.Fatalf("加载 %s 失败: %v", tt.file, err)
			}

			// 验证 server 配置
			if cfg.Server.ReadTimeoutSec != tt.timeout {
				t.Errorf("%s: ReadTimeoutSec = %d, want %d", tt.file, cfg.Server.ReadTimeoutSec, tt.timeout)
			}
			if cfg.Server.TrustedProxies == nil || len(cfg.Server.TrustedProxies) == 0 {
				t.Errorf("%s: TrustedProxies 为空，应该配置了回环地址", tt.file)
			}
			if cfg.Server.ShutdownTimeoutSec == 0 {
				t.Errorf("%s: ShutdownTimeoutSec 不应为 0", tt.file)
			}

			// 验证 file_cleanup 配置
			if cfg.FileCleanup.ExpiresAfter != "" {
				t.Logf("%s: file_cleanup.expires_after = %s", tt.file, cfg.FileCleanup.ExpiresAfter)
			}
		})
	}
}
