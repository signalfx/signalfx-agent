# Deploying SignalFx with prometheus-operator

[kube-prometheus](https://github.com/prometheus-operator/kube-prometheus) is a complete solution to help collect Prometheus metrics and federate them to a Prometheus server.

Using the SignalFx agent, we aim to reuse this architecture and poll the master Prometheus server for data.

Note this may introduce delay in metric reporting and may not always be the preferred option.

This sample uses Minikube to install prometheus-operator, installs SignalFx and points it at Splunk for collection.

## Dependencies

* Docker 19.03 or later
* Minikube 1.12 or later
* Helm v3 or later

## Start the sample

Open a terminal window.

###Start a minikube environment
```bash
$> minikube start --driver=docker
```
### Deploy prometheus-operator
```bash
$> helm repo add stable https://kubernetes-charts.storage.googleapis.com
$> helm repo update
$> helm upgrade --install prometheus-operator --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false stable/prometheus-operator
```

### Deploy Splunk

Note: this section assumes you would want to deploy Splunk alongside in Minikube for evaluation purposes. Please skip if you deployed Splunk already.

Check [splunk-kube/templates/configmap.yaml](splunk-kube/templates/configmap.yaml).

Note the following:

We are going to create a metrics index named `sfx`:
```yaml
        indexes:
          directory: /opt/splunk/etc/apps/search/local
          content:
            sfx:
              datatype: metric
              coldPath: $SPLUNK_DB/sfx/colddb
              homePath: $SPLUNK_DB/sfx/db
              thawedPath: $SPLUNK_DB/sfx/thaweddb
```

We create a HEC data input:
```yaml
        inputs:
          directory: /opt/splunk/etc/apps/search/local
          content:
            http://signalfx-agent:
              disabled: 0
              index: sfx
              token: 12345678-ABCD-EFGH-IJKL-123456789012
```

Now use helm to install Splunk:
```bash
$> helm install splunk ./splunk-kube  -f ./splunk-kube/splunk.yaml
```

###Deploy the signalfx-agent daemonset

We configure the SignalFx agent with the [values file signalfx-values.yml](./signalfx-values.yaml) to send data to the local Splunk installation.

Note: change as required to target your remote Splunk installation, if applicable.

```yaml
    splunk:
      enabled: true
      url: https://splunk-splunk-kube:8088/services/collector
      token: 12345678-ABCD-EFGH-IJKL-123456789012
      index: sfx
      source: sfx
      eventsIndex: traces
      eventsSource: traces
      skipTLSVerify: true
```

We also configure the agent to connect to the Prometheus pod:
```yaml
    - type: prometheus-exporter
      discoveryRule: kubernetes_pod_name =~ "prometheus-prometheus" && target == "pod"
      extraDimensions:
        metric_source: prometheus
      host: prometheus
      metricPath: /federate?match[]=%7Bendpoint%3D%22operations%22%7D
      port: 9090
```

See the metric path is going to the `/federate` endpoint, associated with a PromQL query.

The query in this sample just captures the `up` metric. You will want to replace this with something meaningful to your deployment.

Please note you will need to escape PromQL as a query string parameter: the query `{endpoint="operations"}` is encoded as `%7Bendpoint%3D%22operations%22%7D`.

To learn more about Prometheus federation, see [here](https://prometheus.io/docs/prometheus/latest/federation/).

To learn more about PromQL, see [the Prometheus docs](https://prometheus.io/docs/prometheus/latest/querying/basics/).

You can also choose to enable sending to signalfx.com or to only send data to Splunk.

By default, we only send data to Splunk by adding the values file `disable-signalfx.com.yaml`.

If you would like to enable sending to signalfx.com, adapt `enable-signalfx.com.yaml.tmpl` to add your token and realm information.

Use Helm to install SignalFx:
```bash
$> helm repo add signalfx https://dl.signalfx.com/helm-repo
$> helm repo update
$> helm upgrade --install signalfx-agent -f signalfx-values.yaml -f disable-signalfx.com.yaml signalfx/signalfx-agent
```

### Check the up metric

Our system should now report to Splunk. Enter `minikube service splunk` to map the Splunk port 8000 to a host port.

Proceed to the Analytics Workspace (`/en-US/app/search/analytics_workspace`) and look for the `up` metric.

### Stop the sample

Delete minikube:
```bash
$> minikube delete
```