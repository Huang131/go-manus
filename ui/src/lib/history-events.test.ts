import assert from 'node:assert/strict'
import test from 'node:test'

import { eventsToTimeline } from './session-events'
import { historyToEvents } from './history-events'
import type { RunHistoryPage } from './api/types'

test('按 Run 的持久化追踪恢复计划、步骤和已完成工具调用', () => {
  const history: RunHistoryPage = {
    items: [{
      run: {
        id: 'run-1',
        session_id: 'session-1',
        status: 'succeeded',
        created_at: '2026-10-16T00:00:00Z',
        updated_at: '2026-10-16T00:01:00Z',
      },
      messages: [
        { id: 'user-1', session_id: 'session-1', run_id: 'run-1', role: 'user', content: '搜索 Go 文档', created_at: '2026-10-16T00:00:00Z' },
        { id: 'assistant-1', session_id: 'session-1', run_id: 'run-1', role: 'assistant', content: '已完成搜索', created_at: '2026-10-16T00:01:00Z' },
      ],
      events: [
        { run_id: 'run-1', sequence: 1, event_type: 'plan', data: { plan: { id: 'plan-1', steps: [{ id: 'step-1', description: '检索文档', status: 'pending' }] }, status: 'created' }, created_at: '2026-10-16T00:00:01Z' },
        { run_id: 'run-1', sequence: 2, event_type: 'step', data: { step: { id: 'step-1', description: '检索文档', status: 'running' } }, created_at: '2026-10-16T00:00:02Z' },
        { run_id: 'run-1', sequence: 3, event_type: 'tool_calling', data: { tool_call_id: 'call-1', function_name: 'search', arguments: { query: 'Go' } }, created_at: '2026-10-16T00:00:03Z' },
        { run_id: 'run-1', sequence: 4, event_type: 'tool_called', data: { tool_call_id: 'call-1', function_name: 'search', arguments: { query: 'Go' }, result: { success: true, message: 'ok', data: 'result' } }, created_at: '2026-10-16T00:00:04Z' },
        { run_id: 'run-1', sequence: 5, event_type: 'step', data: { step: { id: 'step-1', description: '检索文档', status: 'pending' }, status: 'completed' }, created_at: '2026-10-16T00:00:05Z' },
      ],
    }],
    total: 1,
    limit: 50,
    offset: 0,
  }

  const events = historyToEvents(history, [])

  assert.deepEqual(events.map((event) => event.type), [
    'message', 'plan', 'step', 'tool_calling', 'tool_called', 'step', 'message',
  ])
  const step = eventsToTimeline(events).find((item) => item.kind === 'step')
  assert.equal(step?.kind, 'step')
  assert.deepEqual(step?.tools.map((tool) => ({ function: tool.function, status: tool.status })), [
    { function: 'search', status: 'called' },
  ])
  assert.equal((events[5].data as { step: { status: string } }).step.status, 'completed')
})

test('无追踪事件的新 Run 从执行快照恢复计划和步骤但不伪造工具调用', () => {
  const history: RunHistoryPage = {
    items: [{
      run: {
        id: 'run-2',
        session_id: 'session-1',
        status: 'waiting_input',
        created_at: '2026-10-16T00:00:00Z',
        updated_at: '2026-10-16T00:01:00Z',
        execution_snapshot: {
          plan_id: 'plan-2',
          plan_title: '补充信息',
          plan_goal: '完成任务',
          plan_language: 'zh-CN',
          plan_message: '需要更多信息',
          steps: [{ id: 'step-2', description: '等待用户确认', status: 'running' }],
        },
      },
      messages: [{ id: 'user-2', session_id: 'session-1', run_id: 'run-2', role: 'user', content: '继续', created_at: '2026-10-16T00:00:00Z' }],
      events: [],
    }],
    total: 1,
    limit: 50,
    offset: 0,
  }

  const events = historyToEvents(history, [])

  assert.deepEqual(events.map((event) => event.type), ['message', 'plan', 'step'])
  assert.equal((events[1].data as { plan: { id: string } }).plan.id, 'plan-2')
  assert.equal((events[2].data as { step: { status: string } }).step.status, 'running')
})

