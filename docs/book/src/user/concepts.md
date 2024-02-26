# Concepts

## CoreProvider

A component responsible for providing the fundamental building blocks of the Cluster API. It defines and implements the main Cluster API resources such as Clusters, Machines, and MachineSets, and manages their lifecycle. This includes:

1. Defining the main Cluster API resources and their schemas.
2. Implementing the logic for creating, updating, and deleting these resources.
3. Managing the overall lifecycle of Clusters, Machines, and MachineSets.
4. Providing the base upon which other providers like BootstrapProvider and InfrastructureProvider build.

## BootstrapProvider

A component responsible for turning a server into a Kubernetes node as well as for:

1. Generating the cluster certificates, if not otherwise specified
2. Initializing the control plane, and gating the creation of other nodes until it is complete
3. Joining control plane and worker nodes to the cluster

## ControlPlaneProvider

A component responsible for managing the control plane of a Kubernetes cluster. This includes:

1. Provisioning the control plane nodes.
2. Managing the lifecycle of the control plane, including upgrades and scaling.

## InfrastructureProvider

A component responsible for the provisioning of infrastructure/computational resources required by the Cluster or by Machines (e.g. VMs, networking, etc.). 
For example, cloud Infrastructure Providers include AWS, Azure, and Google, and bare metal Infrastructure Providers include VMware, MAAS, and metal3.io.

## AddonProvider

A component that extends the functionality of Cluster API by providing a solution for managing the installation, configuration, upgrade, and deletion of Cluster add-ons using Helm charts.

## IPAMProvider

A component that manages pools of IP addresses using Kubernetes resources. It serves as a reference implementation for IPAM providers, but can also be used as a simple replacement for DHCP.