import type { FileInfo } from '@/lib/api/types'

/**
 * 合并本地上传文件与用户显式勾选的历史文件，并按 ID 去重。
 * 未勾选历史文件时返回的附件列表只包含本次上传文件。
 */
export function mergeAttachmentFiles(uploaded: FileInfo[], selected: FileInfo[]): FileInfo[] {
  const merged = new Map<string, FileInfo>()
  for (const file of [...uploaded, ...selected]) {
    if (file.id) merged.set(file.id, file)
  }
  return [...merged.values()]
}

/** 返回 Run 请求使用的附件 ID；空数组表示本轮没有显式文件引用。 */
export function attachmentIdsForSend(uploaded: FileInfo[], selected: FileInfo[]): string[] {
  return mergeAttachmentFiles(uploaded, selected).map((file) => file.id)
}