test('同一 Run 的多轮 waiting 消息与追踪事件保持持久化顺序', () => {
  const history: RunHistoryPage = {
    items: [{
      run: {
        id: 'run-3',
        session_id: 'session-1',
        status: 'succeeded',
        created_at: '2026-10-16T00:00:00Z',
        updated_at: '2026-10-16T00:00:06Z',
      },
      messages: [
        { id: 'user-3', session_id: 'session-1', run_id: 'run-3', role: 'user', content: '开始', created_at: '2026-10-16T00:00:00Z' },
        { id: 'question-3', session_id: 'session-1', run_id: 'run-3', role: 'assistant', content: '请确认', created_at: '2026-10-16T00:00:02Z' },
        { id: 'answer-3', session_id: 'session-1', run_id: 'run-3', role: 'user', content: '确认', created_at: '2026-10-16T00:00:03Z' },
        { id: 'final-3', session_id: 'session-1', run_id: 'run-3', role: 'assistant', content: '完成', created_at: '2026-10-16T00:00:06Z' },
      ],
      events: [
        { run_id: 'run-3', sequence: 1, event_type: 'wait', data: { message: '请确认' }, created_at: '2026-10-16T00:00:02.100Z' },
        { run_id: 'run-3', sequence: 2, event_type: 'tool_called', data: { tool_call_id: 'call-3', function_name: 'shell', arguments: {}, result: { success: true, message: 'ok' } }, created_at: '2026-10-16T00:00:05Z' },
      ],
    }],
    total: 1,
    limit: 50,
    offset: 0,
  }

  const events = historyToEvents(history, [])

  assert.deepEqual(events.map((event) => event.type), [
    'message', 'message', 'wait', 'message', 'tool_called', 'message',
  ])
  assert.equal((events[1].data as { message: string }).message, '请确认')
  assert.equal((events[3].data as { message: string }).message, '确认')
})

test('等待事件落库早于问题消息时仍先展示助手问题', () => {
  const history: RunHistoryPage = {
    items: [{
      run: {
        id: 'run-4',
        session_id: 'session-1',
        status: 'waiting_input',
        created_at: '2026-10-16T00:00:00Z',
        updated_at: '2026-10-16T00:00:02Z',
      },
      messages: [
        { id: 'user-4', session_id: 'session-1', run_id: 'run-4', role: 'user', content: '开始', created_at: '2026-10-16T00:00:00.000000Z' },
        { id: 'question-4', session_id: 'session-1', run_id: 'run-4', role: 'assistant', content: '请确认', created_at: '2026-10-16T00:00:00.020000Z' },
      ],
      events: [
        { run_id: 'run-4', sequence: 1, event_type: 'wait', data: { message: '请确认' }, created_at: '2026-10-16T00:00:00.010000Z' },
      ],
    }],
    total: 1,
    limit: 50,
    offset: 0,
  }

  const events = historyToEvents(history, [])

  assert.deepEqual(events.map((event) => event.type), ['message', 'message', 'wait'])
  assert.equal((events[1].data as { message: string }).message, '请确认')
})

test('已结束 Run 清空等待消息 ID 后仍按问题消息回放 wait', () => {
  const history: RunHistoryPage = {
    items: [{
      run: {
        id: 'run-5',
        session_id: 'session-1',
        status: 'succeeded',
        created_at: '2026-10-16T00:00:00Z',
        updated_at: '2026-10-16T00:00:03Z',
      },
      messages: [
        { id: 'user-5', session_id: 'session-1', run_id: 'run-5', role: 'user', content: '开始', created_at: '2026-10-16T00:00:00.000000Z' },
        { id: 'question-5', session_id: 'session-1', run_id: 'run-5', role: 'assistant', content: '请确认', created_at: '2026-10-16T00:00:00.020000Z' },
        { id: 'answer-5', session_id: 'session-1', run_id: 'run-5', role: 'user', content: '确认', reply_to_message_id: 'question-5', created_at: '2026-10-16T00:00:00.030000Z' },
      ],
      events: [
        { run_id: 'run-5', sequence: 1, event_type: 'wait', data: {}, created_at: '2026-10-16T00:00:00.010000Z' },
      ],
    }],
    total: 1,
    limit: 50,
    offset: 0,
  }

  const events = historyToEvents(history, [])

  assert.deepEqual(events.map((event) => event.type), ['message', 'message', 'wait', 'message'])
  assert.equal((events[1].data as { message: string }).message, '请确认')
})
