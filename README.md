[![Build Status](https://api.travis-ci.com/slrz/synapse-operator.svg?branch=master)](https://travis-ci.com/slrz/synapse-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/slrz/synapse-operator)](https://goreportcard.com/report/github.com/slrz/synapse-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

# Synapse Operator

A Kubernetes Operator for running the Synapse Matrix homeserver. Based on the
[Operator SDK](https://github.com/operator-framework/operator-sdk).

This is still very much a work-in-progress. Don't use it in production.

## Supported Custom Resources
| *CustomResourceDefinition*                                | *Description*               |
| --------------------------------------------------------- | --------------------------- |
| [Synapse](config/crd/bases/matrix.slrz.net_synapses.yaml) | Manage a Synapse homeserver |


## Creating a Synapse Instance

Minimal example manifest for a Synapse instance at `matrix.example.com`, opting
in to anonymous statistics reporting (`reportStats: yes`).

```yaml
apiVersion: matrix.slrz.net/v1alpha1
kind: Synapse
metadata:
  name: mysynapse
spec:
  serverName: matrix.example.com
  reportStats: yes

```

## License

* [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0)
