package client

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// Groups of Node stats being that the monitor collects
const (
	TransportStatsGroup  = "transport"
	HTTPStatsGroup       = "http"
	JVMStatsGroup        = "jvm"
	ThreadpoolStatsGroup = "thread_pool"
	ProcessStatsGroup    = "process"
)

// GetNodeStatsDatapoints fetches datapoints for ES Node stats
func GetNodeStatsDatapoints(nodeStatsOutput *NodeStatsOutput, defaultDims map[string]string, selectedThreadPools map[string]bool, enhancedStatsForIndexGroups map[string]bool, nodeStatsGroupEnhancedOption map[string]bool) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint
	for _, nodeStats := range nodeStatsOutput.NodeStats {
		out = append(out, getNodeStatsDatapointsHelper(nodeStats, defaultDims, selectedThreadPools, enhancedStatsForIndexGroups, nodeStatsGroupEnhancedOption)...)
	}
	return out
}

func getNodeStatsDatapointsHelper(nodeStats NodeStats, defaultDims map[string]string, selectedThreadPools map[string]bool, enhancedStatsForIndexGroups map[string]bool, nodeStatsGroupEnhancedOption map[string]bool) []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint

	dps = append(dps, nodeStats.JVM.getJVMStats(nodeStatsGroupEnhancedOption[JVMStatsGroup], defaultDims)...)
	dps = append(dps, nodeStats.Process.getProcessStats(nodeStatsGroupEnhancedOption[ProcessStatsGroup], defaultDims)...)
	dps = append(dps, nodeStats.Transport.getTransportStats(nodeStatsGroupEnhancedOption[TransportStatsGroup], defaultDims)...)
	dps = append(dps, nodeStats.HTTP.getHTTPStats(nodeStatsGroupEnhancedOption[HTTPStatsGroup], defaultDims)...)
	dps = append(dps, fetchThreadPoolStats(nodeStatsGroupEnhancedOption[ThreadpoolStatsGroup], nodeStats.ThreadPool, defaultDims, selectedThreadPools)...)
	dps = append(dps, nodeStats.Indices.getIndexGroupStats(enhancedStatsForIndexGroups, defaultDims)...)

	return dps
}

