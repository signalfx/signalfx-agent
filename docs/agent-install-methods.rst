.. _install-smart-agent:

*********************************************
Install and Configure the Smart Agent
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

SignalFx offers several different installation mechanisms to match your
needs. Select the topic that matches your situation or preference. Each
topic includes:

* Prerequisites
* Configuration instructions
* Installation instructions
* Instructions for verifying your installation

.. list-table::
   :header-rows: 1
   :widths: 60 40

   * - :strong:`Goal`
     - :strong:`Procedure`

   * - :strong:`Evaluate monitoring` using a quick install to a single host
     - `Quick Install <quick-install.md>`__

   * - :strong:`Evaluate monitoring on *nix hosts` that support Debian or RPG packages
     - `Install Using Linux Packages <pguidance/agent-install-packages.md>`__

   * - :strong:`Evaluate monitoring on Linux hosts` that are behind a firewall
     - `Install to Linux Using gzip File <pguidance/agent-install-standalone-linux.md>`__

   * - :strong:`Evaluate monitoring on Windows hosts` that are behind a firewall
     - `Install to Windows Using zip File <pguidance/agent-install-standalone-windows.md>`__

   * - :strong:`Evaluate Kubernetes monitoring` using Helm to install the Smart Agent
     - `Install Using helm <pguidance/agent-k8s-install-helm.md>`__

   * - :strong:`Evaluate Kubernetes monitoring` using kubectl to install the Smart Agent
     - `Install Using kubectl <pguidance/agent-k8s-install-kubectl.md>`__

   * - :strong:`Evaluate µAPM monitoring` for hosts that use Docker **outside of** Kubernetes
     - Install using the `SignalFx Agent Docker Image <https://github.com/signalfx/signalfx-agent/tree/master/deployments/docker>`__

   * - :strong:`Evaluate µAPM monitoring` for hosts that use Kubernetes
     - If you use Helm: `Install Using helm <pguidance/agent-k8s-install-helm.md>`__.

       If you use kubectl: `Install Using kubectl <pguidance/agent-k8s-install-kubectl.md>`__.

   * - :strong:`Monitor AWS ECS production hosts` by installing the Smart Agent to AWS ECS
     - `Install to AWS ECS <pguidance/agent-install-awsecs.md>`__

   * - :strong:`Monitor production hosts` that use configuration management tools
     - `Install Smart Agent using Configuration Management <pguidance/agent-install-config-mgmt.md>`__.

   * - :strong:`Monitor *nix production hosts` that support Debian or RPG
     - `Install Using Linux Packages <pguidance/agent-install-packages.md>`__

   * - :strong:`Monitor Linux production hosts` that are behind a firewall
     - `Install to Linux Using gzip File <pguidance/agent-install-standalone-linux.md>`__

   * - :strong:`Monitor Windows production hosts` that are behind a firewall
     - `Install to Windows Using zip File <pguidance/agent-install-standalone-windows.md>`__

   * - :strong:`Monitor Kubernetes production hosts`, using Helm to install the Smart Agent
     - `Install Using helm <pguidance/agent-k8s-install-helm.md>`__

   * - :strong:`Monitor Kubernetes hosts`, using kubectl to install the Smart Agent
     - `Install Using kubectl <pguidance/agent-k8s-install-kubectl.md>`__

   * - :strong:`Monitor µAPM hosts`
     - `Installing the SignalFx Smart Agent <https://docs.signalfx.com/en/latest/apm2/apm2-getting-started/apm2-smart-agent.html>`__
