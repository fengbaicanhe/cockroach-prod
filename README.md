# Cockroach cluster deployment

- [Introduction](#introduction)
- [Amazon Web Services](#amazon-web-services)
- [Google Compute Engine](#google-compute-engine)

## Introduction

Cockroach-prod uses docker machine to create instances on various
cloud services, and docker to start the cockroach processes.

To tie this all together and allow node and cluster discovery, it also creates a load balancer.

Many cloud settings use default values. This is to simplify setup and use. Additional customization
will eventually be added.

The main parameter for all `cockroach-prod` commands is `--region=<driver>:<name>` specifying a
cloud driver to use, and the region within the cloud.

Creating a cockroach cluster with 3 nodes is done as follows:

1. Initialize the first node and setup cloud services
  * initialize load balancer and other cloud infrastructure
  * create one instance
  * initialize cluster and start the first node
  * register node with the load balancer
  ```console
  $ cockroach-prod init --region=<driver>:<region>
  ```

2. Add nodes
  * create 2 new instances
  * start cockroach nodes
  * register nodes with the load balancer
  ```console
  $ cockroach-prod add-nodes 2 --region=<driver>:<region>
  ```

3. Display status
  * display docker-machine status
  * display driver status: print the load balancer address
  ```console
  $ cockroach-prod status --region=<driver>:<region>
  ```

4. Client connections
  * specify the load balancer address (as displayed by cockroach-prod status)
  ```console
  $ cockroach kv scan --addr=<load balancer address>
  ```


#### Prerequisites

  * Build cockroach-prod

  ```console
  $ go get github.com/cockroachdb/cockroach-prod
  ```
  * Install [docker](https://docs.docker.com/installation/) and [docker machine](http://docs.docker.com/machine/)
  * Account on a supported cloud platform. See per-platform pre-requisites.


#### TODOs
* generate and push cockroach certs
* use persistent storage as docker volumes (this is local disk only for now)


## Amazon Web Services

#### Prerequisites

* AWS account
* AWS credentials in `~/.aws/credentials` or environment variables. See [AWS cli configuration](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html#cli-config-files) 

#### Driver

Pick a region from the [list](http://docs.aws.amazon.com/general/latest/gr/rande.html#ec2_region) (eg: `us-east-1`) and invoke using:
```console
$ cockroach-prod <command> --region=aws:us-east-1
```

#### Permissions

The credentials file will be parsed by cockroach-prod to configure the AWS client library, or passed to docker-machine.

## Google Compute Engine

#### Prerequisites

* Google Cloud account
* Create a project name `cockroach-<username>` where username is the local user on your machine.
  If using a different project name, pass the flag `--gce-project=<project-name>` to cockroach-prod.
* Enable the `Google Compute Engine` and `Google Compute Engine Instance Groups API` APIs

#### Driver

Pick a region from the [list](https://cloud.google.com/compute/docs/zones#available) (eg: `us-central1`) and invoke using:
```console
$ cockroach-prod <command> --region=gce:us-central1
```

#### Permissions

You will be prompted for oauth tokens granting permissions to cockroach-prod. The token is then shared with docker-machine.
