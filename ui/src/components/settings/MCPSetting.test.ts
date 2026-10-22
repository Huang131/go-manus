import test from 'node:test'
import assert from 'node:assert/strict'
import {normalizeMCPConfig} from './MCPSetting'

test('normalizes a single MCP JSON object and preserves advanced fields', () => {
  const result = normalizeMCPConfig({servers: [{name: 'local', command: 'npx', args: ['server'], env: {TOKEN: 'secret'}, timeout: 60, max_retries: 4, enabled: false}]})
  assert.deepEqual(result.servers[0], {name: 'local', command: 'npx', args: ['server'], env: {TOKEN: 'secret'}, timeout: 60, max_retries: 4, enabled: false})
})

test('rejects malformed MCP config and multiple entries for single item editor', () => {
  assert.throws(() => normalizeMCPConfig({servers: [{name: 'local'}]}), /name 和 command/)
  assert.throws(() => normalizeMCPConfig({servers: [{name: 'a', command: 'a'}, {name: 'b', command: 'b'}]}), /仅提供一个/)
})
