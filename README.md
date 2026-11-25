# TRISA Envoy

**An open source TRISA node for use in Travel Rule information transfers**

Complete documentation for Envoy can be found at [https://trisa.dev/envoy](https://trisa.dev/envoy).

## Travel Rule Compliance

If you're looking to deploy Envoy for travel rule compliance purposes, you'll need to start by requesting mTLS certificates to join the TRISA network. Please visit [The TRISA Directory](https://trisa.directory/) for more information about how to register.

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
$ docker compose exec envoy.local envoy users:create -e [email] -r admin
```

Now open a browser at [http://localhost:8000](http://localhost:8000) (or at [http://envoy.local:8000](http://envoy.local:8000) if you have edited your hosts file) and you should be able to access the envoy node with the email and password created in the previous step!

Step 6: Optionally create an admin user to login to the local counterparty with. The counterparty is intended to allow you to have two Envoy nodes to send transfers back and forth to.

```
$ docker compose exec counterparty.local envoy users:create -e [email] -r admin
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


## Configuring Envoy

Envoy is configured from the environment. Every setting is an environment variable with the `TRISA_` prefix (except for the optional `REGION_INFO_` variables), and most have a default so that only a few need to be set.

When it starts, Envoy also reads a `.env` file from the directory it is run in, if one exists. Variables that are already set in the environment take precedence over the values in the file. For local development, copy `.env.template` to `.env` and edit it.

### Printing the Configuration Reference

The `envoy config` command prints every environment variable that Envoy reads along with its type, default value, and description:

```
$ envoy config
This application is configured via the environment. The following environment
variables can be used:

KEY                      TYPE             DEFAULT                REQUIRED    DESCRIPTION
TRISA_MAINTENANCE        True or False    false                              if true, the node will start in maintenance mode
TRISA_ORGANIZATION       String           Envoy                              specify the name of the organization of the Envoy node for display purposes
...
```

The table is wide, so use the `--list` flag to print one block per variable instead:

```
$ envoy config --list
```

If you're running Envoy with docker compose or from source, use one of the following:

```
$ docker compose exec envoy.local envoy config
$ go run ./cmd/envoy config
```

This command prints the reference for the version of Envoy that you're running; it does not print the values that are currently set in your environment. A few notes on reading it:

- Some variables are listed more than once (e.g. `TRISA_MAINTENANCE` and `TRISA_ENDPOINT`) because several parts of Envoy read the same value; you only need to set them once.
- `false` in the `REQUIRED` column means that the variable is optional.
- The SMTP CRAM-MD5 setting is listed as `USE_CRAM_MD5`; set it as `TRISA_EMAIL_SMTP_USE_CRAM_MD5`.

### Minimum Configuration

Envoy will not start unless the following are set:

- `TRISA_NODE_CERTS`: the path to your TRISA mTLS certificates.
- `TRISA_TRP_ENDPOINT`: the endpoint of your TRP server, unless you set `TRISA_TRP_ENABLED=false`.

Outside of local development you will also want to set `TRISA_ENDPOINT`, `TRISA_WEB_ORIGIN`, the `TRISA_WEB_AUTH_` variables, and `TRISA_DATABASE_URL` rather than relying on their `localhost` defaults.

### Environment Variables

Values are parsed as follows:

- **bool**: `true` or `false`
- **duration**: a duration string with units such as `30s`, `15m`, or `1h30m`
- **map**: comma-separated `key:value` pairs, e.g. `key1:value1,key2:value2`

#### General

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `TRISA_MAINTENANCE` | bool | `false` | If true, the node starts in maintenance mode. Applies to the web UI, the TRISA node, and the TRP server. |
| `TRISA_ORGANIZATION` | string | `Envoy` | The name of your organization, for display purposes. |
| `TRISA_MODE` | string | `release` | The mode of the server: `release`, `debug`, or `test`. |
| `TRISA_LOG_LEVEL` | string | `info` | The verbosity of logging: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, or `panic`. |
| `TRISA_CONSOLE_LOG` | bool | `false` | If true, logs colorized human readable output instead of JSON. |
| `TRISA_DATABASE_URL` | string | `sqlite3:///trisa.db` | The DSN of the backend database. |
| `TRISA_SEARCH_THRESHOLD` | float | `0.0` | The threshold for fuzzy search, from 0.0 to 1.0. |
| `TRISA_ENDPOINT` | string | | The TRISA endpoint as assigned to the mTLS certificates of the node. Used by both the TRISA node and the web UI. |
| `TRISA_BOOT_REGENERATE_TRAVEL_ADDRESSES` | bool | `false` | If true, the travel addresses are regenerated when Envoy starts. |

#### Web UI and API

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `TRISA_WEB_ENABLED` | bool | `true` | If false, the web server is not run. |
| `TRISA_WEB_API_ENABLED` | bool | `true` | If false, the API returns unavailable when accessed. Has no effect if the web server is not enabled. |
| `TRISA_WEB_UI_ENABLED` | bool | `true` | If false, the UI returns unavailable when accessed. Has no effect if the web server is not enabled. |
| `TRISA_WEB_BIND_ADDR` | string | `:8000` | The IP address and port to bind the web server on. |
| `TRISA_WEB_ORIGIN` | string | `http://localhost:8000` | The origin (URL) of the web UI, used for CORS access and to create links, including the links in sunrise emails. Must be a URL with a scheme and host (e.g. `https://envoy.example.com`) if the web server is enabled. |
| `TRISA_WEB_DOCS_NAME` | string | `Envoy API Reference` | The display name of the API docs server in the Swagger app. |
| `TRISA_WEB_LOGO_URI` | string | | Use a custom logo for the web UI. |
| `TRISA_WEB_AUTH_KEYS` | map | | Optional static token signing keys as a map of key ID to the path of the key on disk. Keys can be generated with `envoy admin:tokenkey`. |
| `TRISA_WEB_AUTH_AUDIENCE` | string | `http://localhost:8000` | The value of the `aud` JWT claim. |
| `TRISA_WEB_AUTH_ISSUER` | string | `http://localhost:8000` | The value of the `iss` JWT claim. |
| `TRISA_WEB_AUTH_COOKIE_DOMAIN` | string | `localhost` | Limit cookies to the specified domain (exclude the port). |
| `TRISA_WEB_AUTH_ACCESS_TOKEN_TTL` | duration | `1h` | The amount of time before an access token expires. |
| `TRISA_WEB_AUTH_REFRESH_TOKEN_TTL` | duration | `2h` | The amount of time before a refresh token expires. |
| `TRISA_WEB_AUTH_TOKEN_OVERLAP` | duration | `-15m` | The amount of overlap between the access and refresh token. |

#### Webhook

The webhook is enabled by setting `TRISA_WEBHOOK_URL`; if it is not set, the other webhook variables are ignored.

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `TRISA_WEBHOOK_URL` | string | | The callback webhook that incoming travel rule messages are posted to. Must be a URL with a scheme and host, e.g. `https://example.com/callback`. |
| `TRISA_WEBHOOK_USE_MTLS` | bool | `false` | If true, the webhook client uses mTLS to secure its connection to the webhook server. |
| `TRISA_WEBHOOK_CERTS` | string | `TRISA_NODE_CERTS` | The path to the x509 certificate and private key for mTLS. |
| `TRISA_WEBHOOK_POOL` | string | `TRISA_NODE_POOL` | The path to the x509 cert pool for mTLS. |
| `TRISA_WEBHOOK_AUTH_KEY_ID` | string | | Identifies the shared secret key used for HMAC authorization. Required if the secret is set. |
| `TRISA_WEBHOOK_AUTH_KEY_SECRET` | string | | The hex encoded shared secret key used for HMAC authorization. A key and key ID can be generated with `envoy admin:hmackey`. |
| `TRISA_WEBHOOK_REQUIRE_SERVER_AUTH` | bool | `false` | If true, a valid HMAC authorization header is required in webhook responses. Requires the key ID and secret. |

#### TRISA Node and Directory

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `TRISA_NODE_ENABLED` | bool | `true` | If false, the TRISA node server is not run. |
| `TRISA_NODE_BIND_ADDR` | string | `:8100` | The IP address and port to bind the TRISA gRPC server on. |
| `TRISA_NODE_CERTS` | string | | **Required.** The path to the x509 certificate and private key for mTLS, with or without the certificate chain. |
| `TRISA_NODE_POOL` | string | | The path to the x509 cert pool for mTLS. If not set, the pool is created from the certificate chain in the certs. |
| `TRISA_NODE_KEY_EXCHANGE_CACHE_TTL` | duration | `24h` | The amount of time that keys exchanged with a counterparty are cached. |
| `TRISA_NODE_DIRECTORY_INSECURE` | bool | `false` | If true, do not connect to the directory using TLS. |
| `TRISA_NODE_DIRECTORY_ENDPOINT` | string | `api.trisa.directory:443` | The endpoint of the public GDS service; cannot be empty. Defaults to MainNet, use `api.testnet.directory:443` for TestNet. |
| `TRISA_NODE_DIRECTORY_MEMBERS_ENDPOINT` | string | `members.trisa.directory:443` | The endpoint of the members only GDS service; cannot be empty. Defaults to MainNet, use `members.testnet.directory:443` for TestNet. |
| `TRISA_DIRECTORY_SYNC_ENABLED` | bool | `true` | If false, the background service that synchronizes counterparties with the GDS is not run. |
| `TRISA_DIRECTORY_SYNC_INTERVAL` | duration | `6h` | The interval at which the synchronization is run. |

#### TRP

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `TRISA_TRP_ENABLED` | bool | `true` | If false, the TRP server is not run. |
| `TRISA_TRP_ENDPOINT` | string | | **Required if TRP is enabled.** The TRP endpoint as assigned to the mTLS certificates of the TRP server. Also used by the web UI. |
| `TRISA_TRP_BIND_ADDR` | string | `:8200` | The IP address and port to bind the TRP server on. |
| `TRISA_TRP_USE_MTLS` | bool | `true` | If true, the TRP server requires mTLS authentication. |
| `TRISA_TRP_CERTS` | string | `TRISA_NODE_CERTS` | The path to the x509 certificate and private key for mTLS. |
| `TRISA_TRP_POOL` | string | `TRISA_NODE_POOL` | The path to the x509 cert pool for mTLS. |
| `TRISA_TRP_IDENTITY_VASP_NAME` | string | | The name to respond with in a TRP identity request. |
| `TRISA_TRP_IDENTITY_LEI` | string | | The LEI of your VASP to respond with in a TRP identity request. |

The TRP and webhook certificates default to the TRISA node certificates: if `TRISA_TRP_CERTS`, `TRISA_TRP_POOL`, `TRISA_WEBHOOK_CERTS`, or `TRISA_WEBHOOK_POOL` is not set, the value of `TRISA_NODE_CERTS` or `TRISA_NODE_POOL` is used in its place. To use the default, leave the variable out of the environment entirely; a variable that is set to an empty value is not replaced.

#### Sunrise

Sunrise requires an email configuration to send messages, and creates the links in its emails from `TRISA_WEB_ORIGIN`.

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `TRISA_SUNRISE_ENABLED` | bool | `true` | If false, sunrise emails cannot be sent and sunrise endpoints return a 404. |
| `TRISA_SUNRISE_INVITE_ENDPOINT` | string | `/sunrise/verify` | The path of the endpoint for sunrise token verification. |
| `TRISA_SUNRISE_REQUIRE_OTP` | bool | `true` | If true, the user must enter a one-time code even when the verification token is valid. |

#### Daybreak

If `TRISA_DAYBREAK_ENABLED` is false, the other Daybreak variables are not validated.

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `TRISA_DAYBREAK_ENABLED` | bool | `false` | If true, Daybreak counterparties are synchronized and available in the web UI. |
| `TRISA_DAYBREAK_ENDPOINT` | string | `https://daybreaks.directory` | The endpoint of the Daybreak service. Must be a URL with a scheme and host. |
| `TRISA_DAYBREAK_CLIENT_ID` | string | | **Required if Daybreak is enabled.** The client ID of your license key. |
| `TRISA_DAYBREAK_CLIENT_SECRET` | string | | **Required if Daybreak is enabled.** The client secret of your license key. |
| `TRISA_DAYBREAK_SYNC_INTERVAL` | duration | `5m` | The interval at which Daybreak counterparties are synchronized. Must be greater than `1m`. |
| `TRISA_DAYBREAK_SYNC_JITTER` | duration | `1m` | The window around the interval used to randomize the timing of the synchronization. Must be greater than `1s`. |

#### Email

Emails are sent using either SMTP or SendGrid, but not both: set `TRISA_EMAIL_SMTP_HOST` to use SMTP or `TRISA_EMAIL_SENDGRID_API_KEY` to use SendGrid. If neither is set, Envoy cannot send emails.

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `TRISA_EMAIL_SENDER` | string | | The email address that messages are sent from, e.g. `Compliance <compliance@example.com>`. Required if SMTP or SendGrid is configured. |
| `TRISA_EMAIL_SENDER_NAME` | string | | The name of the sender, usually the name of the VASP or compliance team. |
| `TRISA_EMAIL_SUPPORT_EMAIL` | string | | The email address to refer support requests to. |
| `TRISA_EMAIL_COMPLIANCE_EMAIL` | string | | The email address to refer compliance requests to. |
| `TRISA_EMAIL_TESTING` | bool | `false` | Testing mode. The email configuration is not validated: the sender, support, and compliance addresses and the SMTP settings are not checked, and both SMTP and SendGrid may be configured. Emails are still sent if SMTP or SendGrid is configured. |
| `TRISA_EMAIL_SMTP_HOST` | string | | The SMTP host without the port, e.g. `smtp.example.com`. |
| `TRISA_EMAIL_SMTP_PORT` | integer | `587` | The port to access the SMTP server on. |
| `TRISA_EMAIL_SMTP_USERNAME` | string | | The username for authentication with the SMTP server. |
| `TRISA_EMAIL_SMTP_PASSWORD` | string | | The password for authentication with the SMTP server. |
| `TRISA_EMAIL_SMTP_USE_CRAM_MD5` | bool | `false` | Use CRAM-MD5 authentication (RFC 2195) instead of simple authentication. Requires the username and password. |
| `TRISA_EMAIL_SMTP_POOL_SIZE` | integer | `2` | The size of the SMTP connection pool used to send emails concurrently. |
| `TRISA_EMAIL_SENDGRID_API_KEY` | string | | The SendGrid API key. |

#### Region Info

Optional information about where the node is deployed. These variables do not have the `TRISA_` prefix.

The region must be one of the regions defined by the [`go.rtnl.ai/x/region`](https://pkg.go.dev/go.rtnl.ai/x/region) package; Envoy will not start if `REGION_INFO_ID` is set to a region that is not. If the region is set, then the name, country, cloud, and zone are derived from it unless they are overridden by their own variables.

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `REGION_INFO_ID` | region | | The region identifier, either its code (e.g. `2840302`) or its name (e.g. `gcp-us-east-4c`). |
| `REGION_INFO_NAME` | string | | The human readable name of the region. |
| `REGION_INFO_COUNTRY` | string | | The ISO 3166-1 country code of the region; must be a valid country code if the region is set. |
| `REGION_INFO_CLOUD` | string | | The cloud service provider. |
| `REGION_INFO_CLUSTER` | string | | The name of the cluster the node is hosted in. |
| `REGION_INFO_ZONE` | string | | The zone of the datacenter the node is hosted in. |

### Keeping this Section Up to Date

**For developers**: the configuration is defined by the struct tags in `pkg/config` (and in `pkg/emails/config.go` for email), which is also what `envoy config` prints. The tables above are written by hand, so they need to be updated in the same pull request as any change to the configuration. When you add, rename, or remove a configuration value:

1. Update the `testEnv` and `validConfig` fixtures in `pkg/config/config_test.go`. `TestEnvironmentSpec` fails until every variable is in the fixtures with a non-default value.
2. Update the tables above, either by hand or by giving the following prompt to your AI coding assistant.

```text
The "Configuring Envoy" section of README.md documents Envoy's environment variables
for the people who deploy it, so it has to match what the code actually reads. The
configuration has changed (or may have drifted); please bring that section back in
sync with the code.

The source of truth is the struct tags on Config in pkg/config, plus emails.Config
in pkg/emails/config.go which is nested under the TRISA_EMAIL prefix. Run
`go run ./cmd/envoy config --list` to print every variable with its type, default,
and description as generated from those tags, then compare it with the tables in
the README.

Some things to know when reading that output:

- A variable can be listed more than once (TRISA_MAINTENANCE, TRISA_ENDPOINT,
  TRISA_TRP_ENDPOINT, and TRISA_WEB_ORIGIN are) because several components read the
  same value. Document it once and say which components use it.
- The SMTP CRAM-MD5 flag is listed as USE_CRAM_MD5 because of how its env tag is
  written; the name to document is TRISA_EMAIL_SMTP_USE_CRAM_MD5.
- Descriptions come from the desc tags, which can be missing or wrong. Anything the
  README states about behavior (allowed values, what is required, what falls back
  to what) should be checked against the Validate methods and config.New rather
  than taken from the tag. If a tag is wrong, fix the tag as well so that
  `envoy config` and the README agree.

When you're finished, every variable should appear exactly once, in the table for
its group, with the correct type, default, and description; variables that were
removed or renamed should be gone; and the "Minimum Configuration" list, the notes
on reading the `envoy config` output, and the note on certificate defaults should
still be true. The minimalEnv fixture in pkg/config/config_test.go is the tested
minimum. Keep the existing layout and tone, and don't edit the rest of the README.

Variable names are also written by hand in .env.template, docker-compose.yaml, and
pkg/emails/.env.template, and nothing checks those files. Don't change them, but do
tell me about any names in them that the code no longer reads.

Finally, run `go test ./pkg/config/...` and report: which variables you added,
removed, or changed in the README; any desc tags you corrected; the stale names
you found in the other files; and anything where the code and its description
disagree and you could not tell which one is right.
```
