package client

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// Valid Index stats groups
const (
	DocsStatsGroup         = "docs"
	StoreStatsGroup        = "store"
	IndexingStatsGroup     = "indexing"
	GetStatsGroup          = "get"
	SearchStatsGroup       = "search"
	MergesStatsGroup       = "merges"
	RefreshStatsGroup      = "refresh"
	FlushStatsGroup        = "flush"
	WarmerStatsGroup       = "warmer"
	QueryCacheStatsGroup   = "query_cache"
	FilterCacheStatsGroup  = "filter_cache"
	FieldDataStatsGroup    = "fielddata"
	CompletionStatsGroup   = "completion"
	SegmentsStatsGroup     = "segments"
	TranslogStatsGroup     = "translog"
	RequestCacheStatsGroup = "request_cache"
	RecoveryStatsGroup     = "recovery"
	IDCacheStatsGroup      = "id_cache"
	SuggestStatsGroup      = "suggest"
	PercolateStatsGroup    = "percolate"
)

// ValidIndexStatsGroups is a "set" of valid index stats groups
var ValidIndexStatsGroups = map[string]bool{
	DocsStatsGroup:         true,
	StoreStatsGroup:        true,
	IndexingStatsGroup:     true,
	GetStatsGroup:          true,
	SearchStatsGroup:       true,
	MergesStatsGroup:       true,
	RefreshStatsGroup:      true,
	FlushStatsGroup:        true,
	WarmerStatsGroup:       true,
	QueryCacheStatsGroup:   true,
	FilterCacheStatsGroup:  true,
	FieldDataStatsGroup:    true,
	CompletionStatsGroup:   true,
	SegmentsStatsGroup:     true,
	TranslogStatsGroup:     true,
	RequestCacheStatsGroup: true,
	RecoveryStatsGroup:     true,
	IDCacheStatsGroup:      true,
	SuggestStatsGroup:      true,
	PercolateStatsGroup:    true,
}

// Aggregations types for index stats
const (
	Total     = "total"
	Primaries = "primaries"
)

// GetIndexStatsSummaryDatapoints fetches datapoints for ES Index stats summary aggregated across all indexes
func GetIndexStatsSummaryDatapoints(indexStats IndexStats, defaultDims map[string]string, enhancedStatsForIndexGroupsOption map[string]bool, enablePrimaryIndexStats bool) []*datapoint.Datapoint {
	return getIndexStatsHelper(indexStats, defaultDims, enhancedStatsForIndexGroupsOption, enablePrimaryIndexStats)
}

// GetIndexStatsDatapoints fetches datapoints for ES Index stats per index
func GetIndexStatsDatapoints(indexStatsPerIndex map[string]IndexStats, indexes map[string]bool, defaultDims map[string]string, enhancedStatsForIndexGroupsOption map[string]bool, enablePrimaryIndexStats bool) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint
	collectAllIndexes := len(indexes) == 0

	for indexName, indexStats := range indexStatsPerIndex {
		if !collectAllIndexes && !indexes[indexName] {
			continue
		}

		dims := utils.MergeStringMaps(defaultDims, map[string]string{
			"index": indexName,
		})
		out = append(out, getIndexStatsHelper(indexStats, dims, enhancedStatsForIndexGroupsOption, enablePrimaryIndexStats)...)
	}

	return out
}

func getIndexStatsHelper(indexStats IndexStats, defaultDims map[string]string, enhancedStatsForIndexGroupsOption map[string]bool, enablePrimaryIndexStats bool) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enablePrimaryIndexStats {
		indexStatsGroup := indexStats.Primaries
		defaultDimsForPrimaries := utils.MergeStringMaps(defaultDims, map[string]string{
			"aggregation": Primaries,
		})
		getIndexStatsForAggregation(indexStatsGroup, defaultDimsForPrimaries, enhancedStatsForIndexGroupsOption)
	}

	indexStatsGroup := indexStats.Total
	defaultDimsForTotal := utils.MergeStringMaps(defaultDims, map[string]string{
		"aggregation": Total,
	})
	getIndexStatsForAggregation(indexStatsGroup, defaultDimsForTotal, enhancedStatsForIndexGroupsOption)

	return out
}

