#/usr/bin/env bash

set -x

curl http://admin:admin@localhost:8080/v2/service_instances/abcdef123456/service_bindings/some-binding-guid-123 -d '{
  "service_id": "simple-id",
  "plan_id": "simple-plan",
  "context": {
    "platform": "cloudfoundry",
    "some_field": "some-contextual-data"
  },
  "app_guid": "app-guid-here",
  "organization_guid": "org-guid-here",
  "space_guid": "space-guid-here",
  "bind_resource": {
    "app_guid": "app-guid-here"
  }
}' -X PUT -H "X-Broker-API-Version: 2.13" -H "Content-Type: application/json"
