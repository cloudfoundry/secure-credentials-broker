#/usr/bin/env bash

set -x

curl -k -v "http://admin:admin@localhost:8080/v2/service_instances/abcdef123456/service_bindings/some-binding-guid-123?service_id=simple-id&plan_id=simple-plan" -X DELETE -H "X-Broker-API-Version: 2.13" -H "Content-Type: application/json"
