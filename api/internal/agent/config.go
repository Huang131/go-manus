package agent

import (
	"github.com/Huang131/go-manus/api/internal/agent/tools"
	"github.com/Huang131/go-manus/api/internal/model"
)

// A2AConfig / A2AAgent 类型别名：handler / bootstrap 仍通过 agent.A2AConfig 访问，
// 实际定义已下沉到 tools 包，避免 agent → tools 反向依赖形成循环。
type A2AConfig = tools.A2AConfig
type A2AAgent = tools.A2AAgent

// RuntimeMCPConfig converts the persisted control-plane model into an isolated
// runtime snapshot. Disabled servers never reach the client manager.
// 直接使用 model.MCPConfig 的 ToRuntime 方法进行转换。
func RuntimeMCPConfig(cfg *model.MCPConfig) *model.MCPConfig {
	if cfg == nil {
		return nil
	}
	// 深拷贝以避免后续修改影响原始配置
	runtime := cfg.ToRuntime()
	if runtime == nil || len(runtime.Servers) == 0 {
		return runtime
	}
	// 深拷贝 Args 和 Env
	for i := range runtime.Servers {
		if len(runtime.Servers[i].Args) > 0 {
			runtime.Servers[i].Args = append([]string(nil), runtime.Servers[i].Args...)
		}
		if runtime.Servers[i].Env != nil {
			runtime.Servers[i].Env = cloneStringMap(runtime.Servers[i].Env)
		}
	}
	return runtime
}

// RuntimeA2AConfig converts persisted A2A settings into an isolated runtime snapshot.
func RuntimeA2AConfig(cfg *model.A2AConfig) *A2AConfig {
	if cfg == nil {
		return nil
	}
	runtimeCfg := &A2AConfig{Agents: make([]A2AAgent, 0, len(cfg.Servers))}
	for _, server := range cfg.Servers {
		if !server.Enabled {
			continue
		}
		runtimeCfg.Agents = append(runtimeCfg.Agents, A2AAgent{
			Name: server.ID,
			URL:  server.URL,
		})
	}
	return runtimeCfg
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
