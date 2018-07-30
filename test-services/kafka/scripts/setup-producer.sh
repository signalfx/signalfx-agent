if [[ -z "$KAFKA_BROKER" ]]; then
    echo "ERROR: missing mandatory config: KAFKA_BROKER"
    exit 1
fi
sleep 25
while true; do echo "Hello World"; sleep $((1 + RANDOM % 10)); done | ./opt/kafka_2.11-"$KAFKA_VERSION"/bin/kafka-console-producer.sh --broker-list "$KAFKA_BROKER" --topic sfx-employee
