# Foodtruck

Foodtruck is a service to run ad-hoc jobs on nodes. It provides an HTTP API for administrators
to request jobs be run on a set of nodes, along with querying the status of those jobs. Nodes,
using the provided foodtruck client binary, poll the HTTP API for jobs on a specified interval.

## Overview

### Jobs / Tasks
Administrator can request ad-hoc jobs run against nodes. A job contains a list of nodes, and a task
to run on those nodes. Below is an example job specification:
```json
{
    "task": {
        "window_start": "2020-12-25T21:06:45+00:00",
        "window_end": "2021-12-27T21:07:00+00:00",
        "provider": "infra",
        "spec": {
            "url": "https://example.com/policy.tar.gz"
        }
    },
    "nodes": [
        {
            "org": "org",
            "name": "node1"
        },
        {
            "org": "org",
            "name": "node2"
        },
    ]
}
```

This job runs a task with provider type `infra` on `org/node1` and `org/node2`. The task specifies the
window in which it is allowed to run. It also specifies a `spec` field, which is where any information
needed by the provider to execute the task is placed.

### Client / Providers
The client that runs on each node polls the server on some interval for a task to run on the node. If
a task is available to run, the server will send it to the client. The client inspects the `provider`
field of the task. The client will search for the executable `foodtruck-provider-$provider` and execute
it, providing the `spec` field of the task as `stdin`. For example, in the previous example, we had a task:

```json
{
    "window_start": "2020-12-25T21:06:45+00:00",
    "window_end": "2021-12-27T21:07:00+00:00",
    "provider": "infra",
    "spec": {
        "url": "https://example.com/policy.tar.gz"
    }
}
```

A simple provider that downloads the policy archive, unpacks it, and runs chef-client could look like this:

```bash
#!/bin/env bash

set -xe

tmpdir="$(mktemp -d)"
cd "$tmpdir"
curl -o policy.tar.gz "$(jq -r .url)"
tar -xvzf policy.tar.gz
cd out
chef-client -z --chef-license accept
exit 0
```

This example uses jq to parse the json provided on stdin, and get the url from it. It uses curl to download
the file, unpacks it with tar, and runs chef-client in local mode.

## Usage

### Server

The server requires Azure Cosmos DB to run. Cosmos DB must be configured configured to use the MongoDB 3.6 compatible
interface. The server should be run on a modern Linux system.

#### Building

To build the server, make sure to have the latest version of go install. As of writing this, go 1.15.6 was used.

Run
```bash
make server
```

This will build the server and output a binary `bin/foodtruck-server`

#### Running

The server requires the following environment variables be set before running:

