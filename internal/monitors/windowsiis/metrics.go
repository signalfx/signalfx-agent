package windowsiis

// GAUGE(web_service.current_connections): Number of current connections to the
// web service

// GAUGE(web_service.connection_attempts_sec): Rate that connections to web
// service are attempted Requests

// GAUGE(web_service.post_requests_sec): Rate of HTTP POST requests

// GAUGE(web_service.get_requests_sec): Rate of HTTP GET requests

// GAUGE(web_service.total_method_requests_sec): Rate at which all HTTP requests
// are received

// GAUGE(web_service.bytes_received_sec): Rate that data is received by web
// service

// GAUGE(web_service.bytes_sent_sec): Rate that data is sent by web service

// GAUGE(web_service.files_received_sec): Rate at which files are received by
// web service

// GAUGE(web_service.files_sent_sec): Rate at which files are sent by web
// service

// GAUGE(web_service.not_found_errors_sec): Rate of 'Not Found' Errors

// GAUGE(web_service.anonymous_users_sec): Rate at which users are making
// anonymous requests to the web service

// GAUGE(web_service.nonanonymous_users_sec): Rate at which users are making
// nonanonymous requests to the web service

// GAUGE(web_service.service_uptime): Service uptime

// GAUGE(web_service.isapi_extension_requests_sec): Rate of ISAPI extension
// request processed simultaneously by the web service

// GAUGE(process.handle_count): The total number of handles currently open by
// this process. This number is equal to the sum of the handles currently open
// by each thread in this process.

// GAUGE(process.pct_processor_time): The percentage of elapsed time that all
// process threads used the processor to execution instructions. Code executed
// to handle some hardware interrupts and trap conditions are included in this
// count.

// GAUGE(process.id_process): The unique identifier of this process. ID Process
// numbers are reused, so they only identify a process for the lifetime of that
// process.

// GAUGE(process.private_bytes): The current size, in bytes, of memory that this
// process has allocated that cannot be shared with other processes.

// GAUGE(process.thread_count): The number of threads currently active in this
// process. Every running process has at least one thread.

// GAUGE(process.virtual_bytes): The current size, in bytes, of the virtual
// address space the process is using. Use of virtual address space does not
// necessarily imply corresponding use of either disk or main memory pages.
// Virtual space is finite, and the process can limit its ability to load
// libraries.

// GAUGE(process.working_set): The current size, in bytes, of the Working Set of
// this process. The Working Set is the set of memory pages touched recently by
// the threads in the process. If free memory in the computer is above a
// threshold, pages are left in the Working Set of a process even if they are
// not in use. When free memory falls below a threshold, pages are trimmed from
// Working Sets. If they are needed, they will then be soft-faulted back into
// the Working Set before leaving main memory.
