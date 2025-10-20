# PowerOcean

## Device specific functionality

### Historical Data

EcoFlow offers an endpoint to query **historical data** for PowerOcean devices. This is supported in EcoLinker.

The returned data of the historical data endpoint EcoFlow offers shows the same data as visible inside the mobile
application, though, the labels might be confusing:

**To Home** is actually the **Total Consumption** shown in the upper left corner of the official mobile application
for a day/week. To calculate the "From Solar" shown in the bottom left corner:
`From Solar = To Home - From Battery - From Grid`.

**From Solar** is actually the **Total Generation** shown in the upper left corner of the official mobile application
for a day/week. To calculate the "To Home" shown in the bottom left corner:
`To Home = From Solar - To Battery - To Grid`.

#### Use the [command-line interface client](./../README.md#command-line-interface-client) to fetch data

* Ability to return data in CSV format, e.g., with `--csv`
* Ability to query data with daily ranges for a time range, e.g., with `--interval daily`
* Ability to transform into row-wise output format, e.g., with `--row-wise`

Example for PowerOcean retrieving historical data from EcoFlow's API:

```shell
./ecolinker ecoflow hs --sn 7QX3V2A9KLM8J5WC --begin-time "2025-06-06 00:00:00" --end-time "2025-06-06 23:59:59"

Attribute        | Value    | Unit
From Solar       | 34.5     | kWh
To Battery       | 9.22     | kWh
From Battery     | 10.54    | kWh
From Grid        | 0.08     | kWh
To Grid          | 15.8     | kWh
To Home          | 20.1     | kWh
Self-sufficiency | 99.7     | %
```

#### Manage a collector type to fetch this data automatically

```shell
./ecolinker collectors add device-historical-data --sn 7QX3V2A9KLM8J5WC --frequency 23h --step daily
```

The `--step` and `--frequency` is important. As the provided data is _past_ historical data, EcoLinker queries yesterday
for step daily and last full week (Mon-Sun) for step weekly.

When configured, the collector exposes the following values as Prometheus metric. When MQTT forwarding is enabled, it
also forwards on the collector topic for any 3rd party consumer.

Here's an example from EcoLinker's collector Prometheus results for this historical data:

```shell
ecolinker_historical_data{attribute="From Battery",device="7QX3V2A9KLM8J5WC",end="2025-10-19 23:59:59",start="2025-10-13 00:00:00",unit="kWh"} 20.03
ecolinker_historical_data{attribute="From Grid",device="7QX3V2A9KLM8J5WC",end="2025-10-19 23:59:59",start="2025-10-13 00:00:00",unit="kWh"} 76.38
ecolinker_historical_data{attribute="From Solar",device="7QX3V2A9KLM8J5WC",end="2025-10-19 23:59:59",start="2025-10-13 00:00:00",unit="kWh"} 63.55
ecolinker_historical_data{attribute="Self-sufficiency",device="7QX3V2A9KLM8J5WC",end="2025-10-19 23:59:59",start="2025-10-13 00:00:00",unit="%"} 43.5
ecolinker_historical_data{attribute="To Battery",device="7QX3V2A9KLM8J5WC",end="2025-10-19 23:59:59",start="2025-10-13 00:00:00",unit="kWh"} 20.88
ecolinker_historical_data{attribute="To Grid",device="7QX3V2A9KLM8J5WC",end="2025-10-19 23:59:59",start="2025-10-13 00:00:00",unit="kWh"} 4.08
ecolinker_historical_data{attribute="To Home",device="7QX3V2A9KLM8J5WC",end="2025-10-19 23:59:59",start="2025-10-13 00:00:00",unit="kWh"} 135
```

## Grafana Dashboard

This is how a Grafana dashboard can look like, the source is [dashboard.json](./dashboard.json).

![00.png](00.png)

![01.png](01.png)

![02.png](02.png)

![03.png](03.png)
