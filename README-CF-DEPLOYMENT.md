
# The Entire Steps for BOSH Lite + Cloud Foundry(`cf-deployment`) + Broker

## Prerequisites

BOSH CLI:

```
$ bosh -v
version 2.0.48-e94aeeb-2018-01-09T23:08:08Z
```

VirtualBox:

```
$ VBoxManage --version
5.2.6r120293
```

## Deploy BOSH Lite

```
$ git clone https://github.com/cloudfoundry/bosh-deployment

$ bosh create-env bosh-deployment/bosh.yml \
  --state ./state.json \
  -o bosh-deployment/virtualbox/cpi.yml \
  -o bosh-deployment/virtualbox/outbound-network.yml \
  -o bosh-deployment/bosh-lite.yml \
  -o bosh-deployment/bosh-lite-runc.yml \
  -o bosh-deployment/jumpbox-user.yml \
  -o bosh-deployment/uaa.yml \
  --vars-store ./creds.yml \
  -v director_name="Bosh Lite Director" \
  -v internal_ip=192.168.50.6 \
  -v internal_gw=192.168.50.1 \
  -v internal_cidr=192.168.50.0/24 \
  -v outbound_network_name=NatNetwork

$ bosh alias-env lite -e 192.168.50.6 --ca-cert <(bosh int ./creds.yml --path /director_ssl/ca)
$ export BOSH_CLIENT=admin && export BOSH_CLIENT_SECRET=`bosh int ./creds.yml --path /admin_password`

$ sudo route add -net 10.244.0.0/16     192.168.50.6 # for Mac OS X
```

## Deploy Cloud Foundry By `cf-deployment`

```
$ git clone https://github.com/cloudfoundry/cf-deployment

$ export STEMCELL_VERSION=$(bosh int cf-deployment/cf-deployment.yml --path /stemcells/alias=default/version)
$ bosh -e lite upload-stemcell https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent?v=$STEMCELL_VERSION

$ bosh -e lite update-cloud-config cf-deployment/iaas-support/bosh-lite/cloud-config.yml

$ bosh -e lite -d cf deploy cf-deployment/cf-deployment.yml \
  -o cf-deployment/operations/bosh-lite.yml \
  -o cf-deployment/operations/experimental/use-bosh-dns.yml \
  -o cf-deployment/operations/experimental/use-bosh-dns-for-containers.yml \
  -o cf-deployment/operations/experimental/secure-service-credentials-mine.yml \
  --vars-store cf-deployment-vars.yml \
  -v system_domain=bosh-lite.com
```

> Note:
> - Enable BOSH DNS by `use-bosh-dns.yml` and `use-bosh-dns-for-containers.yml`
> - Enable CredHub by `secure-service-credentials-mine.yml`

Now the BOSH Lite + Cloud Foundry env is ready!


## Create UUA Client For CredHub

```
$ uaac target https://uaa.bosh-lite.com

$ bosh int cf-deployment-vars.yml --path=/uaa_admin_client_secret
<UUA ADMIN CLIENT SECRET SHOWS HERE>

$ uaac token client get admin
Client secret: <UUA ADMIN CLIENT SECRET>

$ uaac client add secure-credentials-broker -i
    New client secret:  my-secret
    Verify new client secret:  my-secret
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
    lastmodified: 1520429360000
    id: secure-credentials-broker
```

## Build, Deploy and Try the Broker

1. Clone the `secure-credentials-broker`

```
$ git clone https://github.com/brightzheng100/secure-credentials-broker.git
```

2. Log into Cloud Foundry

```
$ cf login -a https://api.bosh-lite.com --skip-ssl-validation -u admin -p $(bosh interpolate cf-deployment-vars.yml --path /cf_admin_password)
```

3. Create Org/Space

```
$ cf create-org dev && cf create-space -o dev dev
$ cf target -o dev -s dev
```

4. Setup ASG

```
$ cd secure-credentials-broker
$ cat <<EOF > asg.json
[
    {
        "protocol": "tcp",
        "destination": "10.244.0.0/16",
        "ports": "8844,8443"
    }
]
EOF

$ cf create-security-group secure-service-credentials asg.json
$ cf bind-staging-security-group secure-service-credentials
$ cf bind-running-security-group secure-service-credentials
``` 

> Note: there are two ports must be set in `asg.json`: `8844` is for CredHub and `8443` is for UAA


5. Build, push, create broker

```
$ mkdir bin
$ GOOS=linux go build -o ./bin/secure-credentials-broker
$ cf push

$ cf create-service-broker secure-credentials-broker user password https://secure-credentials-broker.bosh-lite.com
$ cf enable-service-access secure-credentials -p default -o dev
```


6. Create Service

```
$ cf create-service secure-credentials default secure-credential-service-1 -c '{"user":"top secret"}'
$ cf services
Getting services in org dev / space dev as admin...
OK

name                          service              plan      bound apps   last operation
secure-credential-service-1   secure-credentials   default                create succeeded
```

7. Bind Service to App

Assuming there is an app named `hello-world`.

```
$ cf bind-service hello-world secure-credential-service-1
$ cf restage hello-world
```