package imagecolor

const (
	_ int64 = 1 << (10 * iota)
	KB
	MB
	GB
)

const (
	maxParallelDownloads   = 10
	maxDataQueueItems      = 100
	maxCacheDataQueueItems = 100000
)

const (
	enableHistory  = true
	maxHistorySize = 100000
)

const (
	placeholderFileSize = uint64(10 * MB)
	maxParallelAnalysis = 5
)
