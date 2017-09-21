#!/bin/bash

set -e

pip_install="pip install"

# Core agent requirements
$pip_install -r /opt/dd/dd-agent/requirements.txt
$pip_install -r /opt/dd/dd-agent/requirements-opt.txt

# Requirements for each check
for f in $(find /opt/dd/integrations-core | grep requirements.txt)
do
  $pip_install -r $f
done
