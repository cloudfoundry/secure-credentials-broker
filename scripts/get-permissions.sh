curl -k "$CREDHUB_SERVER/api/v1/permissions?credential_name=$1" \
  -X GET \
  -H "authorization: $(credhub --token)" \
  -H 'content-type: application/json'