func (jvm *JVM) getJVMStats(enhanced bool, dims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper("elasticsearch.jvm.threads.count", dims, jvm.JvmThreadsStats.Count),
			prepareGaugeHelper("elasticsearch.jvm.threads.peak", dims, jvm.JvmThreadsStats.PeakCount),
			prepareGaugeHelper("elasticsearch.jvm.mem.heap-used-percent", dims, jvm.JvmMemStats.HeapUsedPercent),
			prepareGaugeHelper("elasticsearch.jvm.mem.heap-max", dims, jvm.JvmMemStats.HeapMaxInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.non-heap-committed", dims, jvm.JvmMemStats.NonHeapCommittedInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.non-heap-used", dims, jvm.JvmMemStats.NonHeapUsedInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.young.max_in_bytes", dims, jvm.JvmMemStats.Pools.Young.MaxInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.young.used_in_bytes", dims, jvm.JvmMemStats.Pools.Young.UsedInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.young.peak_used_in_bytes", dims, jvm.JvmMemStats.Pools.Young.PeakUsedInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.young.peak_max_in_bytes", dims, jvm.JvmMemStats.Pools.Young.PeakMaxInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.old.max_in_bytes", dims, jvm.JvmMemStats.Pools.Old.MaxInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.old.used_in_bytes", dims, jvm.JvmMemStats.Pools.Old.UsedInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.old.peak_used_in_bytes", dims, jvm.JvmMemStats.Pools.Old.PeakUsedInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.old.peak_max_in_bytes", dims, jvm.JvmMemStats.Pools.Old.PeakMaxInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.survivor.max_in_bytes", dims, jvm.JvmMemStats.Pools.Survivor.MaxInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.survivor.used_in_bytes", dims, jvm.JvmMemStats.Pools.Survivor.UsedInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.survivor.peak_used_in_bytes", dims, jvm.JvmMemStats.Pools.Survivor.PeakUsedInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.pools.survivor.peak_max_in_bytes", dims, jvm.JvmMemStats.Pools.Survivor.PeakMaxInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.buffer_pools.mapped.count", dims, jvm.BufferPools.Mapped.Count),
			prepareGaugeHelper("elasticsearch.jvm.mem.buffer_pools.mapped.used_in_bytes", dims, jvm.BufferPools.Mapped.UsedInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.buffer_pools.mapped.total_capacity_in_bytes", dims, jvm.BufferPools.Mapped.TotalCapacityInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.buffer_pools.direct.count", dims, jvm.BufferPools.Direct.Count),
			prepareGaugeHelper("elasticsearch.jvm.mem.buffer_pools.direct.used_in_bytes", dims, jvm.BufferPools.Direct.UsedInBytes),
			prepareGaugeHelper("elasticsearch.jvm.mem.buffer_pools.direct.total_capacity_in_bytes", dims, jvm.BufferPools.Direct.TotalCapacityInBytes),
			prepareGaugeHelper("elasticsearch.jvm.classes.current-loaded-count", dims, jvm.Classes.CurrentLoadedCount),

			prepareCumulativeHelper("elasticsearch.jvm.gc.count", dims, jvm.JvmGcStats.Collectors.Young.CollectionCount),
			prepareCumulativeHelper("elasticsearch.jvm.gc.old-count", dims, jvm.JvmGcStats.Collectors.Old.CollectionCount),
			prepareCumulativeHelper("elasticsearch.jvm.gc.old-time", dims, jvm.JvmGcStats.Collectors.Old.CollectionTimeInMillis),
			prepareCumulativeHelper("elasticsearch.jvm.classes.total-loaded-count", dims, jvm.Classes.TotalLoadedCount),
			prepareCumulativeHelper("elasticsearch.jvm.classes.total-unloaded-count", dims, jvm.Classes.TotalUnloadedCount),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper("elasticsearch.jvm.mem.heap-used", dims, jvm.JvmMemStats.HeapUsedInBytes),
		prepareGaugeHelper("elasticsearch.jvm.mem.heap-committed", dims, jvm.JvmMemStats.HeapCommittedInBytes),
		prepareCumulativeHelper("elasticsearch.jvm.uptime", dims, jvm.UptimeInMillis),
		prepareCumulativeHelper("elasticsearch.jvm.gc.time", dims, jvm.JvmGcStats.Collectors.Young.CollectionTimeInMillis),
	}...)

	return out
}

func (processStats *Process) getProcessStats(enhanced bool, dims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper("elasticsearch.process.max_file_descriptors", dims, processStats.MaxFileDescriptors),
			prepareGaugeHelper("elasticsearch.process.cpu.percent", dims, processStats.CPU.Percent),
			prepareCumulativeHelper("elasticsearch.process.cpu.time", dims, processStats.CPU.TotalInMillis),
			prepareCumulativeHelper("elasticsearch.process.mem.total-virtual-size", dims, processStats.Mem.TotalVirtualInBytes),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper("elasticsearch.process.open_file_descriptors", dims, processStats.OpenFileDescriptors),
	}...)

	return out
}

func fetchThreadPoolStats(enhanced bool, threadPools map[string]ThreadPoolStats, defaultDims map[string]string, selectedThreadPools map[string]bool) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint
	for threadPool, stats := range threadPools {
		if !selectedThreadPools[threadPool] {
			continue
		}
		out = append(out, threadPoolDatapoints(enhanced, threadPool, stats, defaultDims)...)
	}
	return out
}

func threadPoolDatapoints(enhanced bool, threadPool string, threadPoolStats ThreadPoolStats, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint
	threadPoolDimension := map[string]string{}
	threadPoolDimension["thread_pool"] = threadPool

	dims := utils.MergeStringMaps(defaultDims, threadPoolDimension)

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper("elasticsearch.thread_pool.threads", dims, threadPoolStats.Threads),
			prepareGaugeHelper("elasticsearch.thread_pool.queue", dims, threadPoolStats.Queue),
			prepareGaugeHelper("elasticsearch.thread_pool.active", dims, threadPoolStats.Active),
			prepareGaugeHelper("elasticsearch.thread_pool.largest", dims, threadPoolStats.Largest),
			prepareCumulativeHelper("elasticsearch.thread_pool.completed", dims, threadPoolStats.Completed),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareCumulativeHelper("elasticsearch.thread_pool.rejected", dims, threadPoolStats.Rejected),
	}...)
	return out
}

