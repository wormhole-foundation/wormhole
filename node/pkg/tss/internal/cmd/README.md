Please read the entire document before running the protocol.

# Running Distributed Key Generation

The following DKG binary runs a distributed-key-generation protocol to
generate secrets for guardians to use by the threshold signing scheme (TSS).

```
go build -o=./server ./dkg
./server -cnfg=<YOUR CONFIG>
```

As implied in the command above, the binary expects a config file (similar to the config file in `./dkg/1/dkg.json`).
The config file contains a few key fields:

```
"NumParticipants": int,
"WantedThreshold": int,
"Self" : Identifier
"SelfSecret" : PEM encoding of a secret key. (as a byte array)
"Peers" : array of `Identifier`
```


Where `NumParticipants` is the number of guardians in the system,
`WantedThreshold` is the wanted threshold (For instance, `NumParticipants=19` and `WantedThreshold=13`).
`Self` Descries to the binary who it is when running.
`Peers` Describe all possible participants (including `Self`).

To see an example for such a config file, please look into `./cmd/dkg/1/dkg.json`.

#### *Notice:* The DKG.config should contain secret keys. *DO NOT share it with anyone*. 

The DKG protocol is used to generate secrets to TSS, 
and it assumes a public key infrastructure (PKI). 
PKI means that each participant of the DKG protocol know the public key of all of its peers.
These public keys are x509 certificates (and stored inside `Peers[i].TlsX509`), 
and are used later by the TSS to establish TLS channels between the participants.
As a result, the x509 certificate provided by you should be self-signed root-level certificates.
In addition, you should provide put the secret key you've used to sign your certificate in the field `SelfSecret`.

When creating the X509 certificates, be aware that the DNS name you set 
in the certificate will be used as the hostname of
servers participating in the TSS protocol.
As a result, please refrain from using hostnames that are
unreachable.


## Upon Completion
Once you run the protocol, it will generate for each guardian a `secrets.json` fil upon lkg completion. 
This `secrets.json` file contains everything the Guardian process needs to run VAAv2.
Keep this file secret.