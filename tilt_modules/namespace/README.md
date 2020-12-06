# Namespace

Author: [Nick Santos](https://github.com/nicks)

Helper functions for creating Kubernetes namespaces and manipulating
namespaces on Kubernetes objects.

## Functions

### `namespace_yaml(name: str): Blob`

Returns YAML for a Kubernetes namespace.

### `namespace_create(name: str)`

Deploys a namespace to the cluster. Equivalent to

```
load('ext://namespace', 'namespace_yaml')
k8s_yaml(namespace_yaml('name'))
```

### `namespace_inject(objects: Union[str, Blob], namespace: str): Blob`

Given YAML for Kubernetes objects, return new YAML with a different namespace.

## Example Usage

### For a fixed namespace:

```
load('ext://namespace', 'namespace_create', 'namespace_inject')
namespace_create('my-namespace')
k8s_yaml(namespace_inject(read_file('deployment.yaml'), 'my-namespace'))
```

### For a user-specific namespace:

```
load('ext://namespace', 'namespace_create', 'namespace_inject')
ns = 'user-%s' % os.environ.get('USER', 'anonymous')
namespace_create(ns)
k8s_yaml(namespace_inject(read_file('deployment.yaml'), ns))
```

## Caveats

- `namespace_inject` assumes all resources are namespaced-scoped.
  The behavior is undefined for cluster-scoped resources.

- This extension doesn't do any validation to confirm that namespace names are valid.
  The behavior is undefined on invalid namespaces.
