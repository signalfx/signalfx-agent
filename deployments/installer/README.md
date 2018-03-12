# SignalFx Agent Installer Script

This directory contains a convenience script that can be used to quickly
install the agent on any supported Linux environment.  This is **not** the
recommended way of installing the agent in production environments, but is
convenient for testing and/or proof of concepts.

The latest version of the built installer script lives at https://<insert URL
here>.

If you are going to run this script in an automated way, we recommend pinning
the agent version by passing the `--agent-version` flag.  This version includes
the package revision (e.g. a `-1`, `-2`, etc.) after the agent version, so be
sure to include that.
