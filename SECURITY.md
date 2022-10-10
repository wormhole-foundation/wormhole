# Security

## Table of Contents
- [3rd Party Security Audits](#3rd-Party-Security-Audits)
- [Bug Bounty Program](#Bug-Bounty-Program)
- [Trust Assumptions](#Trust-Assumptions)
- [White Hat Hacking](#White-Hat-Hacking)
- [Chain Integrators](#Chain-Integrators)
## 3rd Party Security Audits

The Wormhole project engages 3rd party firms to conduct independent security audits of Wormhole. At any given time, multiple audit streams are likely in progress.

As these 3rd party audits are completed and issues are sufficiently addressed, we make those audit reports public.

- **[January 2022 - Neodyme](https://storage.googleapis.com/wormhole-audits/2022-01-10_neodyme.pdf)**: _Ethereum Contracts_
- **[January 2022 - Neodyme](https://storage.googleapis.com/wormhole-audits/2022-01-10_neodyme.pdf)**: _Solana Contracts_
- **[January 2022 - Neodyme](https://storage.googleapis.com/wormhole-audits/2022-01-10_neodyme.pdf)**: _Terra Contracts_
- **[January 2022 - Neodyme](https://storage.googleapis.com/wormhole-audits/2022-01-10_neodyme.pdf)**: _Guardian_
- **[January 2022 - Neodyme](https://storage.googleapis.com/wormhole-audits/2022-01-10_neodyme.pdf)**: _Solitaire_
- **[July 2022 - Kudelski](https://storage.googleapis.com/wormhole-audits/2022-07-01_kudelski.pdf)**: _Ethereum Contracts_
- **[July 2022 - Kudelski](https://storage.googleapis.com/wormhole-audits/2022-07-01_kudelski.pdf)**: _Solana Contracts_
- **[July 2022 - Kudelski](https://storage.googleapis.com/wormhole-audits/2022-07-01_kudelski.pdf)**: _Terra Contracts_
- **[July 2022 - Kudelski](https://storage.googleapis.com/wormhole-audits/2022-07-01_kudelski.pdf)**: _Guardian_
- **[August 2022 - Kudelski](https://storage.googleapis.com/wormhole-audits/2022-08-16_kudelski.pdf)**: _Algorand Contracts_
- **[September 2022 - OtterSec](https://storage.googleapis.com/wormhole-audits/Wormhole_Near_OtterSec.pdf)**: _NEAR Contracts_
- **[September 2022 - Trail of Bits](https://storage.googleapis.com/wormhole-audits/Wormhole_Audit_Report_TrailOfBits_2022-09.pdf)**: _Solana Contracts_
- **[September 2022 - Trail of Bits](https://storage.googleapis.com/wormhole-audits/Wormhole_Audit_Report_TrailOfBits_2022-09.pdf)**: _CosmWasm Contracts_
- **[October 2022 - OtterSec](https://storage.googleapis.com/wormhole-audits/Wormhole_OtterSec_Aptos_2022-10.pdf)**: _Aptos Contracts_
- **Q3 2022 - Halborn (DRAFT)**: _Wormchain_
- **Q3 2022 - Halborn (DRAFT)**: _Accounting_
- **Q3 2022 - Certik (DRAFT)**: _Ethereum Contracts_
- **Q3 2022 - Certik (DRAFT)**: _Solana Contracts_
- **Q3 2022 - Certik (DRAFT)**: _Terra Contracts_
- **Q3 2022 - Certik (DRAFT)**: _Guardian_
- **Q3 2022 - Certik (DRAFT)**: _Solitaire_
- **Q3 2022 - Coinspect (SCHEDULED)**: _Algorand Contracts_
- **Q3 2022 - Hacken (SCHEDULED)**: _NEAR Contracts_
- **Q1 2023 - Trail of Bits (SCHEDULED)**: _Guardian_


## Bug Bounty Program

The Wormhole project operates two bug bounty programs to financially incentivize independent researchers for finding and responsibly disclosing security issues.

- [Self-Hosted Program](https://wormhole.com/bounty/)
  - **Scopes**: Guardian and Smart Contracts
  - **Rewards**: Up to $10,000,000 USDC
  - **KYC**: Required
- [Immunefi-Hosted Program](https://immunefi.com/bounty/wormhole/)
  - **Scopes**: Guardian and Smart Contracts
  - **Rewards**: Up to $10,000,000 USDC
  - **KYC**: Required

If you find a security issue in Wormhole, please report the issue immediately using one of the two bug bounty programs above.

If there is a duplicate report, either the same reporter or different reporters, the first of the two by timestamp will be accepted as the official bug report and will be subject to the specific terms of the submitting program.

## Trust Assumptions

Consensus on Wormhole is achieved by two subset groups of Guardians (aka: validators) within the Guardian Set, which have the following abilities:

- **Super Majority** (any 2/3+ quorum of Guardians - 13 of 19)
  * Can pass messages
    - Core messaging
    - Token/NFT value movement
  * Can pass governance
    - Set fees
    - Upgrade Contracts
    - Upgrade Guardian Set
- **Super Minority** (any 1/3+ quorum of Guardians - 7 of 19)
  * Can censor messages or governance
    - Refusing to sign observed message(s)
    - Refusing to observe the block chain
    - Refusing to run guardian software

There are 19 Guardians in the current Guardian Set, made up of some of the largest and most reputable staking providers in crypto.  This level of operational security diversity is a useful property in preventing wholesale compromise of the Guardian Set due to operational failures of a single or small number of organizations.

The Guardian Set is expected to grow over time to further decentralize the Wormhole Guardian Set and the Wormhole network.
## White Hat Hacking

The Wormhole project wants to lower the bar for White-hat hackers to find security bugs in Wormhole. Why? The easier this process, the more likely it will be for white-hats to find bugs in Wormhole and responsibly disclose them, helping to secure the network.

Here's a list of strategies that are helpful for getting started on Wormhole:

- Review the existing unit and integration testing (found in [CONTRIBUTING.md](https://github.com/wormhole-foundation/wormhole/blob/dev.v2/CONTRIBUTING.md)) and see what is already being testing for.
  - Check out places where there might be missing test coverage entirely. This could be a ripe spot to look for something we missed.
  - Check out places where there are unit/integration tests, but they lack sufficient [negative test](https://en.wikipedia.org/wiki/Negative_testing) coverage.
- Review different smart contract implementations (eg. Solana, EVM, CosmWasm, Move) and attempt to understand how and why they are different.
  - Does one chain have a safety check that another chain doesn't?
  - Does one chain have a specific set of nuances / gotchas that that were missed on another chain?
- Consider going beyond the source code
  - Review the deployed contracts on chain. Is something odd that may have been missed?

This section will continue iterating on white-hat bootstrap strategies as lessons are learned hacking on Wormhole and from community members.

It's important to remember this is an iterative process and to stay positive. If you spend the time coming up with a new test case, but didn't actually find a bug, please send a [pull request](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request) with additional positive and negative test cases. This process has shown repeatedly to improve your ability to understand Wormhole, and will increase your odds of finding future bugs.

## Chain Integrators

As the list of chains connected to Wormhole increases, so does the risk that a given connected could introduce risks to the Wormhole network. As a result, Wormhole does have built-in safety features (e.g.: [Governor white-paper](https://github.com/wormhole-foundation/wormhole/blob/dev.v2/whitepapers/0007_governor.md)) to reduce the "blast radius" of such case. That said, a defense in depth strategy is required to do as much as possible to secure the network. As part of this methodology, the Wormhole project recommends that all connected chains current and future implement robust security programs of their own to do their part in managing chain compromise risk to the wormhole network.

Here are a few ways in which connected chains can maintain high security standards:

For source code ensure relevant bits are:

- All open source
- Audited by an independent third party with public audit reports
- Included in a public bug bounty program. The bounty rewards should be sufficiently large to incentivize white-hat mindshare in finding security bugs and responsibly disclosing them
- Version control systems contain adequate access controls and mandatory code review (e.g.: In github, use of branch protection and a minimum of one independent reviewer to merge code)
- Maintaining a [SECURITY.md](https://github.com/wormhole-foundation/wormhole/blob/dev.v2/SECURITY.md) in the root of the repository (like this one) to offer guidance and transparency on security relevant topics
- Includes sufficient unit and integration test coverage (including negative tests), which are run on every commit via continuous integration. Ensure that the results of those test runs are visible to the public

Additionally, ensure:

- The Wormhole team has sufficient contact information and an associated call or page tree to reach you in the event of a security incident.
- That Wormhole has the full upgrade authority on relevant bridge contracts to act quickly in the case of a security incident.
- You have an established incident response program in place, with established patterns and playbooks to ensure deterministic outcomes for containment.
- When security issues do occur, please make sure that the chain makes every attempt to inform affected parties and leads with transparency.