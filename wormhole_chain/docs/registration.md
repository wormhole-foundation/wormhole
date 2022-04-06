# Register wormhole chain on other chains

The token bridge emitter address is

```
wormhole1zugu6cajc4z7ue29g9wnes9a5ep9cs7yu7rn3z
```

The wormhole (core module) address is:

```
wormhole1ap5vgur5zlgys8whugfegnn43emka567dtq0jl
```

This is deterministically generated from the module.

## Tiltnet

The VAA signed with the tiltnet guardian:

```
0100000000010047464c64a843f49766edc85c9b94b8b142a3315d6cad6c0045fe171f969b68bf52db1f81b9f40ec749b2ca27ebfe7da304c432f278bb9845448595d93a3519af0000000000d1ffc017000100000000000000000000000000000000000000000000000000000000000000045f2397a84b3f90ce20000000000000000000000000000000000000000000546f6b656e4272696467650100000c200000000000000000000000001711cd63b2c545ee6545415d3cc0bda6425c43c4
```

Rendered:

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ Wormhole VAA v1         │ nonce: 3523198999       │ time: 0                  │
│ guardian set #0         │ #6855489806860783822    │ consistency: 32          │
├──────────────────────────────────────────────────────────────────────────────┤
│ Signature:                                                                   │
│   #0: 47464c64a843f49766edc85c9b94b8b142a3315d6cad6c0045fe171f969b...        │
├──────────────────────────────────────────────────────────────────────────────┤
│ Emitter: 11111111111111111111111111111115 (Solana)                           │
╞══════════════════════════════════════════════════════════════════════════════╡
│ Chain registration (TokenBridge)                                             │
│ Emitter chain: Wormhole                                                      │
│ Emitter address: wormhole1zugu6cajc4z7ue29g9wnes9a5ep9cs7yu7rn3z (Wormhole)  │
└──────────────────────────────────────────────────────────────────────────────┘
```

## Testnet

TBD (need to be signed by testnet guardian)

## Mainnet

TBD (need to be signed by the most recent guardian set)
