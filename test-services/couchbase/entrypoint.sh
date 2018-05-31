#! /bin/bash
# Based on a guide by Laura Czajkowski located at:
# https://dzone.com/articles/using-docker-to-develop-with-couchbase
set -m

# initialize variables
HOSTNAME=`hostname -f`
export MEMORY_QUOTA=${MEMORY_QUOTA:-256}
export INDEX_MEMORY_QUOTA=${INDEX_MEMORY_QUOTA:-256}
export FTS_MEMORY_QUOTA=${FTS_MEMORY_QUOTA:-256}
export SERVICES=${SERVICES:-"kv,n1ql,index,fts"}
export USERNAME=${USERNAME:-"administrator"}
export PASSWORD=${PASSWORD:-"password"}

# Check if couchbase server is up
check_db() {
 curl --silent http://${HOSTNAME}:8091/pools > /dev/null
 echo $?
}

# start couchbase server in the background
/entrypoint.sh couchbase-server &

# Wait until it's ready
until [[ $(check_db) = 0 ]]; do
>&2 echo "Waiting for Couchbase Server to be available"
sleep 1
done
echo "# Couchbase Server Online"

echo "# Starting setup process"
echo "Initialize the node"
curl --silent "http://${HOSTNAME}:8091/nodes/self/controller/settings" \
-d path="/opt/couchbase/var/lib/couchbase/data" \
-d index_path="/opt/couchbase/var/lib/couchbase/data"

echo "# Setting hostname"
curl --silent "http://${HOSTNAME}:8091/node/controller/rename" \
-d hostname=${HOSTNAME}

echo "# Setting up memory"
curl --silent "http://${HOSTNAME}:8091/pools/default" \
-d memoryQuota=${MEMORY_QUOTA} \
-d indexMemoryQuota=${INDEX_MEMORY_QUOTA} \
-d ftsMemoryQuota=${FTS_MEMORY_QUOTA}

echo "# Setting up services"
curl --silent "http://${HOSTNAME}:8091/node/controller/setupServices" \
-d services="${SERVICES}"

echo "# Setting up credentials"
curl --silent "http://${HOSTNAME}:8091/settings/web" \
-d port=8091 \
-d username=${USERNAME} \
-d password=${PASSWORD} > /dev/null

echo "# Re-attaching to original entrypoint"
# Attach to couchbase entrypoint
fg 1

echo "# exited entrypoint.sh"
