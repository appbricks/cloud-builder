# Cloud Builder Automation Services

***Be Safe, Be Secure, Stay Connected***

## Overview

This project implements services for launching [Cloud Builder](https://github.com/appbricks/cloud-builder) automation recipes via a REST API.

## Cloud Automation Service

The automation service is implemented as a collection of AWS lambda functions behind an AWS API gateway. The high-level architecture is given in the diagram below. 

![alt text](docs/images/automation-services.png "Cloud Builder Automation Service")

These functions are invoked by the the Cloud Builder mobile or desktop clients. Using the client a secured network mesh can be built across multiple clouds which will host personal resources and encrypted data, as shown below. Additional functions will be provided to migrate resources and encrypted data across clouds to optimize cloud costs based on recommendations delivered via the client.

![alt text](docs/images/network-mesh.png "Cloud Builder Network Mesh")

This mesh will also provide egress to the internet via a VPN tunnel, and access to all personal resources will traverse the VPN nodes and then the IPSec network mesh providing a secure dark net for hosting personal services.

![alt text](docs/images/client-access.png "Cloud Builder Client")

## Use Cases

### Client Launcher

The launcher use cases describe all client actions around launching cloud builder recipes using a cloud builder client application.

![alt text](docs/images/client-use-cases.png "Client Use Cases")