func (transport *Transport) getTransportStats(enhanced bool, dims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper("elasticsearch.transport.server_open", dims, transport.ServerOpen),
			prepareCumulativeHelper("elasticsearch.transport.rx.count", dims, transport.RxCount),
			prepareCumulativeHelper("elasticsearch.transport.rx.size", dims, transport.RxSizeInBytes),
			prepareCumulativeHelper("elasticsearch.transport.tx.count", dims, transport.TxCount),
			prepareCumulativeHelper("elasticsearch.transport.tx.size", dims, transport.TxSizeInBytes),
		}...)
	}
	return out
}

func (http *HTTP) getHTTPStats(enhanced bool, dims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper("elasticsearch.http.current_open", dims, http.CurrentOpen),
			prepareCumulativeHelper("elasticsearch.http.total_open", dims, http.TotalOpened),
		}...)
	}

	return out
}

func (indexStatsGroups *IndexStatsGroups) getIndexGroupStats(enhancedStatsForIndexGroups map[string]bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	out = append(out, indexStatsGroups.Docs.getDocsStats(enhancedStatsForIndexGroups[DocsStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Store.getStoreStats(enhancedStatsForIndexGroups[StoreStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Indexing.getIndexingStats(enhancedStatsForIndexGroups[IndexingStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Get.getGetStats(enhancedStatsForIndexGroups[GetStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Search.getSearchStats(enhancedStatsForIndexGroups[SearchStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Merges.getMergesStats(enhancedStatsForIndexGroups[MergesStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Refresh.getRefreshStats(enhancedStatsForIndexGroups[RefreshStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Flush.getFlushStats(enhancedStatsForIndexGroups[FlushStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Warmer.getWarmerStats(enhancedStatsForIndexGroups[WarmerStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.QueryCache.getQueryCacheStats(enhancedStatsForIndexGroups[QueryCacheStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.FilterCache.getFilterCacheStats(enhancedStatsForIndexGroups[FilterCacheStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Fielddata.getFielddataStats(enhancedStatsForIndexGroups[FieldDataStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Completion.getCompletionStats(enhancedStatsForIndexGroups[CompletionStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Segments.getSegmentsStats(enhancedStatsForIndexGroups[SegmentsStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Translog.getTranslogStats(enhancedStatsForIndexGroups[TranslogStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.RequestCache.getRequestCacheStats(enhancedStatsForIndexGroups[RequestCacheStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Recovery.getRecoveryStats(enhancedStatsForIndexGroups[RecoveryStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.IDCache.getIDCacheStats(enhancedStatsForIndexGroups[IDCacheStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Suggest.getSuggestStats(enhancedStatsForIndexGroups[SuggestStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroups.Percolate.getPercolateStats(enhancedStatsForIndexGroups[PercolateStatsGroup], defaultDims)...)

	return out
}

func (docs *Docs) getDocsStats(_ bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(ElasticsearchIndicesDocsCount, defaultDims, docs.Count),
		prepareGaugeHelper(ElasticsearchIndicesDocsDeleted, defaultDims, docs.Deleted),
	}...)

	return out
}

func (store *Store) getStoreStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareCumulativeHelper(ElasticsearchIndicesStoreThrottleTime, defaultDims, store.ThrottleTimeInMillis),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(ElasticsearchIndicesStoreSize, defaultDims, store.SizeInBytes),
	}...)

	return out
}

func (indexing Indexing) getIndexingStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesIndexingIndexCurrent, defaultDims, indexing.IndexCurrent),
			prepareGaugeHelper(ElasticsearchIndicesIndexingIndexFailed, defaultDims, indexing.IndexFailed),
			prepareGaugeHelper(ElasticsearchIndicesIndexingDeleteCurrent, defaultDims, indexing.DeleteCurrent),
			prepareCumulativeHelper(ElasticsearchIndicesIndexingDeleteTotal, defaultDims, indexing.DeleteTotal),
			prepareCumulativeHelper(ElasticsearchIndicesIndexingDeleteTime, defaultDims, indexing.DeleteTimeInMillis),
			prepareCumulativeHelper(ElasticsearchIndicesIndexingNoopUpdateTotal, defaultDims, indexing.NoopUpdateTotal),
			prepareCumulativeHelper(ElasticsearchIndicesIndexingThrottleTime, defaultDims, indexing.ThrottleTimeInMillis),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareCumulativeHelper(ElasticsearchIndicesIndexingIndexTotal, defaultDims, indexing.IndexTotal),
		prepareCumulativeHelper(ElasticsearchIndicesIndexingIndexTime, defaultDims, indexing.IndexTimeInMillis),
	}...)

	return out
}

func (get *Get) getGetStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesGetCurrent, defaultDims, get.Current),
			prepareCumulativeHelper(ElasticsearchIndicesGetTime, defaultDims, get.TimeInMillis),
			prepareCumulativeHelper(ElasticsearchIndicesGetExistsTotal, defaultDims, get.ExistsTotal),
			prepareCumulativeHelper(ElasticsearchIndicesGetExistsTime, defaultDims, get.ExistsTimeInMillis),
			prepareCumulativeHelper(ElasticsearchIndicesGetMissingTotal, defaultDims, get.MissingTotal),
			prepareCumulativeHelper(ElasticsearchIndicesGetMissingTime, defaultDims, get.MissingTimeInMillis),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareCumulativeHelper(ElasticsearchIndicesGetTotal, defaultDims, get.Total),
	}...)

	return out
}

func (search *Search) getSearchStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesSearchQueryCurrent, defaultDims, search.QueryCurrent),
			prepareGaugeHelper(ElasticsearchIndicesSearchFetchCurrent, defaultDims, search.FetchCurrent),
			prepareGaugeHelper(ElasticsearchIndicesSearchScrollCurrent, defaultDims, search.ScrollCurrent),
			prepareGaugeHelper(ElasticsearchIndicesSearchSuggestCurrent, defaultDims, search.SuggestCurrent),
			prepareGaugeHelper(ElasticsearchIndicesSearchOpenContexts, defaultDims, search.SuggestCurrent),
			prepareCumulativeHelper(ElasticsearchIndicesSearchFetchTime, defaultDims, search.FetchTimeInMillis),
			prepareCumulativeHelper(ElasticsearchIndicesSearchFetchTotal, defaultDims, search.FetchTotal),
			prepareCumulativeHelper(ElasticsearchIndicesSearchScrollTime, defaultDims, search.ScrollTimeInMillis),
			prepareCumulativeHelper(ElasticsearchIndicesSearchScrollTotal, defaultDims, search.ScrollTotal),
			prepareCumulativeHelper(ElasticsearchIndicesSearchSuggestTime, defaultDims, search.SuggestTimeInMillis),
			prepareCumulativeHelper(ElasticsearchIndicesSearchSuggestTotal, defaultDims, search.SuggestTotal),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareCumulativeHelper(ElasticsearchIndicesSearchQueryTime, defaultDims, search.QueryTimeInMillis),
		prepareCumulativeHelper(ElasticsearchIndicesSearchQueryTotal, defaultDims, search.QueryTotal),
	}...)

	return out
}

func (merges *Merges) getMergesStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesMergesCurrentDocs, defaultDims, merges.CurrentDocs),
			prepareGaugeHelper(ElasticsearchIndicesMergesCurrentSize, defaultDims, merges.CurrentSizeInBytes),
			prepareCumulativeHelper(ElasticsearchIndicesMergesTotalDocs, defaultDims, merges.TotalDocs),
			prepareCumulativeHelper(ElasticsearchIndicesMergesTotalSize, defaultDims, merges.TotalSizeInBytes),
			prepareCumulativeHelper(ElasticsearchIndicesMergesStoppedTime, defaultDims, merges.TotalStoppedTimeInMillis),
			prepareCumulativeHelper(ElasticsearchIndicesMergesThrottleTime, defaultDims, merges.TotalThrottledTimeInMillis),
			prepareCumulativeHelper(ElasticsearchIndicesMergesAutoThrottleSize, defaultDims, merges.TotalAutoThrottleInBytes),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(ElasticsearchIndicesMergesCurrent, defaultDims, merges.Current),
		prepareCumulativeHelper(ElasticsearchIndicesMergesTotal, defaultDims, merges.Total),
		prepareCumulativeHelper(ElasticsearchIndicesMergesTotalTime, defaultDims, merges.TotalTimeInMillis),
	}...)

	return out
}

func (refresh *Refresh) getRefreshStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesRefreshListeners, defaultDims, refresh.Listeners),
			prepareCumulativeHelper(ElasticsearchIndicesRefreshTotal, defaultDims, refresh.Total),
			prepareCumulativeHelper(ElasticsearchIndicesRefreshTotalTime, defaultDims, refresh.TotalTimeInMillis),
		}...)
	}

	return out
}

func (flush *Flush) getFlushStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesFlushPeriodic, defaultDims, flush.Periodic),
			prepareCumulativeHelper(ElasticsearchIndicesFlushTotal, defaultDims, flush.Total),
			prepareCumulativeHelper(ElasticsearchIndicesFlushTotalTime, defaultDims, flush.TotalTimeInMillis),
		}...)
	}

	return out
}

func (warmer *Warmer) getWarmerStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesWarmerCurrent, defaultDims, warmer.Current),
			prepareCumulativeHelper(ElasticsearchIndicesWarmerTotal, defaultDims, warmer.Total),
			prepareCumulativeHelper(ElasticsearchIndicesWarmerTotalTime, defaultDims, warmer.TotalTimeInMillis),
		}...)
	}

	return out
}

func (queryCache *QueryCache) getQueryCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesQueryCacheCacheSize, defaultDims, queryCache.CacheSize),
			prepareGaugeHelper(ElasticsearchIndicesQueryCacheCacheCount, defaultDims, queryCache.CacheCount),
			prepareCumulativeHelper(ElasticsearchIndicesQueryCacheEvictions, defaultDims, queryCache.Evictions),
			prepareCumulativeHelper(ElasticsearchIndicesQueryCacheHitCount, defaultDims, queryCache.HitCount),
			prepareCumulativeHelper(ElasticsearchIndicesQueryCacheMissCount, defaultDims, queryCache.MissCount),
			prepareCumulativeHelper(ElasticsearchIndicesQueryCacheTotalCount, defaultDims, queryCache.TotalCount),
		}...)
	}
	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(ElasticsearchIndicesQueryCacheMemorySize, defaultDims, queryCache.MemorySizeInBytes),
	}...)

	return out
}

