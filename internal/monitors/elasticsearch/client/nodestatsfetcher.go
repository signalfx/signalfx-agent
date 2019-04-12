package client

import (
	"github.com/signalfx/golib/datapoint"
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
		prepareGaugeHelper(indicesDocsCount, defaultDims, docs.Count),
		prepareGaugeHelper(indicesDocsDeleted, defaultDims, docs.Deleted),
	}...)

	return out
}

func (store *Store) getStoreStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareCumulativeHelper(indicesStoreThrottleTime, defaultDims, store.ThrottleTimeInMillis),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(indicesStoreSize, defaultDims, store.SizeInBytes),
	}...)

	return out
}

func (indexing Indexing) getIndexingStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesIndexingIndexCurrent, defaultDims, indexing.IndexCurrent),
			prepareGaugeHelper(indicesIndexingIndexFailed, defaultDims, indexing.IndexFailed),
			prepareGaugeHelper(indicesIndexingDeleteCurrent, defaultDims, indexing.DeleteCurrent),
			prepareCumulativeHelper(indicesIndexingIndexTime, defaultDims, indexing.IndexTimeInMillis),
			prepareCumulativeHelper(indicesIndexingDeleteTotal, defaultDims, indexing.DeleteTotal),
			prepareCumulativeHelper(indicesIndexingDeleteTime, defaultDims, indexing.DeleteTimeInMillis),
			prepareCumulativeHelper(indicesIndexingNoopUpdateTotal, defaultDims, indexing.NoopUpdateTotal),
			prepareCumulativeHelper(indicesIndexingThrottledTime, defaultDims, indexing.ThrottleTimeInMillis),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareCumulativeHelper(indicesIndexingIndexTotal, defaultDims, indexing.IndexTotal),
	}...)

	return out
}

func (get *Get) getGetStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesGetCurrent, defaultDims, get.Current),
			prepareCumulativeHelper(indicesGetTime, defaultDims, get.TimeInMillis),
			prepareCumulativeHelper(indicesGetExistsTotal, defaultDims, get.ExistsTotal),
			prepareCumulativeHelper(indicesGetExistsTime, defaultDims, get.ExistsTimeInMillis),
			prepareCumulativeHelper(indicesGetMissingTotal, defaultDims, get.MissingTotal),
			prepareCumulativeHelper(indicesGetMissingTime, defaultDims, get.MissingTimeInMillis),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareCumulativeHelper(indicesGetTotal, defaultDims, get.Total),
	}...)

	return out
}

func (search *Search) getSearchStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesSearchQueryCurrent, defaultDims, search.QueryCurrent),
			prepareGaugeHelper(indicesSearchFetchCurrent, defaultDims, search.FetchCurrent),
			prepareGaugeHelper(indicesSearchScrollCurrent, defaultDims, search.ScrollCurrent),
			prepareGaugeHelper(indicesSearchSuggestCurrent, defaultDims, search.SuggestCurrent),
			prepareGaugeHelper(indicesSearchOpenContexts, defaultDims, search.SuggestCurrent),
			prepareCumulativeHelper(indicesSearchFetchTime, defaultDims, search.FetchTimeInMillis),
			prepareCumulativeHelper(indicesSearchFetchTotal, defaultDims, search.FetchTotal),
			prepareCumulativeHelper(indicesSearchScrollTime, defaultDims, search.ScrollTimeInMillis),
			prepareCumulativeHelper(indicesSearchScrollTotal, defaultDims, search.ScrollTotal),
			prepareCumulativeHelper(indicesSearchSuggestTime, defaultDims, search.SuggestTimeInMillis),
			prepareCumulativeHelper(indicesSearchSuggestTotal, defaultDims, search.SuggestTotal),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareCumulativeHelper(indicesSearchQueryTime, defaultDims, search.QueryTimeInMillis),
		prepareCumulativeHelper(indicesSearchQueryTotal, defaultDims, search.QueryTotal),
	}...)

	return out
}

