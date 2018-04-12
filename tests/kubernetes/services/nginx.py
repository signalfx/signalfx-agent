from tests.kubernetes import utils

def deploy(configmap_data={}, image="nginx:latest", port=80, replicas=3, labels={"app": "nginx"}, namespace="default"):
    if not configmap_data:
        configmap_data = {"default.conf": '''
            server {
                listen 80;
                server_name  localhost;
                location /nginx_status {
                    stub_status on;
                    access_log off;
                    allow all;
                }
            }'''}
    utils.create_configmap(
        name="nginx-status",
        data=configmap_data,
        labels=labels,
        namespace=namespace)
    pod_template = utils.get_pod_template(
        name="nginx",
        image=image,
        port=port,
        labels=labels,
        volume_mounts=[{"name": "nginx-conf", "mount_path": "/etc/nginx/conf.d", "configmap": "nginx-status"}])
    #create_deployment(
    #    name="nginx-deployment",
    #    pod_template=pod_template,
    #    replicas=3,
    #    labels=labels,
    #    namespace=namespace)
    utils.create_replication_controller(
        name="nginx-replication-controller",
        pod_template=pod_template,
        replicas=replicas,
        labels=labels,
        namespace=namespace)
    utils.create_service(
        name="nginx-service",
        port=port,
        service_type="NodePort",
        labels=labels,
        namespace=namespace)
