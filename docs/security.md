# Security Properties

- Wormhole is a decentralized proof-of-authority system. All nodes - called guardians - have equal voting power.
  
  A 2/3+ majority is required for guardians to achieve consensus. 
  
  The guardian set will consist of Solana validators, reputable community members, ecosystem stake holders
  and other parties whose incentives strongly align with Solana and Solana ecosystem projects like Serum.
  
    - We believe that this model is easier to implement and reason about and more likely to result in incentive alignment
      with Solana ecosystem stakeholders than launching a separate PoS chain.
     
  - It is possible to add staking in the future.
 
- Wormhole is leaderless. All nodes perform the same computation upon observing an event.

- Wormhole acts as a decentralized cross-chain oracle, observing finalized transactions on one
  chain and producing a joint signed statement of all guardians on another chain.
  
<!-- TODO: to be continued
