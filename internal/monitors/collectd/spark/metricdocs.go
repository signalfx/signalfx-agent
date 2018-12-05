package spark

// COUNTER(counter.HiveExternalCatalog.counter.HiveClientCalls): Total number of client calls sent to Hive for query processing

// COUNTER(counter.HiveExternalCatalog.fileCacheHits): Total number of file level cache hits occurred

// COUNTER(counter.HiveExternalCatalog.filesDiscovered): Total number of files discovered

// COUNTER(counter.HiveExternalCatalog.parallelListingJobCount): Total number of Hive-specific jobs running in parallel

// COUNTER(counter.HiveExternalCatalog.partitionsFetched): Total number of partitions fetched

// COUNTER(counter.spark.driver.completed_tasks): Total number of completed tasks in driver mapped to a particular application

// COUNTER(counter.spark.driver.disk_used): Amount of disk used by driver mapped to a particular application

// COUNTER(counter.spark.driver.failed_tasks): Total number of failed tasks in driver mapped to a particular application

// COUNTER(counter.spark.driver.memory_used): Amount of memory used by driver mapped to a particular application

// COUNTER(counter.spark.driver.total_duration): Fraction of time spent by driver mapped to a particular application

// COUNTER(counter.spark.driver.total_input_bytes): Number of input bytes in driver mapped to a particular application

// COUNTER(counter.spark.driver.total_shuffle_read): Size read during a shuffle in driver mapped to a particular application

// COUNTER(counter.spark.driver.total_shuffle_write): Size written to during a shuffle in driver mapped to a particular application

// COUNTER(counter.spark.driver.total_tasks): Total number of tasks in driver mapped to a particular application

// COUNTER(counter.spark.executor.completed_tasks): Completed tasks across executors working for a particular application

// COUNTER(counter.spark.executor.disk_used): Amount of disk used across executors working for a particular application

// COUNTER(counter.spark.executor.failed_tasks): Failed tasks across executors working for a particular application

// COUNTER(counter.spark.executor.memory_used): Amount of memory used across executors working for a particular application

// COUNTER(counter.spark.executor.total_duration): Fraction of time spent across executors working for a particular application

// COUNTER(counter.spark.executor.total_input_bytes): Number of input bytes across executors working for a particular application

// COUNTER(counter.spark.executor.total_shuffle_read): Size read during a shuffle in a particular application's executors

// COUNTER(counter.spark.executor.total_shuffle_write): Size written to during a shuffle in a particular application's executors

// COUNTER(counter.spark.executor.total_tasks): Total tasks across executors working for a particular application

// COUNTER(counter.spark.streaming.num_processed_records): Number of processed records in a streaming application

// COUNTER(counter.spark.streaming.num_received_records): Number of received records in a streaming application

// COUNTER(counter.spark.streaming.num_total_completed_batches): Number of batches completed in a streaming application

// GAUGE(gauge.jvm.MarkSweepCompact.count): Garbage collection count

// GAUGE(gauge.jvm.MarkSweepCompact.time): Garbage collection time

// GAUGE(gauge.jvm.heap.committed): Amount of committed heap memory (in MB)

// GAUGE(gauge.jvm.heap.used): Amount of used heap memory (in MB)

// GAUGE(gauge.jvm.non-heap.committed): Amount of committed non-heap memory (in MB)

// GAUGE(gauge.jvm.non-heap.used): Amount of used non-heap memory (in MB)

// GAUGE(gauge.jvm.pools.Code-Cache.committed): Amount of memory committed for compilation and storage of native code

// GAUGE(gauge.jvm.pools.Code-Cache.used): Amount of memory used to compile and store native code

// GAUGE(gauge.jvm.pools.Compressed-Class-Space.committed): Amount of memory committed for compressing a class object

// GAUGE(gauge.jvm.pools.Compressed-Class-Space.used): Amount of memory used to compress a class object

// GAUGE(gauge.jvm.pools.Eden-Space.committed): Amount of memory committed for the initial allocation of objects

// GAUGE(gauge.jvm.pools.Eden-Space.used): Amount of memory used for the initial allocation of objects

// GAUGE(gauge.jvm.pools.Metaspace.committed): Amount of memory committed for storing classes and classloaders

// GAUGE(gauge.jvm.pools.Metaspace.used): Amount of memory used to store classes and classloaders

// GAUGE(gauge.jvm.pools.Survivor-Space.committed): Amount of memory committed specifically for objects that have survived GC of the Eden Space

// GAUGE(gauge.jvm.pools.Survivor-Space.used): Amount of memory used for objects that have survived GC of the Eden Space

// GAUGE(gauge.jvm.pools.Tenured-Gen.committed): Amount of memory committed to store objects that have lived in the survivor space for a given period of time

