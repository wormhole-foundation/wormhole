The files named "v1" to "vX" document historic guardian set changes.

Wormhole Core contract deployments that are set to a particular guardian set index only accept VAAs that are signed by a quorum of entities in that set. The VAAs that guardian sets had signed to approve the next guardian set are stored in deployments/mainnet/guardianSetVAAs.csv

----

The files named "dgs1" to "dgsX" document historic delegated guardian set changes.

The guardian node pulls the delegated guardian set configuration from a smart contract on Ethereum, based on which it decides which guardian's observations it must take into account before signing their own observation. These VAAs are stored in deployments/mainnet/delegatedGuardianSetVAAs.csv
