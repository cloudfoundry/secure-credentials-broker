In order for the service broker to authenticate with CredHub, it needs to resolve the `uaa.service.cf.interal` DNS address that CredHub returns as its oauth server. If your deployment does not contain an existing BOSH DNS alias applicable to the service broker's container, these tools can help you create one:

1. Upload the `bosh-dns-aliases-ext.tgz` release to your bosh director.
1. Update the `uaa-dns-runtime-config.yml` file with the appropriate deployment and network names.
1. Upload the `uaa-dns-runtime-config.yml` to your bosh director.
1. Redeploy.
