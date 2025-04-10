# TRISA Envoy

**An open source TRISA node for use in Travel Rule information transfers**

> **WARNING**: This branch now contains our v1.0.0 release candidate! However v1.0.0 is not currently backwards compatible with v0.30.1. Please do not run from develop while we QA the release candidate!

Complete documentation for Envoy can be found at [https://trisa.dev/envoy](https://trisa.dev/envoy).

## Running Envoy Locally

**NOTE**: Development is happening rapidly on the node right now; if these instructions don't work correctly, please open an issue on GitHub so we can review the docs.

Step 1: Generate localhost self-signed certificates (ensure that you run the script from within the `.secret` directory):

```
$ cd .secret
$ ./generate.sh
$ cd..
```

This will use `openssl` to create a fake certificate authority (localhost.pem) and certificates for a development server, a development counterparty, and a client. You will need `openssl` installed on your computer for this command to work and you'll need the execute permission set on the `generate.sh` script. The secrets are stored in the `.secret` directory and should not be committed to GitHub.

Prior to step 2 you can optionally set the `$GIT_REVISION` environment variable to prevent warnings and help you track what version is running in docker compose (but this will only work if you've cloned the repository).

```
$ export GIT_REVISION=$(git rev-parse --short HEAD)
```

Step 2: Build the latest version of the envoy node:

```
$ docker compose build
```

You will have to have `docker` installed on your computer with a compatible version of `docker compose`.

Step 3: Run the services (including Envoy and GDS):

```
$ docker compose up
```

**Localhost Networking**: to ensure that the services can reach other internally to the docker compose network environment, the nodes are launched as `envoy.local` and `counterparty.local` respectively. It is likely that if you're developing, you'll have to add these domains to your `/etc/hosts` file as follows:

```
##
# Host Database
#
# localhost is used to configure the loopback interface
# when the system is booting. Do not change this entry.
##
127.0.0.1       localhost
255.255.255.255 broadcasthost
::1             localhost

127.0.0.1 envoy.local
127.0.0.1 counterparty.local
```

Step 4: Initialize the GDS with data about the localhost network.

```
$ go run ./cmd/fsi gds:init
```

Note this requires you to have [Go](https://go.dev/doc/install) installed on your computer. If you cannot install Go, let us know by creating an issue and we can build a binary for your OS that you can download and run.

Step 5: Create an admin user to login to the localhost with

```
$ docker compose exec envoy.local envoy createuser -e [email] -r admin
```

Now open a browser at [http://localhost:8000](http://localhost:8000) (or at [http://envoy.local:8000](http://envoy.local:8000) if you have edited your hosts file) and you should be able to access the envoy node with the email and password created in the previous step!

Step 6: Optionally create an admin user to login to the local counterparty with. The counterparty is intended to allow you to have two Envoy nodes to send transfers back and forth to.

```
$ docker compose exec counterparty.local envoy createuser -e [email] -r admin
```

You can access the counterparty at [http://localhost:9000](http://localhost:9000) or at [http://counterparty.local:9000](http://counterparty.local:9000) if you have edited your hosts file.

> **NOTE**: Due to the way cookie domains work with the credentials, you can only be logged into either the envoy development node or the counterparty development node at the same time. It's annoying, but you'll have to login again when switching between nodes unfortunately.


## Envoy Implementation Options

| Open Source                                                                                                         | One-Time Setup                                                                                                                                    | Managed Service                                                                                                                                                 |
| ------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Envoy is open source (MIT License). Download, install, integrate, host and support your own Envoy node and service. | The Envoy team will install and configure your Envoy node in your environment while you host, maintain, and support the node on an ongoing basis. | The Envoy team will install, configure, host, maintain, and support an Envoy node for you. Includes dedicated, provenance-aware node with regional deployments. |
|                                                                                                                     |                                                                                                                                                   |                                                                                                                                                                 |

If you’d like more information on the one-time integration service or managed services, [schedule a demo](https://rtnl.link/p2WzzmXDuSu) with the Envoy team!


## Envoy Support

|                        | Open Source            | One-Time Setup         | Managed Service |
| ------------------------------- | ---------------------- | ---------------------- | ---------------------- |
| [Envoy Documentation](https://trisa.dev/envoy/index.html)                                            | 	&#10003;                                    | 	&#10003;                                    | 	&#10003;                                    |
| Access to [TRISA Slack Community](https://trisa-workspace.slack.com/)                                | 	&#10003;                                    | 	&#10003;                                    | 	&#10003;                                    |
| Training from Envoy Team                                       |                                              | 	&#10003;                                    | 	&#10003;                                    |
| Dedicated Support                                              |                                              |                                              | 	&#10003;                                    |
| Response Time*                                                 | Within 5 business days                       | Within 5 business days                       | Within 3 business days                       |



*The Envoy team's business hours are 9AM - 6PM Eastern.