func (merges *Merges) getMergesStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesMergesCurrentDocs, defaultDims, merges.CurrentDocs),
			prepareGaugeHelper(indicesMergesCurrentSizeInBytes, defaultDims, merges.CurrentSizeInBytes),
			prepareCumulativeHelper(indicesMergesTotalDocs, defaultDims, merges.TotalDocs),
			prepareCumulativeHelper(indicesMergesTotalSizeInBytes, defaultDims, merges.TotalSizeInBytes),
			prepareCumulativeHelper(indicesMergesTotalTime, defaultDims, merges.TotalTimeInMillis),
			prepareCumulativeHelper(indicesMergesTotalStoppedTime, defaultDims, merges.TotalStoppedTimeInMillis),
			prepareCumulativeHelper(indicesMergesTotalThrottledTime, defaultDims, merges.TotalThrottledTimeInMillis),
			prepareCumulativeHelper(indicesMergesTotalAutoThrottleInBytes, defaultDims, merges.TotalAutoThrottleInBytes),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(indicesMergesCurrent, defaultDims, merges.Current),
		prepareCumulativeHelper(indicesMergesTotal, defaultDims, merges.Total),
	}...)

	return out
}

func (refresh *Refresh) getRefreshStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesRefreshListeners, defaultDims, refresh.Listeners),
			prepareCumulativeHelper(indicesRefreshTotal, defaultDims, refresh.Total),
			prepareCumulativeHelper(indicesRefreshTotalTime, defaultDims, refresh.TotalTimeInMillis),
		}...)
	}

	return out
}

func (flush *Flush) getFlushStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesFlushPeriodic, defaultDims, flush.Periodic),
			prepareCumulativeHelper(indicesFlushTotal, defaultDims, flush.Total),
			prepareCumulativeHelper(indicesFlushTotalTime, defaultDims, flush.TotalTimeInMillis),
		}...)
	}

	return out
}

func (warmer *Warmer) getWarmerStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesWarmerCurrent, defaultDims, warmer.Current),
			prepareCumulativeHelper(indicesWarmerTotal, defaultDims, warmer.Total),
			prepareCumulativeHelper(indicesWarmerTotalTime, defaultDims, warmer.TotalTimeInMillis),
		}...)
	}

	return out
}

func (queryCache *QueryCache) getQueryCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesQuerycacheCacheSize, defaultDims, queryCache.CacheSize),
			prepareGaugeHelper(indicesQuerycacheCacheCount, defaultDims, queryCache.CacheCount),
			prepareCumulativeHelper(indicesQuerycacheEvictions, defaultDims, queryCache.Evictions),
			prepareCumulativeHelper(indicesQuerycacheHitCount, defaultDims, queryCache.HitCount),
			prepareCumulativeHelper(indicesQuerycacheMissCount, defaultDims, queryCache.MissCount),
			prepareCumulativeHelper(indicesQuerycacheTotalCount, defaultDims, queryCache.TotalCount),
		}...)
	}
	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(indicesQuerycacheMemorySizeInBytes, defaultDims, queryCache.MemorySizeInBytes),
	}...)

	return out
}

func (filterCache *FilterCache) getFilterCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareCumulativeHelper(indicesFiltercacheEvictions, defaultDims, filterCache.Evictions),
		}...)

		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesFiltercacheMemorySizeInBytes, defaultDims, filterCache.MemorySizeInBytes),
		}...)
	}

	return out
}

func (fielddata *Fielddata) getFielddataStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareCumulativeHelper(indicesFielddataEvictions, defaultDims, fielddata.Evictions),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(indicesFielddataMemorySizeInBytes, defaultDims, fielddata.MemorySizeInBytes),
	}...)

	return out
}

func (completion *Completion) getCompletionStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesCompletionSizeInBytes, defaultDims, completion.SizeInBytes),
		}...)
	}

	return out
}

