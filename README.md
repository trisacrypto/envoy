# TRISA Envoy

**An open source TRISA node for use in Travel Rule information transfers**

Complete documentation for Envoy can be found at [https://trisa.dev/envoy](https://trisa.dev/envoy).


## Running Envoy Locally

**NOTE**: Development is happening rapidly on the node right now; if these instructions don't work correctly, please open an issue on GitHub so we can review the docs.

Step 1: Generate localhost self-signed certificates:

```
$ ./.secret/generate.sh
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

Step 4: Initialize the GDS with data about the localhost network.

```
$ go run ./cmd/fsi init-gds
```

Note this requires you to have [Go](https://go.dev/doc/install) installed on your computer. If you cannot install Go, let us know by creating an issue and we can build a binary for your OS that you can download and run.

Step 5: Create an admin user to login to the localhost with

```
$ docker compose exec envoy envoy createuser -e [email] -r admin
```

Now open a browser at [http://localhost:8000](http://localhost:8000) and you should be able to access the envoy node with the email and password created in the previous step!

Step 6: Optionally create an admin user to login to the local counterparty with. The counterparty is intended to allow you to have two Envoy nodes to send transfers back and forth to.

```
$ docker compose exec counterparty envoy createuser -e [email] -r admin
```

You can access the counterparty at [http://localhost:9000](http://localhost:9000).

> **NOTE**: Due to the way cookie domains work with the credentials, you can only be logged into either the envoy development node or the counterparty development node at the same time. It's annoying, but you'll have to login again when switching between nodes unfortunately.