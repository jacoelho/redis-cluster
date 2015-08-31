# redis-cluster

redis cluster auto-assemble and auto-healing using an AWS auto scaling group.

## How it works

* setup an auto scaling group in multiple availability zones
* setup redis-cluster to run at startup (upstart,cloud-init,etc)
* redis-cluster retrieves redis cluster members from aws tag
* based on availability zones assigns masters and slaves nodes

## Usage 

redis-cluster --tag redis --masters 3

This application should be called from within a running instance.
