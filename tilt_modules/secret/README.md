# Secret

Author: [Nick Santos](https://github.com/nicks)

Helper functions for creating Kubernetes secrets.

## Functions

### secret_yaml_generic

```
secret_yaml_generic(name: str, namespace: str = "", from_file: Union[str, List] = None, secret_type: str = None): Blob
```

Returns YAML for a generic secret.

* `from_file` ( str ) – equivalent to `kubectl create secret --from-file`
* `secret_type` ( str ) - equivalent to `kubectl create secret --type`

### secret_create_generic

```
secret_create_generic(name: str, namespace: str = "", from_file: Union[str, List] = None, secret_type: str = None)
```

Deploys a secret to the cluster. Equivalent to

```
load('ext://namespace', 'secret_yaml_generic')
k8s_yaml(secret_yaml_generic('name', from_file=[...]))
```

## Example Usage

### For a Postgres password:

```
load('ext://secret', 'secret_create_generic')
secret_create_generic('pgpass', from_file='.pgpass=./.pgpass')
```

### For Google Cloud Platform Key:

```
load('ext://secret', 'secret_generic_create')
secret_create_generic('gcp-key', from_file='key.json=./gcp-creds.json')
```

## Caveats

- This extension doesn't do any validation to confirm that names or namespaces are valid.
