import assert from 'node:assert/strict'
import test from 'node:test'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

import { ChatMessage } from './chat-message'
import type { TimelineItem } from '@/lib/session-events'

test('步骤详情展示 Markdown 执行结果，即使没有工具调用', () => {
  const item: TimelineItem = {
    kind: 'step', id: 'step-1', tools: [],
    data: { id: 'step-1', description: '分析文档', status: 'completed', result: '## 文档结构\n\n完整分析结果' },
  }
  const html = renderToStaticMarkup(createElement(ChatMessage, { item }))
  assert.match(html, /<h2[^>]*>文档结构<\/h2>/)
  assert.match(html, /完整分析结果/)
})

test('失败步骤展示具体错误且不重复相同结果', () => {
  const item: TimelineItem = {
    kind: 'step', id: 'step-2', tools: [],
    data: { id: 'step-2', description: '读取文件', status: 'failed', result: '附件无权限', error: '附件无权限' },
  }
  const html = renderToStaticMarkup(createElement(ChatMessage, { item }))
  assert.equal(html.match(/附件无权限/g)?.length, 1)
  assert.match(html, /失败原因/)
})

test('失败步骤保留部分结果和独立错误', () => {
  const item: TimelineItem = {
    kind: 'step', id: 'step-3', tools: [],
    data: { id: 'step-3', description: '读取文件', status: 'failed', result: '已读取前十行', error: '后续分页失败' },
  }
  const html = renderToStaticMarkup(createElement(ChatMessage, { item }))
  assert.match(html, /已读取前十行/)
  assert.match(html, /后续分页失败/)
})
