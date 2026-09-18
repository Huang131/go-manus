package agent

import (
	"context"
	"io"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/repository"
)

// Repositories 聚合 Agent 服务所需的数据访问依赖。
// 按职责归组后，服务只持有这一个聚合值，避免字段膨胀。
type Repositories struct {
	Session  repository.SessionRepository
	File     repository.FileRepository
	LLMModel repository.LLMModelRepository
}

// Capabilities 聚合 Agent 运行期可调用的外部能力与基础设施。
//
// 这些是"可替换的运行时协作者"（LLM、沙箱、浏览器、搜索、存储、消息队列），
// 与数据访问（Repositories）分属不同层次，收敛后避免服务平铺十几个字段。
type Capabilities struct {
	LLM          external.LLM
	Sandbox      external.Sandbox
	Browser      external.Browser
	SearchEngine external.SearchEngine
	FileStorage  COSFileStorage
	MessageQueue external.TaskMessageQueue
}

// COSFileStorage 对象存储最小接口，供文件同步与附件下载使用。
type COSFileStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	GetURL(ctx context.Context, key string) (string, error)
}
