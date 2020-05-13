# Serverless Kubernetes Deployment

This is a set of (mostly) ready to go resources to deploy the agent in a
serverless environment.  This is tailored to the AWS EKS model but might work
generally in serverless K8s environments where there is no access to the nodes
and where DaemonSets are not allowed.

This will deploy a single instance of the agent using a Deployment.  That
single deployment will monitor everything in the cluster in a centralized
manner.
