# 2. wasmvm will be upgraded to v1.5.2 and not v2.x.x

Date: 2024-06-24

## Status

Accepted

## Context

wasmvm 1.5.2 has an EOL that will arrive sooner, but v2.x.x is a riskier upgrade path.

## Decision

We will upgrade to v1.5.2 and NOT v2.x.x because it was suggested that the risk/reward isn't worth it: too new, 
bugs on tokenfactory for chains which deployed it. We will hold off on that for now.

## Consequences

Not being on wasmvm v2.x.x will leave wormhole gateway missing out on the "factor of 1000" lower cosmwasm gas costs, 
improved submessage ergonomics and ability to query cosmwasm via grpc which are some of the main 2.x benefits. 
Reference: https://medium.com/cosmwasm/cosmwasm-2-0-bbb94126ce6f 
