.. _install-smart-agent:

.. toctree::
   :hidden:

   /integrations/agent/quick-install
   /integrations/agent/pguidance/agent-install-packages
   /integrations/agent/pguidance/agent-install-standalone-linux
   /integrations/agent/pguidance/agent-install-standalone-windows
   /integrations/agent/pguidance/agent-k8s-install-helm
   /integrations/agent/pguidance/agent-k8s-install-kubectl
   /integrations/agent/pguidance/agent-install-awsecs
   /integrations/agent/pguidance/agent-install-config-mgmt

************************************************
Install and configure the SignalFx Smart Agent
************************************************

You have several different options for installing the SignalFx Smart Agent.
Select the topic that matches your situation or preference. Each
topic includes these sections:

* Prerequisites
* Configuration instructions
* Installation instructions
* Instructions for verifying your installation

:strong:`Evaluate monitoring for a single host`

Evaluate monitoring using a quick installation to a single host

`SignalFx Smart Agent quick installation <./quick-install.html>`__

:strong:`Evaluate monitoring for a specific OS`

.. list-table::
   :header-rows: 1
   :widths: 40 60

   * - :strong:`Goal`
     - :strong:`Procedure`

   * - Evaluate monitoring on \*nix hosts that support Debian or RPM packages
     - `Install the SignalFx Smart Agent using *nix packages <./pguidance/agent-install-packages.html>`__

   * - Evaluate monitoring on Linux hosts that are behind a firewall
     - `Install the SignalFx Smart Agent to Linux using a GZIP file <./pguidance/agent-install-standalone-linux.html>`__

   * - Evaluate monitoring on Windows hosts that are behind a firewall
     - `Install the SignalFx Smart Agent to Windows using a ZIP file <./pguidance/agent-install-standalone-windows.html>`__

:strong:`Evaluate Kubernetes monitoring`

.. list-table::
   :header-rows: 1
   :widths: 40 60

   * - :strong:`Goal`
     - :strong:`Procedure`

   * - Evaluate Kubernetes monitoring using Helm
     - `Install the SignalFx Smart Agent using Helm <./pguidance/agent-k8s-install-helm.html>`__

   * - Evaluate Kubernetes monitoring using kubectl
     - `Install the SignalFx Smart Agent using kubectl <./pguidance/agent-k8s-install-kubectl.html>`__

:strong:`Evaluate SignalFx Microservices APM monitoring`

.. list-table::
   :header-rows: 1
   :widths: 40 60

   * - :strong:`Goal`
     - :strong:`Procedure`

   * - Evaluate SignalFx Microservices APM monitoring for hosts that use Docker outside of Kubernetes
     - `Install the SignalFx Smart Agent using the SignalFx Agent Docker image <https://github.com/signalfx/signalfx-agent/tree/master/deployments/docker>`__

   * - Evaluate SignalFx Microservices APM monitoring for hosts that use Kubernetes
     - If you use Helm, see `Install the SignalFx Smart Agent using Helm <./pguidance/agent-k8s-install-helm.html>`__.

       If you use kubectl, see `Install the SignalFx Smart Agent using kubectl <./pguidance/agent-k8s-install-kubectl.html>`__.

:strong:`Monitor production hosts for a specific OS`

.. list-table::
   :header-rows: 1
   :widths: 40 60

   * - :strong:`Goal`
     - :strong:`Procedure`

   * - Monitor AWS ECS production hosts
     - `Install the SignalFx Smart Agent to AWS ECS <./pguidance/agent-install-awsecs.html>`_

   * - Monitor production hosts that use configuration management tools
     - `Install the SignalFx Smart Agent using configuration management <./pguidance/agent-install-config-mgmt.html>`__

   * - Monitor \*nix production hosts that support Debian or RPM
     - `Install the SignalFx Smart Agent using *nix packages <./pguidance/agent-install-packages.html>`_

   * - Monitor Linux production hosts that are behind a firewall
     - `Install the SignalFx Smart Agent to Linux using a GZIP file <./pguidance/agent-install-standalone-linux.html>`__

   * - Monitor Windows production hosts that are behind a firewall
     - `Install the SignalFx Smart Agent to Windows using a ZIP file <./pguidance/agent-install-standalone-windows.html>`__

:strong:`Monitor Kubernetes production hosts`

.. list-table::
   :header-rows: 1
   :widths: 40 60

   * - :strong:`Goal`
     - :strong:`Procedure`

   * - Monitor Kubernetes production hosts using Helm
     - `Install the SignalFx Smart Agent using Helm <./pguidance/agent-k8s-install-helm.html>`__

   * - Monitor Kubernetes production hosts using kubectl
     - `Install the SignalFx Smart Agent using kubectl <./pguidance/agent-k8s-install-kubectl.html>`__


:strong:`Monitor SignalFx Microservices APM hosts`

`Deploy a SignalFx Smart Agent for µAPM <https://docs.signalfx.com/en/latest/apm2/apm2-getting-started/apm2-smart-agent.html>`__