func getIndexStatsForAggregation(indexStatsGroup IndexStatsGroups, defaultDims map[string]string, enhancedStatsForIndexGroupsOption map[string]bool) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	out = append(out, indexStatsGroup.Docs.getIndexDocsStats(enhancedStatsForIndexGroupsOption[DocsStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Store.getIndexStoreStats(enhancedStatsForIndexGroupsOption[StoreStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Indexing.getIndexIndexingStats(enhancedStatsForIndexGroupsOption[IndexingStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Get.getIndexGetStats(enhancedStatsForIndexGroupsOption[GetStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Search.getIndexSearchStats(enhancedStatsForIndexGroupsOption[SearchStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Merges.getIndexMergesStats(enhancedStatsForIndexGroupsOption[MergesStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Refresh.getIndexRefreshStats(enhancedStatsForIndexGroupsOption[RefreshStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Flush.getIndexFlushStats(enhancedStatsForIndexGroupsOption[FlushStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Warmer.getIndexWarmerStats(enhancedStatsForIndexGroupsOption[WarmerStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.QueryCache.getIndexQueryCacheStats(enhancedStatsForIndexGroupsOption[QueryCacheStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.FilterCache.getIndexFilterCacheStats(enhancedStatsForIndexGroupsOption[FilterCacheStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Fielddata.getIndexFielddataStats(enhancedStatsForIndexGroupsOption[FieldDataStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Completion.getIndexCompletionStats(enhancedStatsForIndexGroupsOption[CompletionStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Segments.getIndexSegmentsStats(enhancedStatsForIndexGroupsOption[SegmentsStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Translog.getIndexTranslogStats(enhancedStatsForIndexGroupsOption[TranslogStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.RequestCache.getIndexRequestCacheStats(enhancedStatsForIndexGroupsOption[RequestCacheStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Recovery.getIndexRecoveryStats(enhancedStatsForIndexGroupsOption[RecoveryStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.IDCache.getIndexIDCacheStats(enhancedStatsForIndexGroupsOption[IDCacheStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Suggest.getIndexSuggestStats(enhancedStatsForIndexGroupsOption[SuggestStatsGroup], defaultDims)...)
	out = append(out, indexStatsGroup.Percolate.getIndexPercolateStats(enhancedStatsForIndexGroupsOption[PercolateStatsGroup], defaultDims)...)

	return out

}

func (docs *Docs) getIndexDocsStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(indicesDocsCount, defaultDims, docs.Count),
		prepareGaugeHelper(indicesDocsDeleted, defaultDims, docs.Deleted),
	}...)

	return out
}

func (store *Store) getIndexStoreStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (indexing Indexing) getIndexIndexingStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesIndexingIndexCurrent, defaultDims, indexing.IndexCurrent),
			prepareGaugeHelper(indicesIndexingIndexFailed, defaultDims, indexing.IndexFailed),
			prepareGaugeHelper(indicesIndexingDeleteCurrent, defaultDims, indexing.DeleteCurrent),
			prepareCumulativeHelper(indicesIndexingDeleteTotal, defaultDims, indexing.DeleteTotal),
			prepareCumulativeHelper(indicesIndexingDeleteTime, defaultDims, indexing.DeleteTimeInMillis),
			prepareCumulativeHelper(indicesIndexingNoopUpdateTotal, defaultDims, indexing.NoopUpdateTotal),
			prepareCumulativeHelper(indicesIndexingThrottledTime, defaultDims, indexing.ThrottleTimeInMillis),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareCumulativeHelper(indicesIndexingIndexTotal, defaultDims, indexing.IndexTotal),
		prepareCumulativeHelper(indicesIndexingIndexTime, defaultDims, indexing.IndexTimeInMillis),
	}...)

	return out
}

func (get *Get) getIndexGetStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (search *Search) getIndexSearchStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (merges *Merges) getIndexMergesStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesMergesCurrentDocs, defaultDims, merges.CurrentDocs),
			prepareGaugeHelper(indicesMergesCurrentSizeInBytes, defaultDims, merges.CurrentSizeInBytes),
			prepareGaugeHelper(indicesMergesCurrent, defaultDims, merges.Current),
			prepareCumulativeHelper(indicesMergesTotalDocs, defaultDims, merges.TotalDocs),
			prepareCumulativeHelper(indicesMergesTotalSizeInBytes, defaultDims, merges.TotalSizeInBytes),
			prepareCumulativeHelper(indicesMergesTotalStoppedTime, defaultDims, merges.TotalStoppedTimeInMillis),
			prepareCumulativeHelper(indicesMergesTotalThrottledTime, defaultDims, merges.TotalThrottledTimeInMillis),
			prepareCumulativeHelper(indicesMergesTotalAutoThrottleInBytes, defaultDims, merges.TotalAutoThrottleInBytes),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareCumulativeHelper(indicesMergesTotal, defaultDims, merges.Total),
		prepareCumulativeHelper(indicesMergesTotalTime, defaultDims, merges.TotalTimeInMillis),
	}...)

	return out
}

func (refresh *Refresh) getIndexRefreshStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (flush *Flush) getIndexFlushStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (warmer *Warmer) getIndexWarmerStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (queryCache *QueryCache) getIndexQueryCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (filterCache *FilterCache) getIndexFilterCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareCumulativeHelper(indicesFiltercacheEvictions, defaultDims, filterCache.Evictions),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(indicesFiltercacheMemorySizeInBytes, defaultDims, filterCache.MemorySizeInBytes),
	}...)

	return out
}

func (fielddata *Fielddata) getIndexFielddataStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (completion *Completion) getIndexCompletionStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesCompletionSizeInBytes, defaultDims, completion.SizeInBytes),
		}...)
	}

	return out
}

func (segments *Segments) getIndexSegmentsStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (translog *Translog) getIndexTranslogStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (requestCache *RequestCache) getIndexRequestCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareCumulativeHelper(indicesRequestcacheEvictions, defaultDims, requestCache.Evictions),
			prepareCumulativeHelper(indicesRequestcacheHitCount, defaultDims, requestCache.HitCount),
			prepareCumulativeHelper(indicesRequestcacheMissCount, defaultDims, requestCache.MissCount),
		}...)
	}

	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper(indicesRequestcacheMemorySizeInBytes, defaultDims, requestCache.MemorySizeInBytes),
	}...)

	return out
}

func (recovery *Recovery) getIndexRecoveryStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (idCache *IDCache) getIndexIDCacheStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper(indicesIdcacheMemorySizeInBytes, defaultDims, idCache.MemorySizeInBytes),
		}...)
	}

	return out
}

func (suggest *Suggest) getIndexSuggestStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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

func (percolate *Percolate) getIndexPercolateStats(enhanced bool, defaultDims map[string]string) []*datapoint.Datapoint {
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
