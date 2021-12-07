#!/bin/bash

# Some helper functions to make templating from the shell easier to do.

AGENT_BIN=${AGENT_BIN:-signalfx-agent}

MY_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
selfdescribe_json="$MY_SCRIPT_DIR/../selfdescribe.json"
NUM_CORES=$(getconf _NPROCESSORS_ONLN)

doc_types=$(cat <<EOH
{
  "slice": "list",
  "uint16": "integer",
  "uint": "unsigned integer",
  "int": "integer",
  "struct": "object",
  "interface": "any"
}
EOH
)

generate_selfdescribe_json() {
  $AGENT_BIN selfdescribe > $selfdescribe_json
  echo "" >> $selfdescribe_json
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

# Run a query against the selfdescribe json with jq
j() {
  jq -r "$1" < $selfdescribe_json
}
