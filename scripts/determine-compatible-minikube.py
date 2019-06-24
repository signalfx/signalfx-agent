#!/usr/bin/env python

from __future__ import print_function

import sys


def versiontuple(v):
    """
    Returns (x, y, z) tuple for the given version.
    """
    return tuple(map(int, (v.split("."))))


MINIKUBE_LOCALKUBE_VERSION = versiontuple("0.28.2")
MINIKUBE_KUBEADM_VERSION = versiontuple("1.2.0")
K8S_MIN_VERSION = versiontuple("1.7.0")
K8S_MIN_KUBEADM_VERSION = versiontuple("1.11.0")


def determine_compatible(k8s_version):
    """
    Given a kubernetes version, determine a version of minikube which is
    compatible with it.
    """
    if k8s_version.lower() == "latest" or versiontuple(k8s_version.lstrip("v")) >= K8S_MIN_KUBEADM_VERSION:
        ver = MINIKUBE_KUBEADM_VERSION
    else:
        ver = MINIKUBE_LOCALKUBE_VERSION

    return ver


if __name__ == "__main__":
    ver = ".".join(map(str, determine_compatible(sys.argv[1])))
    print("v" + ver)
