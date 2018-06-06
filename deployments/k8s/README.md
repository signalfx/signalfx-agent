# Kubernetes Deployments

The agent runs in Kubernetes and has some monitors and observers specific to
that environment.  

The resources in this directory can be used to deploy the agent to K8s.  They
are generated from our [Helm](https://github.com/kubernetes/helm) chart,
which is available in the main [Helm Charts
repo](https://github.com/kubernetes/charts/tree/master/stable/signalfx-agent).

A few things to do before deploying these:

 1. Make sure you change the `kubernetes_cluster` global dimension to something
	specific to your cluster in the `configmap.yaml` resource before deploying.

 2. Also make sure you change the `namespace` of the service account token
	reference in [./clusterrolebinding.yaml](./clusterrolebinding.yaml) to the
	namespace in which you are deploying the agent.

 3. Create a secret in K8s with your org's access token:

	`kubectl create secret generic --from-literal access-token=MY_ACCESS_TOKEN signalfx-agent`

Then to deploy run the following from the present directory:

`cat *.yaml | kubectl apply -f -`


## Development

These resources can be refreshed from the Helm chart by using the
`generate-from-helm` script in this dir.
