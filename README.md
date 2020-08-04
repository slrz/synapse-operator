# Synapse Operator

A Kubernetes Operator for running the Synapse Matrix homeserver. Based on the
[Operator SDK](https://github.com/operator-framework/operator-sdk).

This is still very much a work-in-progress. Don't use it in production.

## Supported Custom Resources
| *CustomResourceDefinition*                                | *Description*               |
| --------------------------------------------------------- | --------------------------- |
| [Synapse](config/crd/bases/matrix.slrz.net_synapsis.yaml) | Manage a Synapse homeserver |


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