func (filterCache *FilterCache) getFilterCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareCumulativeHelper(ElasticsearchIndicesFilterCacheEvictions, defaultDims, filterCache.Evictions),
		}...)

		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesFilterCacheMemorySize, defaultDims, filterCache.MemorySizeInBytes),
		}...)
	}

	return out
}

func (fielddata *Fielddata) getFielddataStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareCumulativeHelper(ElasticsearchIndicesFielddataEvictions, defaultDims, fielddata.Evictions),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(ElasticsearchIndicesFielddataMemorySize, defaultDims, fielddata.MemorySizeInBytes),
	}...)

	return out
}

func (completion *Completion) getCompletionStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesCompletionSize, defaultDims, completion.SizeInBytes),
		}...)
	}

	return out
}

func (segments *Segments) getSegmentsStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesSegmentsMemorySize, defaultDims, segments.MemoryInBytes),
			prepareGaugeHelper(ElasticsearchIndicesSegmentsIndexWriterMemorySize, defaultDims, segments.IndexWriterMemoryInBytes),
			prepareGaugeHelper(ElasticsearchIndicesSegmentsIndexWriterMaxMemorySize, defaultDims, segments.IndexWriterMaxMemoryInBytes),
			prepareGaugeHelper(ElasticsearchIndicesSegmentsVersionMapMemorySize, defaultDims, segments.VersionMapMemoryInBytes),
			prepareGaugeHelper(ElasticsearchIndicesSegmentsTermsMemorySize, defaultDims, segments.TermsMemoryInBytes),
			prepareGaugeHelper(ElasticsearchIndicesSegmentsStoredFieldMemorySize, defaultDims, segments.StoredFieldsMemoryInBytes),
			prepareGaugeHelper(ElasticsearchIndicesSegmentsTermVectorsMemorySize, defaultDims, segments.TermVectorsMemoryInBytes),
			prepareGaugeHelper(ElasticsearchIndicesSegmentsNormsMemorySize, defaultDims, segments.NormsMemoryInBytes),
			prepareGaugeHelper(ElasticsearchIndicesSegmentsPointsMemorySize, defaultDims, segments.PointsMemoryInBytes),
			prepareGaugeHelper(ElasticsearchIndicesSegmentsDocValuesMemorySize, defaultDims, segments.DocValuesMemoryInBytes),
			prepareGaugeHelper(ElasticsearchIndicesSegmentsFixedBitSetMemorySize, defaultDims, segments.FixedBitSetMemoryInBytes),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(ElasticsearchIndicesSegmentsCount, defaultDims, segments.Count),
	}...)

	return out
}

