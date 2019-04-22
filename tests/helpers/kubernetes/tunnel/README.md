# Fake Backend Tunnel

This directory contains a Docker image and Kubernetes pod spec that launches a
container that agents running in an external Kubernetes cluster (or any remote
environment separate from the test machine, it doesn't have to be Kubernetes).

It works by running an ssh daemon that accepts connections from the local test
box that does remote TCP tunneling from the pod to the local host.  The script
`client.sh` is run by pytest to establish that tunnel.
