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
