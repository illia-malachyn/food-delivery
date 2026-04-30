#!/bin/sh
set -eu

CONNECT_URL="${KAFKA_CONNECT_URL:-http://kafka-connect:8083}"
CONNECTOR_NAME="${PAYMENT_CONNECTOR_NAME:-payment-outbox-connector}"

PAYMENT_DB_HOST="${PAYMENT_DB_HOST:-postgres}"
PAYMENT_DB_PORT="${PAYMENT_DB_PORT:-5432}"
PAYMENT_DB_NAME="${PAYMENT_DB_NAME:-payments}"
PAYMENT_DB_USER="${PAYMENT_DB_USER:-payments_user}"
PAYMENT_DB_PASSWORD="${PAYMENT_DB_PASSWORD:-payments_password}"

PAYMENT_EVENTS_TOPIC="${KAFKA_TOPIC_PAYMENT_EVENTS:-payment.events}"

echo "waiting for kafka connect at ${CONNECT_URL}"
for i in $(seq 1 90); do
  if curl -fsS "${CONNECT_URL}/connectors" >/dev/null 2>&1; then
    break
  fi
  if [ "$i" -eq 90 ]; then
    echo "kafka connect is not ready"
    exit 1
  fi
  sleep 2
done

cat > /tmp/payment-connector-config.json <<JSON
{
  "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
  "plugin.name": "pgoutput",
  "database.hostname": "${PAYMENT_DB_HOST}",
  "database.port": "${PAYMENT_DB_PORT}",
  "database.user": "${PAYMENT_DB_USER}",
  "database.password": "${PAYMENT_DB_PASSWORD}",
  "database.dbname": "${PAYMENT_DB_NAME}",
  "topic.prefix": "payment-cdc",
  "publication.name": "payment_outbox_publication",
  "publication.autocreate.mode": "filtered",
  "slot.name": "payment_outbox_slot",
  "slot.drop.on.stop": "false",
  "table.include.list": "public.payment_outbox",
  "heartbeat.interval.ms": "10000",
  "tombstones.on.delete": "false",
  "key.converter": "org.apache.kafka.connect.storage.StringConverter",
  "value.converter": "org.apache.kafka.connect.json.JsonConverter",
  "value.converter.schemas.enable": "false",
  "transforms": "outbox",
  "transforms.outbox.type": "io.debezium.transforms.outbox.EventRouter",
  "transforms.outbox.route.topic.replacement": "${PAYMENT_EVENTS_TOPIC}",
  "transforms.outbox.table.field.event.id": "id",
  "transforms.outbox.table.field.event.key": "aggregate_id",
  "transforms.outbox.table.field.event.payload": "payload",
  "transforms.outbox.table.fields.additional.placement": "event_name:header:event_name,event_version:header:event_version,aggregate_type:header:aggregate_type,aggregate_id:header:aggregate_id,occurred_at:header:occurred_at"
}
JSON

cat > /tmp/payment-connector.json <<JSON
{
  "name": "${CONNECTOR_NAME}",
  "config": $(cat /tmp/payment-connector-config.json)
}
JSON

status_code=$(curl -s -o /tmp/connect-response.txt -w "%{http_code}" "${CONNECT_URL}/connectors/${CONNECTOR_NAME}")
if [ "${status_code}" = "200" ]; then
  echo "updating existing connector ${CONNECTOR_NAME}"
  curl -fsS -X PUT "${CONNECT_URL}/connectors/${CONNECTOR_NAME}/config" \
    -H 'Content-Type: application/json' \
    --data-binary @/tmp/payment-connector-config.json
else
  echo "creating connector ${CONNECTOR_NAME}"
  curl -fsS -X POST "${CONNECT_URL}/connectors" \
    -H 'Content-Type: application/json' \
    --data-binary @/tmp/payment-connector.json
fi

echo "connector bootstrap completed"