// GAUGE(gauge.jvm.pools.Tenured-Gen.used): Amount of memory used for objects that have lived in the survivor space for a given period of time

// GAUGE(gauge.jvm.total.committed): Amount of committed JVM memory (in MB)

// GAUGE(gauge.jvm.total.used): Amount of used JVM memory (in MB)

// GAUGE(gauge.master.aliveWorkers): Total functioning workers

// GAUGE(gauge.master.apps): Total number of active applications in the spark cluster

// GAUGE(gauge.master.waitingApps): Total number of waiting applications in the spark cluster

// GAUGE(gauge.master.workers): Total number of workers in spark cluster

// GAUGE(gauge.spark.driver.active_tasks): Total number of active tasks in driver mapped to a particular application

// GAUGE(gauge.spark.driver.max_memory): Maximum memory used by driver mapped to a particular application

// GAUGE(gauge.spark.driver.rdd_blocks): Number of RDD blocks in the driver mapped to a particular application

// GAUGE(gauge.spark.executor.active_tasks): Total number of active tasks across all executors working for a particular application

// GAUGE(gauge.spark.executor.count): Total number of executors performing for an active application in the spark cluster

// GAUGE(gauge.spark.executor.max_memory): Max memory across all executors working for a particular application

// GAUGE(gauge.spark.executor.rdd_blocks): Number of RDD blocks across all executors working for a particular application

// GAUGE(gauge.spark.job.num_active_stages): Total number of active stages for an active application in the spark cluster

// GAUGE(gauge.spark.job.num_active_tasks): Total number of active tasks for an active application in the spark cluster

// GAUGE(gauge.spark.job.num_completed_stages): Total number of completed stages for an active application in the spark cluster

// GAUGE(gauge.spark.job.num_completed_tasks): Total number of completed tasks for an active application in the spark cluster

// GAUGE(gauge.spark.job.num_failed_stages): Total number of failed stages for an active application in the spark cluster

// GAUGE(gauge.spark.job.num_failed_tasks): Total number of failed tasks for an active application in the spark cluster

// GAUGE(gauge.spark.job.num_skipped_stages): Total number of skipped stages for an active application in the spark cluster

// GAUGE(gauge.spark.job.num_skipped_tasks): Total number of skipped tasks for an active application in the spark cluster

// GAUGE(gauge.spark.job.num_tasks): Total number of tasks for an active application in the spark cluster

// GAUGE(gauge.spark.num_active_stages): Total number of active stages for an active application in the spark cluster

// GAUGE(gauge.spark.num_running_jobs): Total number of running jobs for an active application in the spark cluster

// GAUGE(gauge.spark.stage.disk_bytes_spilled): Actual size written to disk for an active application in the spark cluster

// GAUGE(gauge.spark.stage.executor_run_time): Fraction of time spent by (and averaged across) executors for a particular application

// GAUGE(gauge.spark.stage.input_bytes): Input size for a particular application

// GAUGE(gauge.spark.stage.input_records): Input records received for a particular application

// GAUGE(gauge.spark.stage.memory_bytes_spilled): Size spilled to disk from memory for an active application in the spark cluster

// GAUGE(gauge.spark.stage.output_bytes): Output size for a particular application

// GAUGE(gauge.spark.stage.output_records): Output records written to for a particular application

// GAUGE(gauge.spark.stage.shuffle_read_bytes): Read size during shuffle phase for a particular application

// GAUGE(gauge.spark.stage.shuffle_read_records): Number of records read during shuffle phase for a particular application

// GAUGE(gauge.spark.stage.shuffle_write_bytes): Size written during shuffle phase for a particular application

// GAUGE(gauge.spark.stage.shuffle_write_records): Number of records written to during shuffle phase for a particular application

// GAUGE(gauge.spark.streaming.avg_input_rate): Average input rate of records across retained batches in a streaming application

// GAUGE(gauge.spark.streaming.avg_processing_time): Average processing time in a streaming application

// GAUGE(gauge.spark.streaming.avg_scheduling_delay): Average scheduling delay in a streaming application

// GAUGE(gauge.spark.streaming.avg_total_delay): Average total delay in a streaming application

// GAUGE(gauge.spark.streaming.num_active_batches): Number of active batches in a streaming application

// GAUGE(gauge.spark.streaming.num_inactive_receivers): Number of inactive receivers in a streaming application

// GAUGE(gauge.worker.coresFree): Total cores free for a particular worker process

// GAUGE(gauge.worker.coresUsed): Total cores used by a particular worker process

// GAUGE(gauge.worker.executors): Total number of executors for a particular worker process

// GAUGE(gauge.worker.memFree_MB): Total memory free for a particular worker process

// GAUGE(gauge.worker.memUsed_MB): Memory used by a particular worker process
