if [[ -z "$START_AS" ]]; then
    echo "ERROR: missing mandatory config: START_AS"
    exit 1
fi

case "$START_AS" in
 "broker" )
   echo "setting up broker"
   bash scripts/setup-broker.sh
   ;;
 "consumer" )
   echo "setting up consumer"
   bash scripts/setup-consumer.sh
   ;;
 "producer" )
   echo "setting up producer"
   bash scripts/setup-producer.sh
   ;;
 "create-topic" )
   echo "creating topic"
   bash scripts/setup-topic.sh
   ;;
  * )
   echo -n "Valid options include broker, consumer, producer, create-topic"
   ;;
esac
