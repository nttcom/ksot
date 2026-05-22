---
title: "Concepts"
linkTitle: "Concepts"
weight: 3
description: >
   Concepts, architecture, and other technical information about K-SOT.
---


## Network Configuration as Code with Go

K-SOT supports abstracting complex network device configurations and exposing a high-level data model and API that can easily describe user intent. The data model of a network device is too detailed and complex to be treated as an Infrastructure as Code resource model, as it provides a multitude of networking functions. In order to provide a data model that can easily represent user intent, a high-level data model is needed that hides the complexity and abstracts it in an understandable manner. In addition, to achieve a model that abstracts the entire domain, such as an E2E connection, requires a data model that is configured across multiple network devices.

With such abstraction to a high-level model, the data mapping from the high-level model to the low-level equipment configurations becomes very complex.
Because of the many-to-many relationship between the upper and lower models, in addition to data mapping, data synthesis must also be performed simultaneously. At this time, care must be taken to avoid missing data, conflicts, and data constraint violations.
A simple text template approach using Python/Jinja is difficult to solve this complex problem.

With K-SOT, the data mapping logic from the upper model to the lower model can be written in Go. In addition, data composition can be implemented without user awareness.

## Data Management with GitHub

K-SOT provides data management via GitHub for network device configurations. It is intended to be a single source of truth (SSoT) that can be trusted by declaratively describing the configuration of the entire network and versioning it in a GitHub repository.

## Components of K-SOT

K-SOT consists of the following components:

- **nb-server** receives the service model and performs derivation of the device configuration.

- **github-server** updates GitHub.

- **sb-server** submits the configuration to the device.
