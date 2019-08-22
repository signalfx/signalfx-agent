#!/bin/bash -e

MONGOHOST=${MONGOHOST:-"localhost"}

if [ $# -eq 0 ]; then
    sed -i "s|mongodb://localhost|mongodb://$MONGOHOST|" logger/config.js
    npm start
else
    exec "$@"
fi
