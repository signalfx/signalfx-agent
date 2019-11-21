#!/bin/bash

set -eo pipefail

apk add --no-cache jq
curl -LOk https://github.com/hairyhenderson/gomplate/releases/download/v3.4.0/gomplate_linux-amd64
mv gomplate_linux-amd64 /usr/bin/gomplate
chmod +x /usr/bin/gomplate
