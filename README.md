# Lyra operator example

Experiments with [operator-sdk](https://github.com/operator-framework/operator-sdk). You'll need to install the 'operator-sdk' command line tool to use this repo.

## Pre-requistes

Unlike most Lyra repos, this project doesn't (and won't) support Go modules so you will need to disable support:

    export GO111MODULE=off

## Usage

Before first time use, you need to install the Workflow CRD:

    kubectl create -f deploy/crds/lyra_v1alpha1_workflow_crd.yaml

Create a Workflow resource like this:

    kubectl apply -f deploy/crds/lyra_v1alpha1_workflow_sample.yaml

## Development

You can run the operator directly from this repo like this:

    operator-sdk up local --namespace=default

If you make changes you might need to regenerate the controller code:

    operator-sdk generate k8s

## Deployment

To run the lyra controller in cluster using the Dockerfile in the `lyraproj/lyra` repo

(adapted from [the operator-sdk quick start](https://github.com/operator-framework/operator-sdk/#quick-start))

```sh
    # minikube example, reuse the docker daemon on your machine
    # https://github.com/kubernetes/minikube/blob/0c616a6b42b28a1aab8397f5a9061f8ebbd9f3d9/README.md#reusing-the-docker-daemon
    eval $(minikube docker-env)
    # build the image in the lyraproj/lyra repo using e.g. docker build -t lyraproj/lyra-operator . (in that folder)
    # verify an image has been produced (note CREATED time)
    docker images lyraproj/lyra-operator
    # if applicable push this to image to a public image
    # NOTE: imagePullPolicy is set to Never (instead of Always) in operator.yaml for minikube to use the local docker daemon 
    # create the various k8s resources, including the lyra-operator container
    kubectl create -f deploy/service_account.yaml
    kubectl create -f deploy/role.yaml
    kubectl create -f deploy/role_binding.yaml
    kubectl create -f deploy/crds/lyra_v1alpha1_workflow_crd.yaml
    # create the operator deployment
    kubectl create -f deploy/operator.yaml
    # create an instance of the CRD workflow
    kubectl apply -f deploy/crds/lyra_v1alpha1_workflow_sample.yaml
    # check the logs on the lyra-operator pod
    kubectl get pods | grep "lyra-operator.*1\/1" | awk '{print $1}' | xargs kubectl logs
```