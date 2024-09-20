# Overview

The Kyma Control Plane is billing hyperscaler resources used by SKRs using the Kyma Metrics Collector being integrated with Unified Metering via EDP.

## Architecture

Every Kyma cluster is running in a hyperscaler account dedicated for the related global account. So it is shared with many cluster of the same customer. The hyperscaler account is payed by Kyma and individual resource usage gets charged to the customer. The bill to the end user contains one entry, listing the consumed Capacity Units (CU) without any further breakdown. The bill gets created by the Unified Metering service.

[!arch](./assets/arch.drawio.svg)

For that the following steps are undertaken periodically:
1. KMC workers fetch the list of billable clusters from [Kyma Environment Broker (KEB)](https://github.com/kyma-project/kyma-environment-broker/tree/main) and adds them to a queue to work through them. If an error occurs, KMC re-queues the affected runtime. For every process step following, internal metrics are exposed with the [Prometheus client library](https://github.com/prometheus/client_golang). See the [metrics.md](./metrics.md) file for exposed metrics.
2. KMC fetches the kubeconfig for every cluster from the control plane resources
3. KMC retrieves specific Kubernetes resources from the APIServer of every cluster using the related kubeconfig
4. KMC maps the retrieved Kubernetes resources to a Memory/CPU/Storage value and send it to EDP as event stream
5. EDP is calculating the consumed CUs based on the consumed CPU or storage by a fixed formula and sends the consumed CUs to Unified Metering

KMC retrieves the amount of following resource types from the SKR APIServer:
- node type - via the labeled machine type it maps how much memory and cpu the node provides and maps it to an amount of CPU
- storage - for every storage it determines the provisioned GB value
- services - not in use at the moment

## EDP interface

The data send to EDP has to adhere to the following [schema](./assets/edp.json).

An example payload looks like this:

```json
{
  "compute": {
    "vm_types": [
      {
        "name": "Standard_D8_v3",
        "count": 3
      },
      {
        "name": "Standard_D6_v3",
        "count": 2
      }
    ],
    "provisioned_cpus": 24,
    "provisioned_ram_gb": 96,
    "provisioned_volumes": {
      "size_gb_total": 150,
      "count": 3,
      "size_gb_rounded": 192
    }
  },
  "networking": {
    "provisioned_vnets": 2,
    "provisioned_ips": 3
  }
}
```