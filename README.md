# TRISA Envoy

**An open source TRISA node for use in Travel Rule information transfers**

Complete documentation for Envoy can be found at [https://trisa.dev/envoy](https://trisa.dev/envoy).


## Running Envoy Locally

**NOTE**: Development is happening rapidly on the node right now; if these instructions don't work correctly, please open an issue on GitHub so we can review the docs.

Step 1: Generate localhost self-signed certificates:

```
$ ./.secret/generate.sh
```

This will use `openssl` to create a fake certificate authority (localhost.pem) and certificates for both the server and a client. You will need `openssl` installed on your computer for this command to work and you'll need the execute permission set on the `generate.sh` script. The secrets are stored in the `.secret` directory and should not be committed to GitHub.

Step 2: Build the latest version of the envoy node:

```
$ docker compose build
```

You will have to have `docker` installed on your computer with a compatible version of `docker compose`.

Step 3: Run the services (including Envoy and GDS):

```
$ docker compose up
```

Step 4: Create an admin user to login to the localhost with

```
$ docker compose exec envoy envoy createuser -e [email] -r admin
```

Now open a browser at [http://localhost:8000](http://localhost:8000) and you should be able to access the envoy node with the email and password created in the previous step!