- `MONGODB_CONNECTION_STRING` : The MongoDB connection string provided by Cosmos DB. You can find more information
  about it in the [Cosmos DB Docs](https://docs.microsoft.com/en-us/azure/cosmos-db/connect-mongodb-account).
- `MONGODB_DATABASE_NAME` : The name of the Cosmos DB to use. This database must be created before starting the server
  and correctly tuned to handle the load.
- `NODES_API_KEY` : A randomly generated API key used to access the node endpoints. These endpoints will not have the
  ability to create Jobs, only pull tasks for the nodes. If foodtruck is fronted by Chef Infra Server, Chef Infra Server
  would use this API key to talk to Foodtruck. Below is a way to generate the API key using OpenSSL:

  ```bash
  ➜ openssl rand -hex 20
  1ffd0e1090f0842e0cd26008621bad3902db4bb9
  ```
- `ADMIN_API_KEY` : A randomly generated API key used for admin endpoints like creating jobs, getting job status. Below is
  a way to generate the API key using OpenSSL:

  ```bash
  ➜ openssl rand -hex 20
  cfc69ed63341dd2403ed547a3d68babb61d5f248
  ```

Optionally, the following environment variables can also be set:
- `FOODTRUCK_LISTEN_ADDR` : Specifies the interface and port to listen on. By default, foodtruck will listen on "0.0.0.0:1323".

With the environment variables exported, you can run the server with:

```bash
./bin/foodtruck-server
```

#### Proxying through Chef Server
The Foodtruck client supports using Chef Server based Authentication. To allow Foodtruck to make use of this, you must proxy
the endpoints the foodtruck client calls through the Chef Server. This can be done by creating the following files nginx config
files (Note: you may have to tweak this config for your Chef Server setup):

foodtruck_external.conf:
```
location ~ "^/organizations/([^/]+)/foodtruck/nodes/([^/]+)/tasks/(status|next)$" {
    set $request_org $1;
    set $request_client $2;
    access_by_lua_block {
      local headers = ngx.req.get_headers()
      if ngx.var.request_client ~= headers.x_ops_userid then
        ngx.exit(ngx.HTTP_FORBIDDEN)
      end
      validator.validate("POST")
    }
    proxy_set_header authorization "Bearer THE_NODE_API_KEY";

    proxy_pass http://foodtruck;
}
```

foodtruck_upstreams.conf:
```
upstream foodtruck {
  server 127.0.0.1:1323;
}
```

These files must be places in `node['private_chef']['nginx']['dir']/etc/addon.d/`.

#### Example Requests

Create a job:

```bash
➜ curl --location --request POST 'http://localhost:1323/admin/jobs' \
--header "Authorization: Bearer $ADMIN_API_KEY" \
--header 'Content-Type: application/json' \
--data-raw '{
    "task": {
        "window_start": "2020-12-25T21:06:45+00:00",
        "window_end": "2021-12-27T21:07:00+00:00",
        "provider": "infra",
        "spec": {
            "url": "https://example.com/policy.tar.gz"
        }
    },
    "nodes": [
        {
            "org": "org",
            "name": "node"
        }
    ]
}'

{
    "id": "5ff7686a91072739255a4a35"
}
```

Get the job status:

```bash
➜  curl --location --request GET 'http://localhost:1323/admin/jobs/5ff7686a91072739255a4a35?fetchStatuses=true' \
--header "Authorization: Bearer $ADMIN_API_KEY"

{
    "job": {
        "id": "5ff7686a91072739255a4a35",
        "task": {
            "window_start": "2020-12-25T21:06:45Z",
            "window_end": "2021-12-27T21:07:00Z",
            "provider": "infra",
            "spec": {
                "url": ""https://example.com/policy.tar.gz""
            }
        },
        "nodes": [
            {
                "org": "neworg",
                "name": "testnode"
            }
        ]
    },
    "statuses": [
        {
            "job_id": "5ff7686a91072739255a4a35",
            "node_name": "neworg/testnode",
            "status": "failed",
            "last_updated": "0001-01-01T00:00:00Z",
            "result": {
                "exit_code": 1,
                "reason": "exit error"
            }
        }
    ]
}
```

### Client

#### Building

The client can be built for the following os-arch pairs:
- linux-amd64
- windows-amd64
- darwin-amd64
- solaris-amd64
- aix-ppc

To build all clients for all platforms:
```
make client-all
```

#### Running

Running the client requires a JSON config file. For example:

```
{
	"base_url": "http://host.docker.internal:1323",
	"auth": {
		"type": "apiKey",
		"key": "1ffd0e1090f0842e0cd26008621bad3902db4bb9"
	},
	"node": {
		"org": "neworg",
		"name": "testnode"
	},
	"interval": "5s"
}
```

If requests for foodtruck are being proxied through a Chef Server, use the
following config as an example:

```
{
	"base_url": "https://a2-dev.test",
	"auth": {
		"type": "chefServer",
		"key_path": "/etc/chef/client.pem"
	},
	"node": {
		"org": "testorg",
		"name": "testnode"
	},
	"interval": "5s"
}
```

- `base_url`: The url used to talk to foodtruck
- `auth.type`: One of `chefServer` or `apiKey`
- `auth.key`: This is the `NODE_API_KEY` that was set on the server. This can also be specified through the 
  `NODE_API_KEY` environment variable. This is only valid for the `apiKey` type.
- `auth.key_path`: The path the the chef server client key for the node. This is only valid for the `chefServer` type.
- `node`: The name of the node along with the organization
- `interval`: How often to check for jobs. For example `"5s"`, `"5m"`, `"5h"`.

To run:
```
./bin/foodtruck-client-$OS-$ARCH
```

Make certain the providers are in the path.

## Running the Tests
There is a suite of integration tests available to run. You can run these tests against CosmosDB with the MongoDB API,
against a running instance of MongoDB, or the tests can spin up a MongoDB instance for you.

To have the tests run against a MongoDB spun up automatically with docker, run:
```
go test ./test -docker-mongo
```

If you do not want the container to automatically be cleaned up, you can pass an additional parameter `-docker-cleanup=false`:
```
go test ./test -docker-mongo -docker-cleanup=false
```

To have the tests run against a CosmosDB instance, export the `MONGODB_CONNECTION_STRING` and `MONGODB_DATABASE_NAME`
environment variables and run:
```
go test ./test -is-cosmos
```

Similarly, you can run tests against an already running MongoDB instance by first exporting the `MONGODB_CONNECTION_STRING` 
and `MONGODB_DATABASE_NAME` environment variables and running:
```
go test ./test
```
