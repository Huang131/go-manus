# Debug Session: chat-message-not-displaying

## 症状
- **实际行为**：网页上发送对话内容后，显示"正在思考中..."，但不展示用户发送的内容
- **期望行为**：发送的消息应该立即显示在聊天列表中，然后 AI 响应时显示"正在思考中..."

## 环境
- Docker 本地部署
- go-manus 项目

## 假设（Hypotheses）

1. **前端消息状态更新问题**：发送消息后，前端没有立即更新聊天列表状态
2. **后端 API 响应问题**：后端没有正确保存或返回用户消息
3. **前端消息展示组件问题**：消息数据已获取，但展示组件有问题
4. **WebSocket/SSE 实时通信问题**：消息没有通过实时通道正确推送
5. **数据模型/ID 问题**：消息 ID 生成或关联有问题

## 调试进度

### Step 1: 收集证据 [IN_PROGRESS]
- [ ] 分析前端聊天组件代码
- [ ] 分析后端消息保存 API
- [ ] 检查前后端数据流

### Step 2: 插桩验证 [PENDING]
- [ ] 待定

### Step 3: 修复 [COMPLETED]
- [x] 在 Chat handler 中添加用户消息事件的 SSE 推送
- [x] 修改文件：api/internal/handler/session_handler.go
- [x] 在推送 task_id 事件之前，先推送用户消息事件

**修复内容**：
```go
// 先推送用户消息事件（让前端能立即展示用户发送的内容）
userMsgData, _ := json.Marshal(map[string]interface{}{
	"type": "message",
	"data": map[string]interface{}{
		"role":        msg.Role,
		"message":     msg.Message,
		"attachments": msg.Attachments,
	},
})
c.SSEvent("message", string(userMsgData))
c.Writer.Flush()
```

### Step 4: 验证 [PENDING]
- [ ] 重新构建 Docker 镜像
- [ ] 重启服务
- [ ] 测试发送消息是否立即显示

## 操作日志

### 2026-08-25 分析完成

**问题根因定位**：

在 `api/internal/handler/session_handler.go` 的 `Chat` handler 中：

1. **第120-135行**：用户消息被正确保存到数据库
```go
userEvent := &model.Event{
	Type: model.EventTypeMessage,
	Data: userEventData,
}
if err := s.sessionRep.AppendEvent(ctx, sessionID, userEvent); err != nil {
	logger.Warn("添加用户消息事件失败", zap.String("session_id", sessionID), zap.Error(err))
}
```

2. **问题**：用户消息事件**没有被推送到 SSE 流中**！

- 第147-176行的 SSE 推送逻辑只推送：
  - `task_id` 事件
  - 然后轮询 `h.agent.GetTaskEvents` 获取 output_stream 中的事件

- `GetTaskEvents` 获取的是 Agent 任务产生的输出事件（工具调用、步骤等），**不包括用户消息事件**

3. **前端期望**：前端的 `eventsToTimeline` 函数（ui/src/lib/session-events.ts）监听 `type: "message"` 且 `data.role: "user"` 的 SSE 事件来展示用户消息

4. **结论**：用户消息被保存到数据库，但没有被通过 SSE 流推送给前端，导致前端无法立即展示用户消息

**修复方案**：在 Chat handler 中，推送 `task_id` 事件后，立即推送用户消息事件到 SSE 流
