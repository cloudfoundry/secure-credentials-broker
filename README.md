**NB:** This service broker is provided as a proof-of-concept and as a practical example for integrating service brokers with CredHub. It is not actively maintained or intended for deployment or use in production environments.

# Secure Application Credentials Broker

This broker is prototype application service broker that lets user-provided credentials to be securely stored in CredHub for applications to use. 
The service broker stores the user-provided configuration parameters in CredHub, and returns a CredHub reference back to the platform.

For the entire process about how to setup a local environment by using [BOSH Lite](https://bosh.io/docs/bosh-lite) and [Cloud Foundry `cf-deployment`](https://github.com/cloudfoundry/cf-deployment/), please refer to [here](README-CF-DEPLOYMENT.md).

# Using the sample broker
## Creating a UAA client with credhub permissions

* The broker is currently configured to use a UAA client for authentication. You must first login with uaa admin credentials to create a UAA client that has credhub read and write access.

```
$ uaac target https://<your-uaa-domain>

$ uaac token client get admin
Client secret: <admin-password>

$ uaac client add secure-credentials-broker -i
New client secret: my-secret
Verify new secret: my-secret
scope (list):
authorized grant types (list):  client_credentials
authorities (list):  credhub.read,credhub.write
access token validity (seconds):  3600
refresh token validity (seconds):
redirect uri (list):
autoapprove (list):
signup redirect url (url):
  scope: uaa.none
  client_id: secure-credentials-broker
  resource_ids: none
  authorized_grant_types: client_credentials
  autoapprove:
  access_token_validity: 3600
  authorities: credhub.write credhub.read
  name: secure-credentials-broker
  signup_redirect_url:
  required_user_groups:
  lastmodified: 1519917340000
  id: secure-credentials-broker
```

## Configuring the broker

* `git clone` this repo and modify the manifest file to have the client and client secret you created using uaac.

For example:
```
CREDHUB_SERVER: https://credhub.service.cf.internal:8844
CREDHUB_CLIENT: secure-credentials-broker
CREDHUB_SECRET: my-secret
SKIP_TLS_VALIDATION: true
BROKER_AUTH_USERNAME: user
BROKER_AUTH_PASSWORD: password
```

* Target and login to CF, creating appropriate orgs and spaces.

```
$ cf api <your-cf-api-url-goes-here>
$ cf login
$ cf create-org myOrg
$ cf create-space mySpace -o myOrg
$ cf target -o myOrg -s mySpace
```

* Change directories to the secure-credentials-broker directory. Create a application security group (asg) json file that has the following contents:

```
[
  {
    "protocol": "tcp",
    "destination": "10.0.0.0/16",
    "ports": "8844,8443"
  }
]
```

> Note: Please refer to [here](asg.json) for the example and the `destination` is subject to your ERT/PAS network CIDR

* Push the service broker application, and then register it to CF as a broker. 
Note, currently the broker credentials, and the service name, and plan are hardcoded in the broker code.

```
$ cf push

$ cf create-security-group secure-service-credentials asg.json
$ cf bind-staging-security-group secure-service-credentials
$ cf bind-running-security-group secure-service-credentials

$ cf create-service-broker secure-credentials-broker admin admin https://<your-service-broker-app-url-goes-here>
$ cf enable-service-access secure-credentials -p default -o myOrg
```

* Create a service instance of your broker and bind to the application that is meant to talk to the broker
```
$ cf create-service secure-credentials default myInstance -c '{"myJsonKey":"myJsonValue"}'
$ cf push <your-app-that-talks-to-broker> 
$ cf bind-service myApp myInstance
$ cf restage myApp 
```

* Assuming that you are running credhub in assisted-mode your application should be able to access the JSON used when creating the service-instance.  

## Updating the json data

If you would like to update the data that the application has access to, you can do the following:

```
$ cf update-service myInstance -c '{"updatedKey":"updatedValue"}'
$ cf restage myApp
```

