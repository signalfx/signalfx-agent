#! /bin/bash

set -ex

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# read in environment variable for version and map to boxcutter version
WIN_NAME="eval-win2008r2-standard-ssh"
if [ "$WIN_VER" == "server_2008" ]; then 
    WIN_NAME="eval-win2008r2-standard-ssh"
fi
if [ "$WIN_VER" == "server_2012" ]; then 
    WIN_NAME="eval-win2012r2-standard-ssh"
fi
if [ "$WIN_VER" == "server_2016" ]; then
    WIN_NAME="eval-win2016-standard-ssh"
fi

# remove existing vagrant box if it exists
if [ -n "$(vagrant box list | grep $WIN_NAME)" ]; then
    vagrant box remove --force $WIN_NAME
fi

# make directory for boxcutter projects
mkdir -p "$SCRIPT_DIR/boxcutter"

# clone boxcutter windows project if it doesn't exist
# see https://github.com/boxcutter/windows for more info
if [ ! -d "$SCRIPT_DIR/boxcutter/windows" ]; then
    git clone git@github.com:boxcutter/windows.git $SCRIPT_DIR/boxcutter/windows
fi

cd "$SCRIPT_DIR/boxcutter/windows"

# pull latest changes to boxcutter windows
git pull

# build the windows box
make virtualbox/$WIN_NAME

# add box to vagrant
vagrant box add -f --name $WIN_NAME $SCRIPT_DIR/boxcutter/windows/box/virtualbox/$WIN_NAME*.box