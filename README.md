# README

EcoLinker is a web application to help you retrieve information from EcoFlow devices over HTTP and MQTT using EcoFlow's
IoT Developer Platform. EcoLinker exposes them as Prometheus metrics or forwards them to other MQTT brokers. This allows
you to further integrate with other tools like Home Assistant or NodeRED - you will get the raw data! You can also
define collectors which run periodically in the background for any metric which is not transmitted through EcoFlow's
MQTT.

Keep in mind that this has been developed for private use and could only be tested for one specific device. It should be
generic in such a way to easily contribute for required changes. These are very welcome, please
see [Development & contribution](#development-and-contribution).

The main git repository is hosted at
_[https://git.myservermanager.com/varakh/ecolinker](https://git.myservermanager.com/varakh/ecolinker)_. Other
repositories are mirrors and pull requests, issues, and planning are managed there.

## Features

With EcoLinker, you can easily get device information through HTTP. There are also more friendly clients to this,
see [usage](#usage).

* Retrieve your available devices
* Retrieve your device's parameters
* Retrieve your device's battery information
* (**PowerOcean only**) Retrieve historical data

If enabled, EcoLinker also offers live data retrieval from EcoFlow's MQTT broker. This data is exposed
as [Prometheus](#prometheus) metrics.

Furthermore, EcoLinker can act as a forwarder for all messages from EcoFlow's MQTT broker to another MQTT broker.

By default, EcoLinker's functionality is not secured. It can be secured by enabling basic authentication (even
multiple credentials are possible).

## Usage

If you have not set up EcoLinker yet, see [prerequisite](#prerequisite) and
then [installation & deployment](#installation-and-deployment).

If you have deployed EcoLinker properly and you want EcoLinker to expose Prometheus metrics and subscribe to necessary
MQTT topics from EcoFlow, you need to

1. add your EcoFlow device to EcoLinker as tracked device and
2. then create a MQTT subscription in EcoLinker.

**You only need to do it once!** Tracked devices and subscriptions are persisted.

EcoLinker's main interface to control it is a **[command-line interface client](#command-line-interface-client)** (as of
now).

### Command-line interface client

The command-line interface client is bundled in the same binary as the main server application. Download it for your
operating system and then execute to see available commands. For GNU/Linux, this is as simple as `./ecolinker` to see
the help. Make sure that you give it executable rights on your machine first.

The necessary steps outlined above (adding your EcoFlow device, then subscribing) can all be done with three single
commands, from your machine or directly inside the docker container or on the host running EcoLinker.

#### Finding the URL of your EcoLinker instance

When you've deployed your EcoLinker instance to a home server or NAS, make sure to expose its port. When you've deployed
it under a domain with a reverse proxy, then this is your URL for further usage.

> The command-line interface supports proper [configuration](#client-configuration) instead of setting environment
> variables or providing necessary configuration in each call.

#### Getting your EcoFlow devices

You need to get your device's serial number, so you can add this as tracked device in EcoLinker.

```shell
$ ./ecolinker ecoflow ls

Serial Number    | Online
7QX3V2A9KLM8J5WC | 1
```

You can see, our EcoFlow account listed _one_ device with serial number `7QX3V2A9KLM8J5WC` which is currently online.

Write the serial number down for the next step.

#### Adding your EcoFlow device as a tracked device

Let's add a new tracked device as follows:

```shell
$ ./ecolinker devices add --sn 7QX3V2A9KLM8J5WC --label "my test" --device-kind other

SN      7QX3V2A9KLM8J5WC
Kind    other
Label   my test
Created 2025-06-07 11:51:22.541351047 +0200 CEST
Updated 2025-06-07 11:51:22.541351047 +0200 CEST
```

#### Subscribing to EcoFlow MQTT

We're now ready to start subscribing to EcoFlow's MQTT topics for this tracked device. This means that EcoLinker will
start to publish Prometheus metrics afterward.

Let's add a subscription (sub) with the tracked device serial number.

```shell
$ ./ecolinker subs add --sn 7QX3V2A9KLM8J5WC --topic-kind quota

ID      0b226b46-ce36-483d-a602-fcdaf18fd94b
Device SN       7QX3V2A9KLM8J5WC
Topic Kind      quota
Created 2025-06-07 23:55:34.94871 +0200 CEST
Updated 2025-06-07 23:55:34.94871 +0200 CEST
```

Finished! EcoLinker now starts to produce Prometheus metrics (see [Prometheus](#prometheus)) which you can use to build
Grafana dashboards for your EcoFlow device.

You can also use EcoLinker as [MQTT forward](#mqtt-forwarding) mechanism, meaning that all incoming MQTT messages from
EcoFlow's MQTT broker
are forwarded to a MQTT broker of your liking, maybe for Home Assistant or NodeRED. Please see
the [configuration](#configuration) `MQTT_FORWARD_` keys on how to enable it.

#### Using collectors

EcoFlow's MQTT might not be enough to get all metrics from your tracked device, some are just not distributed over MQTT
and the MQTT topics `/get` and `/get_reply` are not available in the EcoFlow's Open IoT platform (as of now).

For such scenarios, EcoLinker provides the concept of _collectors_. EcoLinker can query EcoFlow's device parameters
endpoint to periodically retrieve its values and expose them as Prometheus metrics and/or forward them
via [MQTT forward](#mqtt-forwarding).

> Be aware that collectors do regular calls to EcoFlow's API. As of now, EcoFlow did not reply to questions about rate
> limits.

> EcoFlow's device parameters do have an internal update interval. Fetching them too frequently might not yield any
> updated values.

> Consider to always favor MQTT over this. Don't add collectors for parameters which are retrieved over EcoFlow's MQTT
> mechanism already.

To get a list of devices parameters, use `./ecolinker ecoflow ps --sn 7QX3V2A9KLM8J5WC`. You can use any printed key to
be collected.

In the following example, a new collector is created which runs every two minutes for 5 parameters. The `--parameter`
flag is optional. If parameters are omitted, all device's parameters are processed. It's advised to only query specific
ones which are missing from EcoFlow's MQTT payload.

```shell
$ ./ecolinker collectors add device-parameters --sn 7QX3V2A9KLM8J5WC --fq 2m --pp bpSoc --pp mmpptPwr --p sysGridPwr --pp sysLoadPwr --pp bpPwr
ID      677cdb08-f20d-49dc-beac-54cd32d86735
Device SN       7QX3V2A9KLM8J5WC
Kind    device_parameters
Frequency       2m0s
Payload map[parameters:[bpSoc mmpptPwr sysLoadPwr bpPwr] sn:7QX3V2A9KLM8J5WC]
Created 2025-06-14 08:45:22.737204 +0200 CEST
Updated 2025-06-14 08:45:22.737204 +0200 CEST
```

There are more collectors, look into the help for the `collectors add` command with:

```shell
$ ./ecolinker collectors add -h
``` 

#### Client configuration

The command-line interface client supports either configuration via environment variables or via a `.toml` configuration
file. It's a convenient way to avoid providing all arguments in each call.

Example: in each command-line call, you can manually provide `--url http://192.168.0.42:8080` (or similar). This can get
quite tedious to do, thus you can also export an environment variable `ECOLINKER_URL`, then you don't need to provide
the EcoLinker instance URL in each command-line call anymore.

```shell
export ECOLINKER_URL=http://192.168.0.42:8080
```

The same applies when you've secured your EcoLinker instance, e.g., with basic authentication. Usually, you would need
to provide `--user` and `--pass` in each call. You can save the manual writing with these exports:

```shell
# Values come from EcoLinker itself and how you set it up
export ECOLINKER_USER=myuser
export ECOLINKER_PASSWORD=mypassword
```

Besides the environment variable configuration, EcoLinker's command-line client supports configuration via TOML. You can
change the location of the configuration file in each call with `--config` or setting its location with
`ECOLINKER_CONFIG`. If no explicit configuration location is provided, EcoLinker searches a config file in **the default
configuration** directory, e.g., on GNU/Linux in `$HOME/.config/ecolinker.toml`.

Here's an example of such a `ecolinker.toml` file. Remember, EcoLinker's command-line client won't create one by
default!

```toml
# Defines the URL to EcoLinker
[server]
url = "http://192.168.0.42:8080"

# If you've set up EcoLinker to be secured with authentication, make sure to also set it here.
[auth]
user = "ecolinkeruser"
password = "mySecretPassword"
# Takes priority over 'password'
# Make sure the fail contains no line breaks
passwordFile = "/home/myuser/.config/ecolinker.secret"

# Always print JSON
[parsing]
raw = true

# Set your device serial number
#[device]
#serialNumber = "XXXX"
```

#### Other hints

The command-line interface also supports more actions, feel free to look into its help function with
`./ecolinker --help`.

Example for retrieving device specific parameters via `--pp` (`--parameter`) from EcoFlow's API:

```shell
./ecolinker ecoflow ps --sn HJ36ZDHAZH2A0119 --pp bpSoc --pp bpPwr --pp mpptPwr --pp sysGridPwr --pp sysLoadPwr

Attribute  | Value
bpPwr      | -485
bpSoc      | 59
mpptPwr    | 0
sysGridPwr | -6
sysLoadPwr | 479
```

Avoiding all specific parameters queries _all_ parameter values from EcoFlow's API.

EcoLinker supports more actions and [device specifics](./_doc/README.md) functionality, like editing a tracked device's
name etc. This is (as of now) not exposed in the command-line tool, but you can do it with `curl`.

The command-line interface client supports autocompletion. There's a hidden command which can be used to generate the
necessary autocompletion code for shells like `bash`, `zsh`, or alike with e.g. `./ecolinker completion zsh`. Afterward,
you need to place this into your shell's autocompletion source directory.

## Prerequisite

For EcoLinker you **MUST** have an EcoFlow IoT Developer Platform account!

1. Go to https://developer-eu.ecoflow.com/.
2. Click on _"Become a Developer"_.
3. Login with your EcoFlow username and password.
4. Wait until the access is approved by EcoFlow.
5. Receive email with subject _"Approval notice from EcoFlow Developer Platform"_. This may take some time.
6. Go to https://developer-eu.ecoflow.com/us/security and create new _Access Key_ and _Secret Key_ which can be used as
   EcoLinker's configuration values with `ECOFLOW_ACCESS_KEY` and `ECOFLOW_SECRET_KEY`.

EcoLinker also **requires** a Postgres database for its internals.
See [installation & deployment](#installation-and-deployment) next.

## Installation and deployment

EcoLinker is distributed as docker image or native binaries for all common operating systems.

Depending on **how you like to reach EcoLinker** (reverse proxy setup with a (sub)domain or reverse proxy setup on sub
path of your existing domain), pick one of the below **deployment** options.

The following sections outline how to deploy EcoLinker in a containerized environment and also natively.

Most important is that EcoLinker needs proper EcoFlow credentials via `ECOFLOW_ACCESS_KEY` and `ECOFLOW_SECRET_KEY`.
See [prerequisite](#prerequisite) if you don't have a EcoFlow IoT Developer account yet.

### Container

The following outlines how to deploy using `docker-compose`. If you prefer using plain `docker` or `podman` commands,
make sure to create necessary network (for podman use the _pod_ concept). Please refer to online resources if you're not
familiar how to translate the docker-compose examples to plain container engine commands.

By default, the following examples only make EcoLinker listen on `localhost`/`127.0.0.1` which can be used with
a [reverse proxy](#reverse-proxy) (**recommended**). For testing, you can also remove the local part in the port mapping
directives and expose EcoLinker directly.

#### docker-compose: Deployment on a (sub)domain

```yaml
networks:
  internal:
    external: false
    driver: bridge
    driver_opts:
      com.docker.network.bridge.name: br-ecolinker

services:
  app:
    container_name: ecolinker_app
    image: git.myservermanager.com/varakh/ecolinker:latest
    environment:
      - TZ=Europe/Berlin
      - DB_POSTGRES_TZ=Europe/Berlin
      - DB_POSTGRES_HOST=db
      - DB_POSTGRES_PORT=5432
      - DB_POSTGRES_NAME=ecolinker
      - DB_POSTGRES_USER=ecolinker
      - DB_POSTGRES_PASSWORD=$SECURE_RANDOM_DATABASE_PASSWORD
    restart: unless-stopped
    networks:
      - internal
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      - db

  db:
    container_name: ecolinker_db
    image: docker.io/postgres:17
    restart: unless-stopped
    environment:
      - POSTGRES_USER=ecolinker
      - POSTGRES_PASSWORD=$SECURE_RANDOM_DATABASE_PASSWORD
      - POSTGRES_DB=ecolinker
    networks:
      - internal
    volumes:
      - ecolinker-db-vol:/var/lib/postgresql/data

volumes:
  ecolinker-db-vol:
    external: false
```

#### docker-compose: Deployment on a sub path

Use the [deployment on a (sub)domain](#docker-compose-deployment-on-a-subdomain) as starting point and adapt your
`docker-compose.yaml` file accordingly. Let's assume you like to deploy under the `/ecolinker-app` base path, then add
`SERVER_BASE_PATH=/ecolinker-app/`.

Next, look into the fitting [reverse proxy setup](#reverse-proxy) or decide if you
need [high availability](#high-availability).

### High availability

For high availability, add [REDIS](https://redis.io/) to support proper distributed locking.

Make changes to your docker-compose deployment similar to the following:

```yaml
# ... = other defined directives
services:
  app:
    # ...
    environment:
      - LOCK_REDIS_ENABLED=true
      - LOCK_REDIS_HOST=redis
      - LOCK_REDIS_PORT=6379
      # ...

  redis:
    container_name: ecolinker_redis
    image: redis
    restart: unless-stopped
    networks:
      - internal
    volumes:
      - redis-data-vol:/var/redis/data
    # optionally expose port depending on your setup
    ports:
      - "127.0.0.1:6379:6379"

volumes:
  redis-data-vol:
    external: false
  # ...
```

You need a proper load balancer which routes incoming traffic to all of your instances.

### Reverse proxy

The following examples use `nginx` as reverse proxy and Let's Encrypt for transport encryption (https).

### (Sub)Domain

Most likely, this is the default setup and used for the majority of deployments. _ecolinker_ is deployed as a single
container (excluding database) or [natively](#native-deployment).

We assume your deployment works, and you like to make it available behind `https://ecolinker.domain.tld`.

```shell
server {
    listen 443 ssl http2;
    ssl_certificate /etc/letsencrypt/live/ecolinker.domain.tld/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/ecolinker.domain.tld/privkey.pem;
    
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Sub path

We assume your deployment works, and you like to make it available behind `https://domain.tld/ecolinker-app`.

This requires to set `SERVER_BASE_PATH=/ecolinker-app/` as outlined in the deployment section above.

```shell
server {
    # ... your other domain setup

    # forward matching requests to the main ecolinker application
    # make sure that SERVER_BASE_PATH is the same as the path inside the location (except for trailing slash)
    location /ecolinker-meta {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Native deployment

Deploying _ecolinker_ natively is also possible.

First, download the binary for your operating system, make it executable, e.g., with `chmod +x ecolinker`, then
place it into the directory you want, e.g., `/usr/local/bin`. Afterward, run the binary with `./ecolinker server serve`.

For a native deployment, it's recommended to use a service orchestrator like systemd on UNIX/Linux machines. Here's an
example file `ecolinker.service` which you can put into `/etc/systemd/system` or alike, then reload available systemd
services with `systemctl daemon-reload` to make it available.

Make sure that your `/etc/ecolinker.conf` has all necessary environment variables, e.g. `DB_POSTGRES_*` and alike set to
configure the database connection.

Afterward, start and enable it with `systemctl enable --now ecolinker.service`.

```shell
[Unit]
Description=ecolinker
After=network.target

[Service]
Type=simple
# Using a dynamic user drops privileges and sets some security defaults
# See https://www.freedesktop.org/software/systemd/man/latest/systemd.exec.html
DynamicUser=yes
# All environment variables for ecolinker can be put into this file
# ecolinker picks them up (on each restart)
EnvironmentFile=/etc/ecolinker.conf
# Requires ecolinker binary to be installed at this location, e.g., via package manager or copying it over manually
ExecStart=/usr/local/bin/ecolinker server serve
```

For a full set of available configuration, look into the [Configuration](#configuration) section.

### Nix

Add EcoLinker as **Nix flakes** input:

```nix
# flake.nix
{
    inputs.ecolinker.url = "git+https://git.myservermanager.com/varakh/ecolinker?ref=refs/tags/latest";
}
```

There's a NixOS module which you can use. For available properties, see `nix/module.nix`. Here's a minimal example:

```nix
services.ecolinker = {
  enable = true;
  environment = {
    # ...
  };
  environmentFiles = [
    # ..., e.g., config.sops.upda-env.path
  ];
};
```

There's a Home Manger module which you can use. For available properties, see `nix/hm-module.nix`. Here's a minimal example:

```nix
programs.ecolinker = {
  enable = true;
  settings = {
    server.url = "http://192.168.1.2:8181";
    auth = {
      user = "administrator";
      passwordFile = config.sops.secrets.ecolinker-password.path;
    };
    device.serialNumber = "xxx";
    parsing.raw = false;
  };
};
```

### Prometheus

Metrics are exposed with [prometheus](https://prometheus.io), so that you can easily build a
dashboard in [Grafana](https://grafana.com), or even attach alerts
via [alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/).

Prometheus exporter is enabled by default. See [configuration](#configuration) how to fine-tune its behavior.

A Prometheus scrape configuration might look like the following if `PROMETHEUS_SECURE_TOKEN_ENABLED` is set to `true`.

```shell
scrape_configs:
  - job_name: 'ecolinker'
    static_configs:
      - targets: ['<ip address of ecolinker>:8080']
    bearer_token: 'VALUE_OF_PROMETHEUS_SECURE_TOKEN'
```

The prometheus exporter can also be spawned independently of the main application server by setting `PROMETHEUS_PORT` to
a different port than `SERVER_PORT`.

#### EcoFlow specific metrics

When EcoLinker connects to EcoFlow's MQTT and subscribes to the topics you added your device to in EcoLinker, the
EcoFlow's MQTT message payload is transformed into Prometheus metrics and exposed by EcoLinker. Lists are transformed to
`index=` labels. The payload is **highly depending on YOUR EcoFlow device and YOUR environment**, though metrics are
always exposed with label `device` and prefixed with `ecolinker_`.

Here is an example for a PowerOcean device, having data about the battery's state of charge `bpSoc`, data for the PV
`mpptPv_`, and the three phases system `pcsXPhase`.

```shell
ecolinker_bpSoc{device="<YOUR DEVICE SERIAL NUMBER>"} 55
ecolinker_emsBpAliveNum{device="<YOUR DEVICE SERIAL NUMBER>"} 2
ecolinker_mpptHeartBeat_mpptPv_amp{device="<YOUR DEVICE SERIAL NUMBER>",index0="0",index1="0"} 0.09888114
ecolinker_mpptHeartBeat_mpptPv_amp{device="<YOUR DEVICE SERIAL NUMBER>",index0="0",index1="1"} 0
ecolinker_mpptHeartBeat_mpptPv_pwr{device="<YOUR DEVICE SERIAL NUMBER>",index0="0",index1="0"} 2.8028738
ecolinker_mpptHeartBeat_mpptPv_pwr{device="<YOUR DEVICE SERIAL NUMBER>",index0="1",index1="0"} 0
ecolinker_mpptHeartBeat_mpptPv_vol{device="<YOUR DEVICE SERIAL NUMBER>",index0="0",index1="0"} 28.34589
ecolinker_mpptHeartBeat_mpptPv_vol{device="<YOUR DEVICE SERIAL NUMBER>",index0="0",index1="1"} 24.111694
ecolinker_pcsAPhase_actPwr{device="<YOUR DEVICE SERIAL NUMBER>"} -139.80458
ecolinker_pcsAPhase_amp{device="<YOUR DEVICE SERIAL NUMBER>"} 0.88920784
ecolinker_pcsAPhase_apparentPwr{device="<YOUR DEVICE SERIAL NUMBER>"} 207.52975
ecolinker_pcsAPhase_reactPwr{device="<YOUR DEVICE SERIAL NUMBER>"} 153.373
ecolinker_pcsAPhase_vol{device="<YOUR DEVICE SERIAL NUMBER>"} 233.38724
ecolinker_pcsBPhase_actPwr{device="<YOUR DEVICE SERIAL NUMBER>"} -146.44104
ecolinker_pcsBPhase_amp{device="<YOUR DEVICE SERIAL NUMBER>"} 0.89949435
ecolinker_pcsBPhase_apparentPwr{device="<YOUR DEVICE SERIAL NUMBER>"} 210.12268
ecolinker_pcsBPhase_reactPwr{device="<YOUR DEVICE SERIAL NUMBER>"} 150.68697
ecolinker_pcsBPhase_vol{device="<YOUR DEVICE SERIAL NUMBER>"} 233.60089
ecolinker_pcsCPhase_actPwr{device="<YOUR DEVICE SERIAL NUMBER>"} -141.35924
ecolinker_pcsCPhase_amp{device="<YOUR DEVICE SERIAL NUMBER>"} 0.86465895
ecolinker_pcsCPhase_apparentPwr{device="<YOUR DEVICE SERIAL NUMBER>"} 202.7179
ecolinker_pcsCPhase_reactPwr{device="<YOUR DEVICE SERIAL NUMBER>"} 145.30008
ecolinker_pcsCPhase_vol{device="<YOUR DEVICE SERIAL NUMBER>"} 234.44838
```

To start building a Grafana dashboard for your setup, look into the `_doc/` folder for an example or set up EcoLinker
and your Prometheus instance and then use Prometheus web interface to discover metrics, building it iteratively.

The meaning of attributes is not well-documented by EcoFlow,
but https://github.com/foxthefox/ioBroker.ecoflow-mqtt/tree/main/doc/devices might help you to get started. AI is also
quite good at giving you explanations if you paste the metrics prefixed with `ecolinker_` in.

#### Application specific metrics

EcoLinker exposes custom application specific metrics, for its **own resource usage** and also for its **EcoFlow
connection and MQTT forwarding service**.

The following metrics are tied to EcoFlow and EcoLinker's MQTT or collector functionality.

```shell
# HELP ecolinker_ecoflow_mqtt_connected EcoFlow MQTT is connected, 0 indicating not connected, 1 indicating that it's connected
# TYPE ecolinker_ecoflow_mqtt_connected gauge
ecolinker_ecoflow_mqtt_connected 1
# HELP ecolinker_ecoflow_mqtt_enabled EcoFlow MQTT is enabled, 0 indicating not enabled, 1 indicating that it's enabled
# TYPE ecolinker_ecoflow_mqtt_enabled gauge
ecolinker_ecoflow_mqtt_enabled 1
# HELP ecolinker_mqtt_forward_connected MQTT forward is connected, 0 indicating not connected, 1 indicating that it's connected
# TYPE ecolinker_mqtt_forward_connected gauge
ecolinker_mqtt_forward_connected 1
# HELP ecolinker_mqtt_forward_enabled MQTT forward is enabled, 0 indicating not enabled, 1 indicating that it's enabled
# TYPE ecolinker_mqtt_forward_enabled gauge
ecolinker_mqtt_forward_enabled 1
# HELP ecolinker_ecoflow_mqtt_message_last_received Last received message timestamp (UNIX epoch in seconds) from EcoFlow MQTT
# TYPE ecolinker_ecoflow_mqtt_message_last_received gauge
ecolinker_ecoflow_mqtt_message_last_received{device="7QX3V2A9KLM8J5WC",topicKind="quota"} 1.749372062e+09
# HELP ecolinker_ecoflow_mqtt_messages_received Messages received from EcoFlow MQTT
# TYPE ecolinker_ecoflow_mqtt_messages_received counter
ecolinker_ecoflow_mqtt_messages_received{device="7QX3V2A9KLM8J5WC",topicKind="quota"} 12
# HELP ecolinker_collector_invocations Invocations of a collector
# TYPE ecolinker_collector_invocations counter
ecolinker_collector_invocations{device="7QX3V2A9KLM8J5WC",id="57528e68-b11f-4b49-aebb-7fa958ca4fd5",kind="device_parameters"} 1
# HELP ecolinker_collector_last_invocation Last invocation timestamp (UNIX epoch in seconds) for a collector
# TYPE ecolinker_collector_last_invocation gauge
ecolinker_collector_last_invocation{device="7QX3V2A9KLM8J5WC",id="57528e68-b11f-4b49-aebb-7fa958ca4fd5",kind="device_parameters"} 1.749901826e+09
```

EcoLinker exposes some performance metrics for its own runtime.

#### Dashboards with Grafana

Building a Grafana dashboard to visualize your device's metrics is probably why you started using EcoLinker in the first
place. As this is heavily device and environment specific, there are two examples in the `_doc/` folder to get started.

Remember, available metrics depend on defined collectors, your device, and MQTT payload sent from EcoFlow itself.

* Example for a PowerOcean (inverter) device, look into the [PowerOcean](./_doc/powerocean/README.md).
  It uses MQTT and collectors and incorporates the performance metrics. There are also
  some [screenshots](./_doc/powerocean/README.md).
* Example for a basic Grafana dashboard visualizing EcoLinker's performance metrics only, look into
  the [go_ginprom](./_doc/go_ginprom.json) dashboard.

#### Alerts with Alertmanager

The [application metrics](#application-specific-metrics) can help you build proper alerting
with [Alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/). Especially the timestamp based metrics
have been introduced to serve this need.

Here's an example [Alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/) configuration snippet to get
started:

```yaml
# rules
groups:
  - name: ecolinker
    rules:
      # apply when you have EcoFlow MQTT enabled, checks if last received message from EcoFlow is at most 1 hour ago
      - alert: EcoLinkerEcoFlowMQTTMessageMissed
        expr: ((time()-ecolinker_ecoflow_mqtt_message_last_received)/60/60) >= 1
        for: 1h
        labels:
          severity: critical
          class: smarthome
        annotations:
          summary: "EcoFlow MQTT's last message is too long ago"
          description: "EcoFlow MQTT's last message is too long ago"

      # checks if last invocation of any collector is at least 2 hours ago
      - alert: EcoLinkerCollectorMissed
        expr: ((time()-ecolinker_collector_last_invocation)/60/60) >= 2
        for: 1h
        labels:
          severity: critical
          class: smarthome
        annotations:
          summary: "EcoLinker's collector last invocation is too long ago {{ $labels.id }}"
          description: "EcoLinker's collector last invocation is too long ago.\nID: {{ $labels.id }}"

      # apply when you have EcoFlow MQTT enabled, checks if not disconnected
      - alert: EcoLinkerEcoFlowMQTTDisconnected
        expr: ecolinker_ecoflow_mqtt_connected == 0
        for: 30m
        labels:
          severity: critical
          class: smarthome
        annotations:
          summary: "EcoFlow MQTT connection lost"
          description: "EcoFlow MQTT connection lost"

      # apply when you have MQTT forwarding enabled, checks if not disconnected
      - alert: EcoLinkerForwardMQTTDisconnected
        expr: ecolinker_mqtt_forward_connected == 0
        for: 30m
        labels:
          severity: critical
          class: smarthome
        annotations:
          summary: "Forward MQTT connection lost"
          description: "Forward MQTT connection lost"
```

### MQTT Forwarding

When MQTT Forwarding has been enabled via [configuration](#configuration), EcoLinker forwards all messages to that
configured MQTT broker.

The receiving broker needs to be configured during
EcoLinker's [installation and deployment](#installation-and-deployment).

The following topics are used as forward topics:

- `/ecolinker/<Device Serial Number>/quota` for quota messages from EcoFlow's MQTT broker
- `/ecolinker/<Device Serial Number>/status` for status messages from EcoFlow's MQTT broker
- `/ecolinker/<Device Serial Number>/device_parameters` for device parameters message produced by a collector
- `/ecolinker/<Device Serial Number>/device_historical_data` for device historical message produced by a collector

## Configuration

The following table describe most important configuration values.

| Variable                          | Purpose                                                                                                                                                                                                               |
|:----------------------------------|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ECOFLOW_URL`                     | EcoFlow endpoint for communication. Defaults to `https://api-e.ecoflow.com`.                                                                                                                                          |
| `ECOFLOW_ACCESS_KEY`              | EcoFlow developer account access key. Not set by default, you need to explicitly set it.                                                                                                                              |
| `ECOFLOW_SECRET_KEY`              | EcoFlow developer account secret key. Not set by default, you need to explicitly set it.                                                                                                                              |
| `ECOFLOW_MQTT_ENABLED`            | If EcoLinker connects to EcoFlow's MQTT. Defaults to `true`.                                                                                                                                                          |
| `ECOFLOW_MQTT_DEBUG_MESSAGES`     | Increases verbosity for retrieved messages from EcoFlow's MQTT. Defaults to `false`.                                                                                                                                  |
| `MQTT_FORWARD_ENABLED`            | If MQTT forwarding is enabled, meaning that all incoming MQTT messages from EcoFlow's MQTT instance are forwarded (as is, same topic, same payload) to another MQTT broker. Defaults to `false`.                      |
| `MQTT_FORWARD_PROTOCOL`           | If MQTT forwarding is enabled, the connection information. Defaults to `tcp`, can be any of `tcp`, `ssl`, `ws`, or `mqtts`.                                                                                           |
| `MQTT_FORWARD_HOST`               | If MQTT forwarding is enabled, the connection information. Not set by default.                                                                                                                                        |
| `MQTT_FORWARD_PORT`               | If MQTT forwarding is enabled, the connection information. Not set by default.                                                                                                                                        |
| `MQTT_FORWARD_USERNAME`           | If MQTT forwarding is enabled, the connection information. Not set by default.                                                                                                                                        |
| `MQTT_FORWARD_PASSWORD`           | If MQTT forwarding is enabled, the connection information. Not set by default.                                                                                                                                        |
| `TZ`                              | The time zone (**recommended** to set it properly, background tasks depend on it). Defaults to `Etc/UTC`, can be any time zone according to _tz database_.                                                            |
| `AUTH_MODE`                       | The auth mode to secure EcoLinker. Possible values are `none`, `basic_single` and `basic_credentials`. Defaults to `none`.                                                                                            |
| `BASIC_AUTH_USER`                 | For auth mode `basic_single`: Username for login. The user name                                                                                                                                                       |
| `BASIC_AUTH_PASSWORD`             | For auth mode `basic_single`: User's password for login. The user's password                                                                                                                                          |
| `BASIC_AUTH_CREDENTIALS`          | For auth mode `basic_credentials`: list of semicolon separated credentials, e.g. `username1\|password1;username2\|password2`. Not set by default, you need to explicitly set it.                                      |
| `DB_POSTGRES_HOST`                | The postgres host. Postgres host address, defaults to `localhost`                                                                                                                                                     |
| `DB_POSTGRES_NAME`                | The postgres database name. Postgres database name                                                                                                                                                                    |
| `DB_POSTGRES_PASSWORD`            | The postgres password. Postgres user password                                                                                                                                                                         |
| `DB_POSTGRES_PORT`                | The postgres port. Postgres port, defaults to `5432`                                                                                                                                                                  |
| `DB_POSTGRES_TZ`                  | The postgres time zone. Postgres time zone settings, defaults to `Etc/UTC`                                                                                                                                            |
| `DB_POSTGRES_USER`                | The postgres user. Postgres user name                                                                                                                                                                                 |
| `PROMETHEUS_ENABLED`              | If Prometheus metrics are exposed. Defaults to `true`                                                                                                                                                                 |
| `PROMETHEUS_PORT`                 | Port. Defaults to `8080` (same as `SERVER_PORT`). If it differs from `SERVER_PORT`, a separate Prometheus server is started on that port. Please also see the listen and base path environment variables in addition. |
| `PROMETHEUS_LISTEN`               | Prometheus's listen address. Defaults to empty which equals `0.0.0.0`                                                                                                                                                 |
| `PROMETHEUS_BASE_PATH`            | Prometheus's base path. Defaults to `/`, must always end with trailing slash                                                                                                                                          |
| `PROMETHEUS_METRICS_PATH`         | Defines the metrics endpoint path. Defaults to `/metrics` (adheres to `PROMETHEUS_BASE_PATH`)                                                                                                                         |
| `PROMETHEUS_SECURE_TOKEN_ENABLED` | If Prometheus metrics endpoint is protected by a token when enabled (**recommended**). Defaults to `false`                                                                                                            |
| `PROMETHEUS_SECURE_TOKEN`         | The token securing the metrics endpoint when enabled (**recommended**)                                                                                                                                                |
| `SERVER_BASE_PATH`                | Server's base path. Defaults to `/`, must always end with trailing slash                                                                                                                                              |
| `SERVER_LISTEN`                   | Server's listen address. Defaults to empty which equals `0.0.0.0`                                                                                                                                                     |
| `SERVER_PORT`                     | Port. Defaults to `8080`                                                                                                                                                                                              |
| `SERVER_TIMEOUT`                  | Timeout the server waits before shutting down to end any pending tasks. Defaults to `10s` (10 second), qualifier can be `s = second`, `m = minute`, `h = hour` prefixed with a positive number                        |
| `SERVER_TLS_CERT_PATH`            | When TLS enabled, provide the certificate path                                                                                                                                                                        |
| `SERVER_TLS_ENABLED`              | If server uses TLS. Defaults `false`                                                                                                                                                                                  |
| `SERVER_TLS_KEY_PATH`             | When TLS enabled, provide the key path                                                                                                                                                                                |
| `LOGGING_LEVEL`                   | Logging level. Possible are `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`, `disabled`. Setting to `trace` enables high verbosity output. Defaults to `info`                                             |
| `LOGGING_LEVEL_REQUESTS`          | Logging level for requests. Possible are `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`, `disabled`. Setting to `trace` enables high verbosity output. Defaults to `disabled`                            |
| `LOGGING_ENCODING`                | Determines how logs are printed to stdout. Possible are `console`, `json`. Defaults to `console`                                                                                                                      |
| `LOGGING_ENCODING_COLORIZE`       | When logs are encoded as `console`, colorizes output. Defaults to `false`                                                                                                                                             |

There are more advanced configuration settings not outlined in the simple table. Head over to the definition
in [config.go](./internal/server/config/config.go) and look for `env:`.

## Development and contribution

The most straight forward way to get started is by looking into available commands inside the `Makefile`.

For the full setup, you need the following tools:

- go (see minimum version in `go.mod`)
- make to execute commands of the `Makefile`

Quick start is to a terminal and run:

```shell
make run
```

### Git workflow

The main branch is `main`. It's protected and only eligible users can push to it. Merge requests to protected branches
are safeguarded: they need review or at least a successful pipeline run to be merged.

- Use conventional commits as commit style and branch naming strategy, e.g., `feat/`, `fix/`, `refactor/`, `chore/`, or
  `ci/`
- **All** merge request commits should have a meaningful commit **title** and **message** stating the **why**
- Use atomic git commits, separate **preparatory** from **functional** commits to speed up review
- Avoid merging trunk back, use `git-rebase`

### Pipeline workflow

Pipeline runs

* on merge request change (open, new push, ...)
* on protected branches

This means you need to create a merge request to trigger a pipeline run. Without merge request, no build is triggered,
thus your code cannot be merged.

### Dependency updates

Dependency updates are handled by Renovate using the `renovate.json5` file. The base branch is `main`.

Major updates undergo manual review.

### Releases

> Use the `v` prefix in the Forge. Don't use it for internal version code references!

1. Prepare a new MR to trunk with the following changes
    * Adjust and align versions
        * `flake.nix`: `version`
        * `internal/meta/pkg.go`: `Version`
    * Make sure `make clean dependencies checkstyle build-all test-coverage` is fine
    * Make sure `nix build` is fine (you need `nix` for it, update checksums in `flake.nix` if it fails)
      ```shell
      nix build .#packages.x86_64-linux.default -L
      nix build .#packages.aarch64-linux.default -L
      ```
    * Use `release/` as branch prefix and `release: prepare XYZ` as commit message
2. Merge to trunk
3. Trigger the release job the semantic version which is inside the main trunk (use `v` prefix!)
4. Generate changelog and attach it to the release (use `git-cliff`)
5. Pull changes from trunk, prepare a new MR to trunk to prepare next version
    *  Adjust and align versions to the next semantic _patch_ version
    * `flake.nix`: `version`
    * `internal/meta/pkg.go`: `Version`
    * Use `release/` as branch prefix and `release: prepare next cycle...` as commit message
6. Merge to trunk

### Dependency updates

Dependency updates are handled by Renovate using the `renovate.json5` file. The base branch is `main`.

Major updates undergo manual review.
