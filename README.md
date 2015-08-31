# redis-cluster

redis cluster made easy.

## How it works

* retrieves redis cluster members from aws tag
* based on availability zones assigns masters and slaves nodes

## Usage 

redis-cluster --tag redis --masters 3
