package sandbox

// maxDownloadBytes 下载文件大小上限（64MB），避免一次性读爆 API 进程内存。
const maxDownloadBytes = 64 << 20
