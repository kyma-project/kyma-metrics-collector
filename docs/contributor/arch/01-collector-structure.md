# 1. Collector Structure

Date: 2024-11-20

## Status

Proposed

## Context

The current implementation of KMC's SKR processing has been built with a monolithic architecture. This has led to a number of issues, including:
- a single issue processing an SKR renders the entire billing process for this cluster invalid
- adding new ressources to the billing process is difficult
- the current implementation is hard to test
- every client needs to implement 

## Decision

We will refactor the current implementation of the SKR processing to a more modular architecture. This will enable us to:
- process each scan result independently
- add new resources to scan easily
- each scan result will contain its own logic for processing the data. Processing will include:
  - extracting all required information for unified metering (such as capacity units calculated for the resource, additional non-billable metrics)
  - converting the scan result to storage / cpu / memory units for the EDP backend

The interfaces and their purpose are defined as follows:
- `Scanner` is an interface for extracting a specific resource related to a single cluster.
- `ScanConverter` is the interface required for all ScanResults. It specifies converting a result to a backend-specific measurement.
- `CollectorSender` is an interface for collecting and sending the scan results (ScanConverter interface) to the backend.

Collectors call the scanners to get the information for a cluster. The collector then processes the results and sends them to the backend.
All processed results are stored in a map with the name of the scanner as the key. This map will then be stored for the next run of the collector.