func (segments *Segments) getSegmentsStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesSegmentsMemory, defaultDims, segments.MemoryInBytes),
			prepareGaugeHelper(indicesSegmentsIndexWriterMemory, defaultDims, segments.IndexWriterMemoryInBytes),
			prepareGaugeHelper(indicesSegmentsMaxIndexWriterMemory, defaultDims, segments.IndexWriterMaxMemoryInBytes),
			prepareGaugeHelper(indicesSegmentsVersionMapMemory, defaultDims, segments.VersionMapMemoryInBytes),
			prepareGaugeHelper(indicesSegmentsTermsMemory, defaultDims, segments.TermsMemoryInBytes),
			prepareGaugeHelper(indicesSegmentsStoredFieldMemory, defaultDims, segments.StoredFieldsMemoryInBytes),
			prepareGaugeHelper(indicesSegmentsTermVectorsMemory, defaultDims, segments.TermVectorsMemoryInBytes),
			prepareGaugeHelper(indicesSegmentsNormsMemory, defaultDims, segments.NormsMemoryInBytes),
			prepareGaugeHelper(indicesSegmentsPointsMemory, defaultDims, segments.PointsMemoryInBytes),
			prepareGaugeHelper(indicesSegmentsDocValuesMemory, defaultDims, segments.DocValuesMemoryInBytes),
			prepareGaugeHelper(indicesSegmentsFixedBitSetMemory, defaultDims, segments.FixedBitSetMemoryInBytes),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(indicesSegmentsCount, defaultDims, segments.Count),
	}...)

	return out
}

func (translog *Translog) getTranslogStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesTranslogUncommittedOperations, defaultDims, translog.UncommittedOperations),
			prepareGaugeHelper(indicesTranslogUncommittedSizeInBytes, defaultDims, translog.UncommittedSizeInBytes),
			prepareGaugeHelper(indicesTranslogEarliestLastModifiedAge, defaultDims, translog.EarliestLastModifiedAge),
			prepareGaugeHelper(indicesTranslogOperations, defaultDims, translog.Operations),
			prepareGaugeHelper(indicesTranslogSize, defaultDims, translog.SizeInBytes),
		}...)
	}

	return out
}

func (requestCache *RequestCache) getRequestCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesRequestcacheMemorySizeInBytes, defaultDims, requestCache.MemorySizeInBytes),
			prepareCumulativeHelper(indicesRequestcacheEvictions, defaultDims, requestCache.Evictions),
			prepareCumulativeHelper(indicesRequestcacheHitCount, defaultDims, requestCache.HitCount),
			prepareCumulativeHelper(indicesRequestcacheMissCount, defaultDims, requestCache.MissCount),
		}...)
	}

	return out
}

func (recovery *Recovery) getRecoveryStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesRecoveryCurrentAsSource, defaultDims, recovery.CurrentAsSource),
			prepareGaugeHelper(indicesRecoveryCurrentAsTarget, defaultDims, recovery.CurrentAsTarget),
			prepareCumulativeHelper(indicesRecoveryThrottleTime, defaultDims, recovery.ThrottleTimeInMillis),
		}...)
	}

	return out
}

func (idCache *IDCache) getIDCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesIdcacheMemorySizeInBytes, defaultDims, idCache.MemorySizeInBytes),
		}...)
	}

	return out
}

func (suggest *Suggest) getSuggestStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesSuggestCurrent, defaultDims, suggest.Current),
			prepareCumulativeHelper(indicesSuggestTime, defaultDims, suggest.TimeInMillis),
			prepareCumulativeHelper(indicesSuggestTotal, defaultDims, suggest.Total),
		}...)
	}

	return out
}

func (percolate *Percolate) getPercolateStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesPercolateCurrent, defaultDims, percolate.Current),
			prepareCumulativeHelper(indicesPercolateTotal, defaultDims, percolate.Total),
			prepareCumulativeHelper(indicesPercolateQueries, defaultDims, percolate.Queries),
			prepareCumulativeHelper(indicesPercolateTime, defaultDims, percolate.TimeInMillis),
		}...)
	}

	return out
}
