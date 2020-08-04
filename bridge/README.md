# The bridge

The bridge is a lightweight, distributed cross-chain notary. 
Read the [the protocol spec](../docs/protocol.md) first.

- **Leaderless**. There is no synchrony/consensus or proposers - the bridge merely observes finalized transactions on
  one chain, signs them using its piece of the joint key, and pushes its signature to an off-chain peer-to-peer network.
  Once 2/3+ of the guardian set agree, the threshold signature is valid and this jointly signed proof 
  (which we call a Verifiable Action Approval or VAA) can be posted to the other chain to release or mint funds that
  were locked/burned on the first.
  
- **Stateless**. Nodes do not keep persistent state about transactions they observed.
