#!/bin/bash
#############################################################################
# Script to set/append files based on environment variables for agent       #
#############################################################################
#
# To create file using an environment variable
# add the following environment variables
#
#   KEY:   SET_FILE
#   VALUE: path to file
#
#   KEY:   SET_FILE_CONTENT
#   VALUE: content for file
#
#   Note: additional files can be included by appending number to the end
#         of additional keys (number must start with 1 and increment by 1).
#         e.g. SET_FILE1, SET_FILE_CONTENT1
#
# To append to file using an environment variable
# add the following environment variables
#
#   KEY:   APPEND_FILE
#   VALUE: path to file
#
#   KEY:   APPEND_FILE_CONTENT
#   VALUE: content for file
#
#   Note: additional files can be included by appending number to the end
#         of additional keys (number must start with 1 and increment by 1).
#         e.g. APPEND_FILE1, APPEND_FILE_CONTENT1
#
#############################################################################
set -e

# set file function (includes arg in names)
set_content_to_file() {
  envvar="SET_FILE$1"
  if [ -n "${!envvar}" ]; then
    file="${!envvar}"
    content_envvar="SET_FILE_CONTENT$1"
    echo "${!content_envvar}" > $file
  fi
}

# append function (includes arg in names)
append_content_to_file() {
  envvar="APPEND_FILE$1"
  if [ -n "${!envvar}" ]; then
    file="${!envvar}"
    content_envvar="APPEND_FILE_CONTENT$1"
    echo "${!content_envvar}" >> $file
  fi
}

# set file using default environment variable
# (no number appended)
set_content_to_file

# set additional environment variables to files
# (they have numbers appended to envvar key)
for counter in {1..100};
do
  envvar="SET_FILE${counter}"
  if [ -n "${!envvar}" ]; then
    set_content_to_file "$counter"
  else
    break
  fi
done

# append default environment variables to file
# (no number appended)
append_content_to_file

# append additional environment variables to files
# (they have numbers appended to envvar key)
for counter in {1..100};
do
  envvar="APPEND_FILE${counter}"
  if [ -n "${!envvar}" ]; then
    append_content_to_file "$counter"
  else
    break
  fi
done

