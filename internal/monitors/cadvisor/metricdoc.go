package cadvisor

// TIMESTAMP(container_last_seen): Last time a container was seen by the exporter
// COUNTER(container_cpu_user_seconds_total): Cumulative user cpu time consumed in nanoseconds

// DIMENSION(kubernetes_namespace): The K8s namespace the container is part of
// DIMENSION(kubernetes_pod_name): The pod instance under which this container runs
// DIMENSION(kubernetes_pod_uid): The UID of the pod instance under which this container runs
// DIMENSION(container_spec_name): The container's name as it appears in the pod spec
// DIMENSION(container_name): The container's name as it appears in the pod spec, the same as container_spec_name but retained for backwards compatibility.
// DIMENSION(container_id): The ID of the running container
// DIMENSION(container_image): The container image name
