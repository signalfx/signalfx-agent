#!/bin/bash

set -eo pipefail

DIRS=$@

if [[ $EVENT_NAME = "pull_request" ]]
then
    changed_files="$(git diff --name-only --diff-filter=ACMRT $BASE_REF $HEAD_REF | xargs)"
elif [[ $EVENT_NAME = "push" ]]
then
    changed_files="$(git diff --name-only --diff-filter=ACMRT $HEAD_REF $HEAD_REF^ | xargs)"
fi

for DIR in $DIRS; do
    if [ $(echo $changed_files | grep -c "$DIR") -gt 0 ]; then
        echo "::set-output name=files_changed::true"
        exit 0
    fi
done

echo "::set-output name=files_changed::false"
