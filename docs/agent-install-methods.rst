.. _choose-install-type:

*********************************************
Install the Smart Agent
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
needs. Select the one that matches your situation or preference.


+---------------------------------------+----------------------------------------------------------------------------------+
| Situation                             | Procedure                                                                        |
+=======================================+==================================================================================+
| You want a quick test of SignalFx     | `Quick Install <quick-install.md>`__                                             |
+---------------------------------------+----------------------------------------------------------------------------------+
| You want to test monitoring for       | `Install Using Linux Packages <pguidance/agent-install-packages.md>`__           |
| Linux hosts that have Internet access |                                                                                  |
|                                       |                                                                                  |
+---------------------------------------+----------------------------------------------------------------------------------+
| Test Kubernetes monitoring            |                                                                                  |
+---------------------------------------+----------------------------------------------------------------------------------+
| Test µAPM monitoring                  |                                                                                  |
+---------------------------------------+----------------------------------------------------------------------------------+
| Monitor infrastructure in             |                                                                                  |
| beta test or production               |                                                                                  |
+---------------------------------------+----------------------------------------------------------------------------------+
| Monitor Kubernetes in                 |                                                                                  |
| beta test or production               |                                                                                  |
+---------------------------------------+----------------------------------------------------------------------------------+
| Monitor µAPM in                       |                                                                                  |
| beta test or production               |                                                                                  |
+---------------------------------------|----------------------------------------------------------------------------------+

========================
Install to a single host
========================

For a single host, follow the instructions in the Setup section of
the topic .

=======================================
Test infrastructure monitoring
=======================================

Install and configure the Smart Agent on a few test hosts. The Smart Agent sends
data such as cpu usage to SignalFx, and you can view it using visualization tools
such as charts. Use one of these installation methods:

* For Linux hosts that have Internet access, follow the instructions in
  the topic `Install Using Linux Packages <pguidance/agent-install-packages.md>`__.

* For Linux hosts that don't have Internet access, follow the
  instructions in the topic `Install to Linux Using gzip File <pguidance/agent-install-standalone-linux.md>`__.

* For Windows hosts, use `Install to Windows Using zip File <pguidance/agent-install-standalone-windows.md>`__.

.. _test-kubernetes-install:

====================================
Test Kubernetes cluster monitoring
====================================

Install and configure the Smart Agent in a test Kubernetes cluster. The Agent
sends Kubernetes monitoring data to Splunk, which you can visualize using Kubernetes
Navigator.

* If you have Helm installed, follow the instructions in the topic
  `Install Using helm <pguidance/agent-k8s-install-helm.md>`__.

* If you don't have Helm, you can install Smart Agent with kubectl.
  Follow the instructions in the topic `Install Using
  kubectl <pguidance/agent-k8s-install-kubectl.md>`__.

=================================
Test microservices APM (**µAPM**)
=================================

Install and configure the Smart Agent in a test µAPM system. The Agent sends
monitoring data to Splunk, which you can visualize using µAPM tools.

* If you're running your microservices in Docker outside of Kubernetes,
  follow the instructions in the topic `SignalFx Agent Docker
  Image <https://github.com/signalfx/signalfx-agent/tree/master/deployments/docker>`__.

* If you're running your microservices in Kubernetes, install and
  configure Smart Agent according to the instructions in the section
  :ref:`test-kubernetes-install`.

==================================================
Monitor infrastructure in beta test or production
==================================================

If you want to monitor your infrastructure, but you're not using Kubernetes clusters
or APM, use one of the following methods to install the Smart Agent to beta test or
production systems. You can then view incoming metrics in SignalFx:

* To install the Agent in Amazon Web Services
  (**AWS**) Elastic Container Service (**ECS**), see `Install to AWS ECS <pguidance/agent-install-awsecs.md>`__.

* SignalFx provides installation packages for several popular configuration management
  tools such as ``puppet``. To see a list of supported configuration management
  tools, and to learn how to use a supported tool to install the Smart Agent, see
  `Install Smart Agent using Configuration Management <pguidance/agent-install-config-mgmt.md>`__..

* If you don't have a configuration management tool:

  ** **For Linux:** Install the Smart Agent to each host in your system.
     See `Install Using Linux Packages <pguidance/agent-install-packages.md>`__.

     **If your hosts are firewalled for Internet access, download a
     standalone Linux package to a machine that has access and install the
     package to each host. See `Install to Linux Using gzip File <pguidance/agent-install-standalone-linux.md>`__.

  ** **For Windows:** Install the Smart agent to each host in your system.

     **If your hosts are firewalled for Internet access, download a
     standalone Windows package to a machine that has access and install the
     package to each host. See `Install to Windows Using zip File <pguidance/agent-install-standalone-windows.md>`__.


=============================================
Monitor Kubernetes in beta test or production
=============================================

If you want to monitor your infrastructure, and you're also using Kubernetes clusters,
use one of the following methods to install the Smart Agent to beta test or
production systems. You can then view incoming infrastructure metrics. You can also
view Kubernetes metrics and troubleshoot Kubernetes using the Kubernetes Navigator:


* If you have Helm installed, follow the instructions in the topic
  `Install Using helm <pguidance/agent-k8s-install-helm.md>`__.

* If you don't have Helm, you can install Smart Agent with kubectl.
  Follow the instructions in the topic `Install Using
  kubectl <pguidance/agent-k8s-install-kubectl.md>`__.

=======================================
Monitor µAPM in beta test or production
=======================================

To monitor infrastructure, Kubernetes clusters and µAPM,
install the Smart Agent

