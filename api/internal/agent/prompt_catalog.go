package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// PromptName 是可持久化的稳定 Prompt 标识。
type PromptName string

const (
	PromptSystem        PromptName = "system"
	PromptPlannerSystem PromptName = "planner_system"
	PromptCreatePlan    PromptName = "create_plan"
	PromptUpdatePlan    PromptName = "update_plan"
	PromptReActSystem   PromptName = "react_system"
	PromptExecution     PromptName = "execution"
	PromptSummarize     PromptName = "summarize"
)

// PromptDefinition 描述一次可追踪的 Prompt 内容。
type PromptDefinition struct {
	Name    PromptName
	Version string
	Content string
	Hash    string
}

// PromptCatalog 是不可变的 Prompt 目录。创建后只暴露值副本，不允许运行期修改。
type PromptCatalog struct {
	version     string
	hash        string
	definitions map[PromptName]PromptDefinition
}

// NewPromptCatalog 创建 Prompt 目录，并为每个模板及整个目录计算稳定 SHA-256。
func NewPromptCatalog(version string, contents map[PromptName]string) (PromptCatalog, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return PromptCatalog{}, fmt.Errorf("prompt catalog version is required")
	}
	if len(contents) == 0 {
		return PromptCatalog{}, fmt.Errorf("prompt catalog definitions are required")
	}
	definitions := make(map[PromptName]PromptDefinition, len(contents))
	names := make([]string, 0, len(contents))
	for name, content := range contents {
		if strings.TrimSpace(string(name)) == "" {
			return PromptCatalog{}, fmt.Errorf("prompt name is required")
		}
		if content == "" {
			return PromptCatalog{}, fmt.Errorf("prompt %q content is required", name)
		}
		digest := sha256.Sum256([]byte(content))
		definitions[name] = PromptDefinition{
			Name:    name,
			Version: version,
			Content: content,
			Hash:    hex.EncodeToString(digest[:]),
		}
		names = append(names, string(name))
	}
	sort.Strings(names)
	catalogDigest := sha256.New()
	_, _ = catalogDigest.Write([]byte(version))
	for _, rawName := range names {
		definition := definitions[PromptName(rawName)]
		_, _ = catalogDigest.Write([]byte{0})
		_, _ = catalogDigest.Write([]byte(rawName))
		_, _ = catalogDigest.Write([]byte{0})
		_, _ = catalogDigest.Write([]byte(definition.Hash))
	}
	return PromptCatalog{
		version:     version,
		hash:        hex.EncodeToString(catalogDigest.Sum(nil)),
		definitions: definitions,
	}, nil
}

func mustNewPromptCatalog(version string, contents map[PromptName]string) PromptCatalog {
	catalog, err := NewPromptCatalog(version, contents)
	if err != nil {
		panic(err)
	}
	return catalog
}

// Version 返回目录版本。
func (c PromptCatalog) Version() string { return c.version }

// Hash 返回由版本、稳定名称和各模板 hash 共同生成的目录 hash。
func (c PromptCatalog) Hash() string { return c.hash }

// Get 返回指定 Prompt 的值副本。
func (c PromptCatalog) Get(name PromptName) (PromptDefinition, bool) {
	definition, ok := c.definitions[name]
	return definition, ok
}

// MustText 返回指定 Prompt 内容。固定目录缺少编译期声明的名称属于编程错误。
func (c PromptCatalog) MustText(name PromptName) string {
	definition, ok := c.Get(name)
	if !ok {
		panic(fmt.Sprintf("prompt %q is not registered", name))
	}
	return definition.Content
}

// Names 按名称排序返回目录项，保证日志、测试和未来快照稳定。
func (c PromptCatalog) Names() []PromptName {
	names := make([]PromptName, 0, len(c.definitions))
	for name := range c.definitions {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

var defaultPromptCatalog = mustNewPromptCatalog("v1", map[PromptName]string{
	PromptSystem:        systemPromptContent,
	PromptPlannerSystem: plannerSystemPromptContent,
	PromptCreatePlan:    createPlanPromptContent,
	PromptUpdatePlan:    updatePlanPromptContent,
	PromptReActSystem:   reactSystemPromptContent,
	PromptExecution:     executionPromptContent,
	PromptSummarize:     summarizePromptContent,
})

// DefaultPromptCatalog 返回当前应用使用的不可变 Prompt 目录。
func DefaultPromptCatalog() PromptCatalog { return defaultPromptCatalog }
