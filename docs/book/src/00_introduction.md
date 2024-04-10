# Cluster API Operator

The **Cluster API Operator** is a Kubernetes Operator designed to empower cluster administrators to handle the lifecycle of Cluster API providers within a management cluster using a declarative approach. It aims to improve user experience in deploying and managing Cluster API, making it easier to handle day-to-day tasks and automate workflows with GitOps. 

This operator leverages a declarative API and extends the capabilities of the `clusterctl` CLI, allowing greater flexibility and configuration options for cluster administrators.

## Features

- Offers a **declarative API** that simplifies the management of Cluster API providers and enables GitOps workflows.
- Facilitates **provider upgrades and downgrades** making it more convenient for distributed teams and CI pipelines.
- Aims to support **air-gapped environments** without direct access to GitHub/GitLab.
- Leverages **controller-runtime** configuration API for a more flexible Cluster API providers setup.
- Provides a **transparent and effective** way to interact with various Cluster API components on the management cluster.

## Getting started

* [Quick Start](./01_user/02_quick-start.md)
* [Concepts](./01_user/01_concepts.md)
* [Developer guide](./04_developer/02_guide.md)
* [Contributing](./05_reference/04_contributing.md)

