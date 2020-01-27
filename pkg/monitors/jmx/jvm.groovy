Map dims(Map args) {
    // so that dashboard will import
    d = ["plugin": "GenericJMX"]
    d.putAll(args ?: [:])
    d
}
dps = [
    util.makeGauge("gauge.jvm.threads.count", util.queryJMX("java.lang:type=Threading").first().ThreadCount, dims()),
    util.makeGauge("gauge.loaded_classes", util.queryJMX("java.lang:type=ClassLoading").first().LoadedClassCount, dims()),
]
mem = util.queryJMX("java.lang:type=Memory").first()
memoryPool = util.queryJMX("java.lang:type=MemoryPool,*")
for (i in ["committed", "init", "max", "used"]) {
    metric = "jmx_memory.$i"
    dps << util.makeGauge(metric, mem.HeapMemoryUsage."$i", dims(plugin_instance: "memory-heap"))
    dps << util.makeGauge(metric, mem.NonHeapMemoryUsage."$i", dims(plugin_instance: "memory-nonheap"))
    for (bean in memoryPool) {
        dps << util.makeGauge(metric, bean.Usage."$i", dims(plugin_instance: "memory_pool-${bean.Name}"))
    }
}
for (bean in util.queryJMX("java.lang:type=GarbageCollector,*")) {
    dps << util.makeCumulative("invocations", bean.CollectionCount, dims(plugin_instance:  "gc-${bean.Name}"))
    dps << util.makeCumulative("total_time_in_ms.collection_time", bean.CollectionTime, dims(plugin_instance: "gc-${bean.Name}"))
}
output.sendDatapoints(dps)
