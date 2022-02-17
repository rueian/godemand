# Godemand

Generalized resource reconciliation framework by pluggable resource controller binaries.

## Features

* Pluggable resource controllers. Controllers can be implemented in any excutable format.
* Supports hot reloading resource controller binaries.

## How it works

Each resource controller defined what `Resource Pool`s they managed through a yaml config file.

And Godemand reads the config file and exposes resource controller's `FindResource` method via HTTP API to users.

It is the responsibility of `FindResource` to create a new resource record or pick a existing one remembered by Godemand.

Godemand will remember all resource records created by the controller and call the its `SyncResource` method periodically.

It is the responsibility of `SyncResource` to manage the real status of the resource.

## Example Use Case

Dynamically scale out PostgreSQL instances from GCP snapshot when clients connected through pgbroker proxy.

https://github.com/rueian/godemand-example



