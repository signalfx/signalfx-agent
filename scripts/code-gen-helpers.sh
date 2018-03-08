#!/bin/bash

# Some helper functions to make templating from the shell easier to do.

AGENT_BIN=${AGENT_BIN:-signalfx-agent}

if [ -z ${json_file-} ]; then
  json_file=$(mktemp -t XXXXXXX.json)
fi

ensure_description_json() {
  if ! [ -e $json_file ] || ! [ -s $json_file ]; then
    $AGENT_BIN selfdescribe > $json_file
  fi
}

seq_from_len() {
  seq 0 $(($(j "$1 | length")-1))
}

words_to_json_array() {
  jq -nR '[inputs]'
}

escape_newlines() {
  #echo -n $(sed -e 's/$/\\n/' | tr -d '\n')
  sed -E ':a;N;$!ba;s/\r{0,1}\n/\\n/g'
}

inject_str_to_obj() {
  # Bash makes nested double quotes very painful
  j ". | map(. + {$1: "'"'"$2"'"'"}'"
}

inject_to_obj() {
  jq -r ". + {$1: $2}"
}

# This converts camelCase strings to underscore separation in such a way that
# it can be inverted back to camelCase deterministically.
camel_case_to_underscore() {
  perl -pe 's/([a-z0-9])([A-Z])([a-z])/$1_\L$2$3/g' | \
  perl -pe 's/([a-z0-9])([A-Z])([A-Z])/$1_$2$3/' | \
  perl -pe 's/([A-Z])([A-Z])([a-z])/$1_\L$2$3/'
}

# Ensure we have an agent self description json and run a query against it
# with jq
j() {
  ensure_description_json
  jq -r "$1" < $json_file
}
