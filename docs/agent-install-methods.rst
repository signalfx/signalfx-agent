.. _install-smart-agent:

.. |sfxagt| replace:: the Smart Agent
.. |signalfxagent| replace:: the SignalFx Smart Agent
.. |unix| replace:: *nix
.. |debian| replace:: Debian
.. |rpm| replace:: RPM
.. |linux| replace:: Linux
.. |helm| replace:: Helm
.. |kubectl| replace:: kubectl
.. |win| replace:: Windows
.. |microapm| replace:: SignalFx Microservices APM
.. |mapm| replace:: µAPM

*********************************************
Install and configure |signalfxagent|
*********************************************

.. toctree::
   :maxdepth: 1
   :hidden:

   /integrations/agent/quick-install
   /integrations/agent/pguidance/agent-install-packages
   /integrations/agent/pguidance/agent-install-standalone-linux
   /integrations/agent/pguidance/agent-install-standalone-windows
   /integrations/agent/pguidance/agent-k8s-install-helm
   /integrations/agent/pguidance/agent-k8s-install-kubectl
   /integrations/agent/pguidance/agent-install-awsecs
   /integrations/agent/pguidance/agent-install-config-mgmt

You have several different options for installing |signalfxagent|.
Select the topic that matches your situation or preference. Each
topic includes these sections:

* Prerequisites
* Configuration instructions
* Installation instructions
* Instructions for verifying your installation

Evaluate monitoring for a single host
=======================================

Evaluate monitoring using a quick installation to a single host

`Quick Install <./quick-install.html>`__

Evaluate monitoring for a specific OS
=======================================

.. list-table::
   :header-rows: 1
   :widths: 60 40

   * - :strong:`Goal`
     - :strong:`Procedure`

   * - :strong:`Evaluate on |unix| hosts` that support |debian| or |rpm| packages
     - `Install |signalfxagent| using |unix| packages <./pguidance/agent-install-packages.html>`__

   * - :strong:`Evaluate on |linux| hosts` that are behind a firewall
     - `Install |signalfxagent| to Linux using a GZIP file <./pguidance/agent-install-standalone-linux.html>`__

   * - :strong:`Evaluate on Windows hosts` that are behind a firewall
     - `Install |signalfxagent| to |win| using a ZIP file <./pguidance/agent-install-standalone-windows.html>`__

Evaluate Kubernetes monitoring
================================

.. list-table::
   :header-rows: 1
   :widths: 60 40

   * - :strong:`Evaluate Kubernetes monitoring` using |helm| to install |signalfxagent|
     - `Install |signalfxagent| using |helm| <./pguidance/agent-k8s-install-helm.html>`__

   * - :strong:`Evaluate Kubernetes monitoring` using |kubectl| to install |signalfxagent|
     - `Install |signalfxagent| using kubectl <./pguidance/agent-k8s-install-kubectl.html>`__


Evaluate |microapm| monitoring
================================

.. list-table::
   :header-rows: 1
   :widths: 60 40

   * - :strong:`Evaluate |microapm| monitoring` for hosts that use Docker outside of Kubernetes
     - Install |signalfxagent| using the `SignalFx Agent Docker image <https://github.com/signalfx/signalfx-agent/tree/master/deployments/docker>`__

   * - :strong:`Evaluate |microapm| monitoring` for hosts that use Kubernetes
     - If you use Helm: `Install |signalfxagent| using |helm| <./pguidance/agent-k8s-install-helm.html>`__

       If you use kubectl: `Install |signalfxagent| using kubectl <./pguidance/agent-k8s-install-kubectl.html>`__

Monitor production hosts for a specific OS
============================================

.. list-table::
   :header-rows: 1
   :widths: 60 40

   * - :strong:`Monitor AWS ECS production hosts`
     - `Install |signalfxagent| to AWS ECS <./pguidance/agent-install-awsecs.html>`_

   * - :strong:`Monitor production hosts` that use configuration management tools
     - `Install |signalfxagent|  using configuration management <./pguidance/agent-install-config-mgmt.html>`__

   * - :strong:`Monitor *nix production hosts` that support |debian| or |rpm|
     - `Install Using *nix Packages <./pguidance/agent-install-packages.html>`_

   * - :strong:`Monitor Linux production hosts` that are behind a firewall
     - `Install |signalfxagent| to Linux using a GZIP file <./pguidance/agent-install-standalone-linux.html>`__

   * - :strong:`Monitor Windows production hosts` that are behind a firewall
     - `Install |signalfxagent| to Windows using a ZIP file <./pguidance/agent-install-standalone-windows.html>`__

Monitor Kubernetes production hosts
=====================================

.. list-table::
   :header-rows: 1
   :widths: 60 40

   * - :strong:`Monitor Kubernetes production hosts` using |helm|
     - `Install |signalfxagent| using |helm| <./pguidance/agent-k8s-install-helm.html>`__

   * - :strong:`Monitor Kubernetes production hosts` using |kubectl|
     - `Install |signalfxagent| using kubectl <./pguidance/agent-k8s-install-kubectl.html>`__


Monitor |microapm|
====================

`Installing |signalfxagent| <https://docs.signalfx.com/en/latest/apm2/apm2-getting-started/apm2-smart-agent.html>`__
