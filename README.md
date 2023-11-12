# Pagerduty-operator (WIP)
The objective of this operator is to allow pagerduty services and business services to be created alongside a deployment of an application, automating the setup of on-call. This ensures that a pagerduty service is specific to the application it is "monitoring".

## To do list

- Secret management for PD token
- Setup adapter pattern to mediate between CRD and PD API ✅
    - EscalationPolicy subroutines ✅
    - Service subroutines ✅
    - Clean up CRD to PD Object adapter ✅
- Add business services ✅
- Establish dependencies between CRDs/Objects 
    - Service - Escalation Policy
    - Business Service - Service
- Add testing
- Organize utilities 
- Semantic versioning for commits
- Create helm chart to release
- Pipeline?

## Basic structure (Draft overview)

![Diagram](./PDoperator.drawio.svg)

The operator consists on a manager that manages three Kubernetes Custom Resources controllers: EscalationPolicies, PagerDutyServices and BusinessServices.

As seen on the image, PagerDuty Services depend on Escalation Policies and Business Services depend on PagerDuty Services. However, any of these objects can be created on their own without referencing anything else.

In terms of how the controllers for a specific resource are structured you can take a look at the top right corner of the image. Each controller has a reconciler which runs the Reconcile() function whenever there is a Kubernetes event on an observed object. The objective of the controller is to make the state of the upstream resource in pagerduty match the desired state defined in the custom resource manifest.

This reconcile function will run a set of subroutines each time. These subroutines are idempotent functions that will perform certain actions depending on whether the event is relevant to them or not. As an example, the basic Create subroutine will execute the creation of a Pagerduty object through the API whenever a new custom resource is created, and do nothing when other events occur.

In order to decouple the Custom resources from the pagerduty API objects each controller will have an Adapter. This component is responsible for making sure that the information from the Custom resource in Kubernetes can be successfully translated to objects that PagerDuty API understands in order to make the necessary API calls.

You can see examples of the resource definitions in [`/config/samples`](/config/samples/)


## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/pagerduty-operator:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/pagerduty-operator:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

