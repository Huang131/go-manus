package llmcore

import "testing"

// TestIsDeterministicKind 锁定每个 Kind 的确定性归类（单一事实源的守护测试）。
// 该归类同时决定两个决策，任何一侧的语义变化都会在此暴露：
//   - ProviderError.Fallbackable：确定性错误不 fallback
//   - llm 层健康统计：确定性错误不污染模型健康
func TestIsDeterministicKind(t *testing.T) {
	cases := []struct {
		kind ErrorKind
		want bool
		why  string
	}{
		{KindAuth, true, "key 配错：确定性配置错误"},
		{KindNotFound, true, "模型名写错：确定性配置错误"},
		{KindBadRequest, true, "协议不兼容：确定性配置/请求错误"},
		{KindContentFilter, true, "内容被拒：模型没病，是请求触发"},
		{KindContextLimit, true, "请求太大：模型没病，是请求问题"},
		{KindRateLimit, false, "限流：供应商瞬态抖动"},
		{KindServer, false, "5xx：供应商健康问题"},
		{KindNetwork, false, "网络层抖动"},
		{KindTimeout, false, "超时：瞬态"},
		{KindUnknown, false, "未知：默认按瞬态处理"},
		{ErrorKind("future_kind"), false, "新增未归类 Kind 默认瞬态（清单漏改时新 Kind 仍可 fallback/计入健康）"},
	}
	for _, tc := range cases {
		if got := IsDeterministicKind(tc.kind); got != tc.want {
			t.Errorf("IsDeterministicKind(%s) = %v, want %v（%s）", tc.kind, got, tc.want, tc.why)
		}
	}
}

// TestProviderErrorFallbackableConsistency 锁定 Fallbackable 与 IsDeterministicKind
// 的反相关关系：凡是确定性 Kind，构造出的 ProviderError 必须 Fallbackable=false。
// 防止 isFallbackable 未来被改成独立实现导致两份清单重新分叉。
func TestProviderErrorFallbackableConsistency(t *testing.T) {
	allKinds := []ErrorKind{
		KindAuth, KindNotFound, KindRateLimit, KindServer, KindBadRequest,
		KindContentFilter, KindNetwork, KindTimeout, KindContextLimit, KindUnknown,
	}
	for _, kind := range allKinds {
		pe := NewProviderError(kind, "test", "m", "msg")
		if wantFallbackable := !IsDeterministicKind(kind); pe.Fallbackable != wantFallbackable {
			t.Errorf("kind=%s: Fallbackable=%v, want %v（与 IsDeterministicKind 分叉）",
				kind, pe.Fallbackable, wantFallbackable)
		}
	}
}
