<p align="center">
<img src="https://github.com/kubernetes/kubernetes/raw/master/logo/logo.png"  width="100x"></a>
</p>
<p align="center">
<a href="https://godoc.org/sigs.k8s.io/cluster-api-operator"><img src="https://godoc.org/sigs.k8s.io/cluster-api-operator?status.svg"></a>
</p>

# Cluster API Operator

Home for Cluster API Operator, a subproject of sig-cluster-lifecycle

## ✨ What is Cluster API Operator?

The **Cluster API Operator** is a Kubernetes Operator designed to empower cluster administrators to handle the lifecycle of Cluster API providers within a management cluster using a declarative approach. It aims to improve user experience in deploying and managing Cluster API, making it easier to handle day-to-day tasks and automate workflows with GitOps. 

This operator leverages a declarative API and extends the capabilities of the `clusterctl` CLI, allowing greater flexibility and configuration options for cluster administrators. 

## 📖 Documentation

Please see our [book](https://cluster-api-operator.sigs.k8s.io) for in-depth documentation.

## 🌟 Features

- Offers a **declarative API** that simplifies the management of Cluster API providers and enables GitOps workflows.
- Facilitates **provider upgrades and downgrades** making it more convenient for distributed teams and CI pipelines.
- Aims to support **air-gapped environments** without direct access to GitHub/GitLab.
- Leverages **controller-runtime** configuration API for a more flexible Cluster API providers setup.
- Provides a **transparent and effective** way to interact with various Cluster API components on the management cluster.

## 🤗 Community, discussion, contribution, and support

You can reach the maintainers of this project at:

- Kubernetes [Slack](http://slack.k8s.io/) in the [#cluster-api-operator][#cluster-api-operator slack] channel

Pull Requests and feedback on issues are very welcome!

See also our [contributor guide](CONTRIBUTING.md) and the Kubernetes [community page] for more details on how to get involved.

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[community page]: https://kubernetes.io/community
[#cluster-api-operator slack]: https://kubernetes.slack.com/archives/C030JD32R8W
[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[Creative Commons 4.0]: https://git.k8s.io/website/LICENSE
