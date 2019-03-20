package client

// List of all Index stats as metric names in SignalFx

const (
	// Docs stats
	indicesDocsDeleted = "elasticsearch.indices.docs.deleted"
	indicesDocsCount   = "elasticsearch.indices.docs.count"

	// Store stats
	indicesStoreSize         = "elasticsearch.indices.store.size"
	indicesStoreThrottleTime = "elasticsearch.indices.store.throttle-time" // Deprecated since ES 6.0

	// Translog stats
	indicesTranslogEarliestLastModifiedAge = "elasticsearch.indices.translog.earliest_last_modified_age"
	indicesTranslogUncommittedOperations   = "elasticsearch.indices.translog.uncommitted_operations"
	indicesTranslogUncommittedSizeInBytes  = "elasticsearch.indices.translog.uncommitted_size_in_bytes"
	indicesTranslogSize                    = "elasticsearch.indices.translog.size"
	indicesTranslogOperations              = "elasticsearch.indices.translog.operations"

	// Segments stats
	indicesSegmentsCount                = "elasticsearch.indices.segments.count"
	indicesSegmentsMemory               = "elasticsearch.indices.segments.memory-size"
	indicesSegmentsIndexWriterMemory    = "elasticsearch.indices.segments.index-writer-memory-size"
	indicesSegmentsMaxIndexWriterMemory = "elasticsearch.indices.segments.index-writer-max-memory-size"
	indicesSegmentsVersionMapMemory     = "elasticsearch.indices.segments.version-map-memory-size"
	indicesSegmentsTermsMemory          = "elasticsearch.indices.segments.terms-memory-size"
	indicesSegmentsTermVectorsMemory    = "elasticsearch.indices.segments.term-vectors-memory-size"
	indicesSegmentsStoredFieldMemory    = "elasticsearch.indices.segments.stored-field-memory-size"
	indicesSegmentsNormsMemory          = "elasticsearch.indices.segments.norms-memory-size"
	indicesSegmentsPointsMemory         = "elasticsearch.indices.segments.points-memory-size"
	indicesSegmentsDocValuesMemory      = "elasticsearch.indices.segments.doc-values-memory-size"
	indicesSegmentsFixedBitSetMemory    = "elasticsearch.indices.segments.fixed-bit-set-memory-size"

	// Flush stats
	indicesFlushTotal     = "elasticsearch.indices.flush.total"
	indicesFlushPeriodic  = "elasticsearch.indices.flush.periodic"
	indicesFlushTotalTime = "elasticsearch.indices.flush.total-time"

	// Warmer stats
	indicesWarmerCurrent   = "elasticsearch.indices.warmer.current"
	indicesWarmerTotal     = "elasticsearch.indices.warmer.total"
	indicesWarmerTotalTime = "elasticsearch.indices.warmer.total-time"

	// Field Data stats
	indicesFielddataMemorySizeInBytes = "elasticsearch.indices.fielddata.memory-size"
	indicesFielddataEvictions         = "elasticsearch.indices.fielddata.evictions"

	// Refresh stats
	indicesRefreshTotal     = "elasticsearch.indices.refresh.total"
	indicesRefreshTotalTime = "elasticsearch.indices.refresh.total-time"
	indicesRefreshListeners = "elasticsearch.indices.refresh.listeners"

	// Merges stats
	indicesMergesCurrent                  = "elasticsearch.indices.merges.current"
	indicesMergesCurrentDocs              = "elasticsearch.indices.merges.current-docs"
	indicesMergesCurrentSizeInBytes       = "elasticsearch.indices.merges.current-size"
	indicesMergesTotal                    = "elasticsearch.indices.merges.total"
	indicesMergesTotalTime                = "elasticsearch.indices.merges.total-time"
	indicesMergesTotalDocs                = "elasticsearch.indices.merges.total-docs"
	indicesMergesTotalSizeInBytes         = "elasticsearch.indices.merges.total-size"
	indicesMergesTotalStoppedTime         = "elasticsearch.indices.merges.stopped-time"
	indicesMergesTotalThrottledTime       = "elasticsearch.indices.merges.throttle-time"
	indicesMergesTotalAutoThrottleInBytes = "elasticsearch.indices.merges.auto-throttle-size"

	// Indexing stats
	indicesIndexingIndexTotal      = "elasticsearch.indices.indexing.index-total"
	indicesIndexingIndexTime       = "elasticsearch.indices.indexing.index-time"
	indicesIndexingIndexCurrent    = "elasticsearch.indices.indexing.index-current"
	indicesIndexingIndexFailed     = "elasticsearch.indices.indexing.index-failed"
	indicesIndexingDeleteTotal     = "elasticsearch.indices.indexing.delete-total"
	indicesIndexingDeleteTime      = "elasticsearch.indices.indexing.delete-time"
	indicesIndexingDeleteCurrent   = "elasticsearch.indices.indexing.delete-current"
	indicesIndexingNoopUpdateTotal = "elasticsearch.indices.indexing.noop-update-total"
	indicesIndexingThrottledTime   = "elasticsearch.indices.indexing.throttle-time"

	// Get stats
	indicesGetTotal        = "elasticsearch.indices.get.total"
	indicesGetTime         = "elasticsearch.indices.get.time"
	indicesGetExistsTotal  = "elasticsearch.indices.get.exists-total"
	indicesGetExistsTime   = "elasticsearch.indices.get.exists-time"
	indicesGetMissingTotal = "elasticsearch.indices.get.missing-total"
	indicesGetMissingTime  = "elasticsearch.indices.get.missing-time"
	indicesGetCurrent      = "elasticsearch.indices.get.current"

	// Search stats
	indicesSearchOpenContexts   = "elasticsearch.indices.search.open-contexts"
	indicesSearchQueryTotal     = "elasticsearch.indices.search.query-total"
	indicesSearchQueryTime      = "elasticsearch.indices.search.query-time"
	indicesSearchQueryCurrent   = "elasticsearch.indices.search.query-current"
	indicesSearchFetchTotal     = "elasticsearch.indices.search.fetch-total"
	indicesSearchFetchTime      = "elasticsearch.indices.search.fetch-time"
	indicesSearchFetchCurrent   = "elasticsearch.indices.search.fetch-current"
	indicesSearchScrollTotal    = "elasticsearch.indices.search.scroll-total"
	indicesSearchScrollTime     = "elasticsearch.indices.search.scroll-time"
	indicesSearchScrollCurrent  = "elasticsearch.indices.search.scroll-current"
	indicesSearchSuggestTotal   = "elasticsearch.indices.search.suggest-total"
	indicesSearchSuggestTime    = "elasticsearch.indices.search.suggest-time"
	indicesSearchSuggestCurrent = "elasticsearch.indices.search.suggest-current"

	// Query Cache stats (known as Filter Cache stats before ES 2.0)
	indicesQuerycacheMemorySizeInBytes = "elasticsearch.indices.query-cache.memory-size"
	indicesQuerycacheTotalCount        = "elasticsearch.indices.query-cache.total-count"
	indicesQuerycacheHitCount          = "elasticsearch.indices.query-cache.hit-count"
	indicesQuerycacheMissCount         = "elasticsearch.indices.query-cache.miss-count"
	indicesQuerycacheCacheCount        = "elasticsearch.indices.query-cache.cache-count"
	indicesQuerycacheCacheSize         = "elasticsearch.indices.query-cache.cache-size"
	indicesQuerycacheEvictions         = "elasticsearch.indices.query-cache.evictions"

	// Filter Cache stats (known as Query Cache stats from ES 2.0)
	indicesFiltercacheMemorySizeInBytes = "elasticsearch.indices.filter-cache.memory-size"
	indicesFiltercacheEvictions         = "elasticsearch.indices.filter-cache.evictions"

	// ID Cache stats (Deprecated since ES 2.0)
	indicesIdcacheMemorySizeInBytes = "elasticsearch.indices.id-cache.memory-size"

	// Suggest stats (Deprecated since ES 5.0)
	indicesSuggestTotal   = "elasticsearch.indices.suggest.total"
	indicesSuggestTime    = "elasticsearch.indices.suggest.time"
	indicesSuggestCurrent = "elasticsearch.indices.suggest.current"

	// Completion stats
	indicesCompletionSizeInBytes = "elasticsearch.indices.completion.size"

	// Request Cache stats
	indicesRequestcacheMemorySizeInBytes = "elasticsearch.indices.request-cache.memory-size"
	indicesRequestcacheEvictions         = "elasticsearch.indices.request-cache.evictions"
	indicesRequestcacheHitCount          = "elasticsearch.indices.request-cache.hit-count"
	indicesRequestcacheMissCount         = "elasticsearch.indices.request-cache.miss-count"

	// Recovery stats
	indicesRecoveryCurrentAsSource = "elasticsearch.indices.recovery.current-as-source"
	indicesRecoveryCurrentAsTarget = "elasticsearch.indices.recovery.current-as-target"
	indicesRecoveryThrottleTime    = "elasticsearch.indices.recovery.throttle-time"

	// Percolate stats
	indicesPercolateCurrent = "elasticsearch.indices.percolate.current"
	indicesPercolateTotal   = "elasticsearch.indices.percolate.total"
	indicesPercolateQueries = "elasticsearch.indices.percolate.queries"
	indicesPercolateTime    = "elasticsearch.indices.percolate.time"
)
