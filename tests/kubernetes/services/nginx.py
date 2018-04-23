from tests.kubernetes import utils

CONFIG = {
    "name": "nginx",
    "pod_name": "nginx-replication-controller",
    "image": "nginx:latest",
    "port": 80,
    "replicas": 3,
    "labels": {"app": "nginx"},
    "namespace": "default",
}

def deploy():
    configmap_data = {
        "default.conf": """
            server {
                listen %d;
                server_name  localhost;
                location /nginx_status {
                    stub_status on;
                    access_log off;
                    allow all;
                }
            }""" % CONFIG["port"]
    }
    utils.create_configmap(
        name="nginx-status",
        data=configmap_data,
        labels=CONFIG["labels"],
        namespace=CONFIG["namespace"])
    pod_template = utils.get_pod_template(
        name=CONFIG["name"],
        image=CONFIG["image"],
        port=CONFIG["port"],
        labels=CONFIG["labels"],
        volume_mounts=[{"name": "nginx-conf", "mount_path": "/etc/nginx/conf.d", "configmap": "nginx-status"}])
    #create_deployment(
    #    name="nginx-deployment",
    #    pod_template=pod_template,
    #    replicas=3,
    #    labels=labels,
    #    namespace=namespace)
    utils.create_replication_controller(
        name=CONFIG["pod_name"],
        pod_template=pod_template,
        replicas=CONFIG["replicas"],
        labels=CONFIG["labels"],
        namespace=CONFIG["namespace"])
    utils.create_service(
        name="nginx-service",
        port=CONFIG["port"],
        service_type="NodePort",
        labels=CONFIG["labels"],
        namespace=CONFIG["namespace"])

