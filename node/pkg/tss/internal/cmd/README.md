Please read the entire document before running the protocol.

# Running 'local' Key Generation

The following binary runs a localised version of a DKG protocol (denote lkg) to
generate secrets for guardians to use by the threshold signing scheme (TSS).

The script expects a config file (similar to the cnfg.json) provided
in this package.
The config file contains a few key fields:

```
"NumParticipants": int,
"WantedThreshold": int,
"GuardianSpecifics" : array
```


Where `NumParticipants` is the number of guardians in the system,
`WantedThreshold` is the wanted threshold (For instance, `NumParticipants=19` and `WantedThreshold=13`).

The following is an example of the `GuardianSpecifics` array (for a working example, 
please see *`lkg/cnfg.example.json`*):


```
   "GuardianSpecifics": [
        {
            "Identifier": {
                "TlsX509": PEM X509 CERT in byte format
            },
            "WhereToSaveSecrets": "/Path/To/folder/that/will/contain/the/result"
        },
        {
            "Identifier": {...},
            "WhereToSaveSecrets": "..."
        },
        {...},
        .
        .
        .
   ]
```

The LocalKG protocol is used to generate secrets to TSS, 
and it assumes a public key infrastructure. 
These public keys are x509 certificates (and stored inside `GuardianSpecifics[i].Identifier.TlsX509`), 
and are used later by the TSS to establish TLS channels between the participants.
As a result, the x509 certificate provided by you should be self-signed root-level certificates. 
In addition, you should safely store the signing key you've used to sign your certificate in a known location
since it is still needed by the TSS protocol ([see after running the protocol for further details](#after-running-the-local-key-generation-protocol)).


When creating the X509 certificates, be aware that the DNS name you set 
in the certificate will be used as the hostname of
servers participating in the TSS protocol.
As a result, please refrain from using hostnames that are
unreachable.

# After running the local key generation protocol.

Once you run the protocol, it will generate for each guardian a single directory (as specified in the `WhereToSaveSecrets` field), each such directory should contain a `secret.json` upon lkg completion. 
Each guardian operator should take the `secret.json` file saved to the directory they
provided in the config.
The resulting `secret.json` file should be guarded with care, and out of reach by untrusted entities, and 
out of reach from other guardians (each guardian should have only one `secret.json` file).

Before using the `secret.json` file with the TSS engine, it needs to be set with the private key 
that was used to sign the x509 certificate (the same signed certificate that each guardian operator set in the lkg config file):
Run the `setkey` command, it expects a secrets file generated from running the lkg protocol, and a file for 
a private key in PEM format (see `setkey/lkg.example.json` and `setkey/key.example.pem`).
