#!/bin/bash

set -eo pipefail

sudo apt-get install jq -y
curl -LOk https://github.com/hairyhenderson/gomplate/releases/download/v3.4.0/gomplate_linux-amd64
sudo mv gomplate_linux-amd64 /usr/bin/gomplate
chmod +x /usr/bin/gomplate
