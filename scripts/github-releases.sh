#!/bin/bash

github_request() {
  local username=$1
  local token=$2
  local method=$3
  local path=$4
  local datafile="${5-}"
  local dataflag=
  if test -n "$datafile"; then
    dataflag="--data-binary @$datafile"
  fi
  curl -f -X "$method" -u $username:$token -H 'Content-Type: application/json' $dataflag "https://api.github.com/repos/signalfx/signalfx-agent/$path"
}

new_github_release() {
  local tag=$1 
  local username=$2
  local token=$3
  local prerelease="false"

  if [[ ! $tag =~ v[0-9]\.[0-9]\.[0-9] ]]; then
    prerelease="true"
  fi

  tmpfile=$(mktemp)
  cat <<EOH > $tmpfile
    {
      "tag_name": "$tag",
      "name": "$tag",
      "prerelease": $(if [[ $tag =~ -beta ]]; then echo -n "true"; else echo -n "false"; fi)
    }
EOH

  github_request "$username" "$token" "POST" "releases" $tmpfile
}

get_github_release() {
  local release_name=$1
  local username=$2
  local token=$3

  github_request "$username" "$token" "GET" "releases/tags/$release_name"
}

upload_asset_to_release() {
  local tag=$1
  local file=$2
  local type=$3
  local username=$4
  local token=$5

  set -x
  upload_url="$(get_github_release "$tag" "$username" "$token" | jq -r '.upload_url' | sed -E -e 's/\{\?.*//')"

  curl -X POST -u $username:$token --data-binary @$file -H "Content-Type: $type" "$upload_url?name=$(basename $file)"
  set +x
}
