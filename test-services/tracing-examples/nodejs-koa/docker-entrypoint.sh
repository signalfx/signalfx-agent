#!/bin/bash -e

MONGOHOST=${MONGOHOST:-"localhost"}

if [ $# -eq 0 ]; then
    sed -i "s|mongodb://localhost|mongodb://$MONGOHOST|" wordExplorer/mongoDriver.js
    npm start
else
    exec "$@"
fi
