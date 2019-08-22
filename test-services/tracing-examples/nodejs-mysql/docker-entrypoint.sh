#!/bin/bash -e

MYSQLHOST=${MYSQLHOST:-"localhost"}

if [ $# -eq 0 ]; then
    sed -i "s|localhost|$MYSQLHOST|" deedScheduler/dbDriver/dbConfig.js
    npm start
else
    exec "$@"
fi
