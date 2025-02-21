# Guardian Signer

The guardian signer is responsible for signing various operations within the Wormhole ecosystem, such as observations (which results in the creation of VAAs) and gossip messages on the peer-to-peer network (see the [whitepaper](../whitepapers/0009_guardian_key.md)). Historically, the guardian only supported signing using a private key on disk. However, the guardian now allows developers to easily add alternative signing mechanisms through the `GuardianSigner` interface introduced in [PR #4120](https://github.com/wormhole-foundation/wormhole/pull/4120).

The guardian node supports the following signing mechanisms:
* File-based signer - Load a private key from disk, and use it for signing operations.
* Amazon Web Services KMS - Use AWS' KMS for signing operations.

## Usage

### Traditional Usage

For backwards-capability the traditional `guardianKey` command line argument is still supported. The argument accepts a path to a private key file on disk, that is loaded and used for signing operations:

```sh
--guardianKey PATH_TO_GUARDIAN_KEY
```

### Guardian Signer URI Scheme

To make use of alternative signing mechanisms, the `guardianSignerUri` argument can be used. The generic format of the argument is shown below, where `signer` is the name of the mechanism to use and the `signer-config` denotes the configuration of the specified signer. 

```
--guardianSignerUri <signer>://<signer-config>
```

The supported signing mechanisms are tabled below.

| Signer | URI Scheme | Description |
|--------|------------|-------------|
| File Signer | `file://<path-to-file>` | `path-to-file` denotes the path to the private key on disk |
| Amazon Web Services KMS | `amazonkms://<arn>` | `<arn>` denotes the Amazon Resource Name of the Key Management Service (KMS) key to use |

## Setup

### AWS KMS Key Setup

_NOTE_ For the best possible performance, it is recommended that the guardian be run from an EC2 instance that is in the same region as the KMS key.

The KMS key's spec should be `ECC_SECQ_P256K1`, and should be enabled for signing. In order for the guardian to authenticate against the KMS service, one of two options are available:

* Create new API keys in the AWS console that are permissioned to use the KMS key for signing, and add the keys to the EC2 instance's `~/.aws/credentials` file. ([example here](https://docs.aws.amazon.com/cli/v1/userguide/cli-configure-files.html)).
* Create a role that is permissioned to use the KMS key and attach that role to the Guardian EC2 instance.