func (translog *Translog) getTranslogStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesTranslogUncommittedOperations, defaultDims, translog.UncommittedOperations),
			prepareGaugeHelper(ElasticsearchIndicesTranslogUncommittedSizeInBytes, defaultDims, translog.UncommittedSizeInBytes),
			prepareGaugeHelper(ElasticsearchIndicesTranslogEarliestLastModifiedAge, defaultDims, translog.EarliestLastModifiedAge),
			prepareGaugeHelper(ElasticsearchIndicesTranslogOperations, defaultDims, translog.Operations),
			prepareGaugeHelper(ElasticsearchIndicesTranslogSize, defaultDims, translog.SizeInBytes),
		}...)
	}

	return out
}

func (requestCache *RequestCache) getRequestCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareCumulativeHelper(ElasticsearchIndicesRequestCacheEvictions, defaultDims, requestCache.Evictions),
			prepareCumulativeHelper(ElasticsearchIndicesRequestCacheHitCount, defaultDims, requestCache.HitCount),
			prepareCumulativeHelper(ElasticsearchIndicesRequestCacheMissCount, defaultDims, requestCache.MissCount),
		}...)
	}

	out = append(out, prepareGaugeHelper(ElasticsearchIndicesRequestCacheMemorySize, defaultDims, requestCache.MemorySizeInBytes))

	return out
}

