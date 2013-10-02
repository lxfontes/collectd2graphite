# Overview

Load data from collectd sent over http (write_http plugin) into graphite.

## Metrics format

  collect.hostname.metric...

Dots in hostnames will be replaced by `_`

## why not use write_graphite ?

We would if we could :)

As of writing, collectd does not ship with `write_graphite` by default.

## collectd config example

    LoadPlugin "write_http"

    <Plugin "write_http">
      <URL "http://collectd2graphite.endpoint:9292">
        Format "JSON"
      </URL>
    </Plugin>

## graphite storage-schemas.conf

    [collectd]
    pattern = ^collectd\..*
    priority = 100
    retentions = 10s:7d,1m:31d,10m:5y


