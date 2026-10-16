import assert from 'node:assert/strict'
import test from 'node:test'
import { attachmentIdsForSend, mergeAttachmentFiles } from './session-files'
import type { FileInfo } from './api/types'

const file = (id: string, filename = `${id}.md`): FileInfo => ({
  id,
  filename,
  filepath: `files/session/${id}.md`,
  key: `files/session/${id}.md`,
  extension: '.md',
  content_type: 'text/markdown',
  size: 10,
})

test('未勾选历史文件时不生成附件 ID', () => {
  assert.deepEqual(attachmentIdsForSend([], []), [])
})

test('只发送用户显式勾选的历史文件和本地上传文件', () => {
  assert.deepEqual(attachmentIdsForSend([file('upload')], [file('history')]), ['upload', 'history'])
})

test('合并附件按文件 ID 去重并保留最新选择', () => {
  const selected = file('same', 'history.md')
  assert.deepEqual(mergeAttachmentFiles([file('same', 'upload.md')], [selected]), [selected])
})