func (recovery *Recovery) getRecoveryStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesRecoveryCurrentAsSource, defaultDims, recovery.CurrentAsSource),
			prepareGaugeHelper(ElasticsearchIndicesRecoveryCurrentAsTarget, defaultDims, recovery.CurrentAsTarget),
			prepareCumulativeHelper(ElasticsearchIndicesRecoveryThrottleTime, defaultDims, recovery.ThrottleTimeInMillis),
		}...)
	}

	return out
}

func (idCache *IDCache) getIDCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesIDCacheMemorySize, defaultDims, idCache.MemorySizeInBytes),
		}...)
	}

	return out
}

func (suggest *Suggest) getSuggestStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesSuggestCurrent, defaultDims, suggest.Current),
			prepareCumulativeHelper(ElasticsearchIndicesSuggestTime, defaultDims, suggest.TimeInMillis),
			prepareCumulativeHelper(ElasticsearchIndicesSuggestTotal, defaultDims, suggest.Total),
		}...)
	}

	return out
}

func (percolate *Percolate) getPercolateStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(ElasticsearchIndicesPercolateCurrent, defaultDims, percolate.Current),
			prepareCumulativeHelper(ElasticsearchIndicesPercolateTotal, defaultDims, percolate.Total),
			prepareCumulativeHelper(ElasticsearchIndicesPercolateQueries, defaultDims, percolate.Queries),
			prepareCumulativeHelper(ElasticsearchIndicesPercolateTime, defaultDims, percolate.TimeInMillis),
		}...)
	}

	return out
}
