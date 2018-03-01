
#Secure Application Credentials Broker

This broker is prototype application service broker that lets user-provided credentials to be securely stored in CredHub for applications to use. 
The service broker stores the user-provided configuration parameters in CredHub, and returns a CredHub reference back to the platform.


#Using the sample broker
## Creating a UAA client with credhub permissions

The broker is currently configured to use a UAA client for authentication. You must first login with uaa admin credentials to create a UAA client that has credhub read and write access.

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

```
CREDHUB_CLIENT: secure-credentials-broker
CREDHUB_SECRET: my-secret

```

* Target and login to CF, creating appropriate orgs and spaces.

```
$ cf api <your-cf-api-url-goes-here>
$ cf login
API endpoint: https://api.cf.security.cf-app.com
Email> admin
Password> <password>
$ cf create-org myOrg
$ cf target -o myOrg
$ cf create-space mySpace
$ cf target -s mySpace
```

* Change directories to the secure-credentials-broker directory. Create a application security group (asg) json file that has the following contents:

```
[
  {
    "protocol": "tcp",
    "destination": "10.0.0.0/16",
    "ports": "8844"
  }
]

```

* Push the service broker application, and then register it to CF as a broker. 
Note, currently the broker credentials, and the service name, and plan are hardcoded in the broker code.


```
$ cf push
$ cf create-security-group secure-service-credentials <path-to-asg-json-file    >
$ cf create-service-broker secure-credentials-broker admin admin https://<your-service-broker-app-url-goes-here>
$ cf enable-service-access secure-credentials -p default -o myOrg
```

* Create a service instance of your broker and bind to the application that is meant to talk to the broker
```
$ cf create-service myInstance default myInstance -c '{"myJsonKey":"myJsonValue"}'
$ cf push <your-app-that-talks-to broker> 
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

