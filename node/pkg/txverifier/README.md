# Transfer Verifier - Package Documentation

## Overview

The package is organized by runtime environment. Currently there are implementations for the Ethereum and Sui blockchains.
Because the Ethereum implementation is (hopefully) generalizable to other EVM-chains, it is referred to as 
`evm` implementation rather than the `ethereum` implementation

For each implementation, the code is divided into separate files. The core logic is contained in the main file
and the supporting structs and utility methods are defined in a separate file. The hope here is that this makes the
overall algorithm easier to reason about: a developer new to the program can focus on the main file and high-level
concepts and avoid low-level details.

### Main file -- Core Algorithm

The main file contains the algorithm for Transfer Verification, handling tasks such as tracking deposits and transfers
into the Token Bridge, cross-referencing these with messages emitted from the core bridge, and emitting errors when
suspicious activity is detected.

### Structs file -- Parsing and Encapsulation

The structs file defines the major conceptual building blocks used by the algorithm in the main file. It is also responsible
for lower-level operations such as establishing a subscription or polling mechanisms to a supported chain. This file
also handles parsing and conversions, transforming things like JSON blobs or byte slices into concepts like a
Message Receipt or Deposit. 

### Utilities file

There is also a utilities file that contains functions used by more than one runtime implementation, such as
performing de/normalization of decimals.
