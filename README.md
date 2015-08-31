# redis-cluster

redis cluster auto-assemble and auto-healing using an AWS auto scaling group.

## How it works

### Prerequisites

* setup an auto scaling group (or more) in multiple availability zones
* setup redis-cluster to run at startup (upstart,cloud-init,etc)

### Functionality

* redis-cluster retrieves redis cluster members based on aws tag
* based on availability zones assigns masters and slaves nodes

## Usage 

redis-cluster --tag redis --masters 3

This application should be called from within a running instance.

## FAQ

1. What happens if I change numbers of masters after setup?

It will be ignored.
