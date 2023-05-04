# Security

The following document describes various aspects of the Wormhole security program.

## Table of Contents
- [3rd Party Security Audits](#3rd-Party-Security-Audits)
- [Bug Bounty Program](#Bug-Bounty-Program)
- [Trust Assumptions](#Trust-Assumptions)
- [White Hat Hacking](#White-Hat-Hacking)
- [Chain Integrators](#Chain-Integrators)
- [Social Media Monitoring](#Social-Media-Monitoring)
- [Incident Response](#Incident-Response)
- [Emergency Shutdown](#Emergency-Shutdown)
- [Security Monitoring](#Security-Monitoring)
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
- **[October 2022 - Hacken](https://storage.googleapis.com/wormhole-audits/Wormhole_dApp_NEAR_AuditReport_Hacken_2022-10-25.pdf)**: _NEAR Integration_
- **[November 2022 - Zellic](https://storage.googleapis.com/wormhole-audits/Wormhole_Aptos_Audit_Report_Zellic_2022-11.pdf)**: _Aptos Integration_
- **[February 2023 - OtterSec](https://storage.googleapis.com/wormhole-audits/Wormhole_OtterSec_Aptos_NFT_2023-02.pdf)**: _Aptos NFT Bridge_
- **[April 2023 - Trail of Bits](https://storage.googleapis.com/wormhole-audits/Wormhole_Audit_Report_TrailOfBits_2023-04.pdf)**: _Guardian node: Governor and Watchers_
- **Q4 2022 - Halborn (DRAFT)**: _Wormchain_
- **Q4 2022 - Halborn (DRAFT)**: _Accounting_
- **Q4 2022 - Certik (DRAFT)**: _Ethereum Contracts_
- **Q4 2022 - Certik (DRAFT)**: _Solana Contracts_
- **Q4 2022 - Certik (DRAFT)**: _Terra Contracts_
- **Q4 2022 - Certik (DRAFT)**: _Guardian_
- **Q4 2022 - Certik (DRAFT)**: _Solitaire_
- **Q4 2022 - Coinspect (DRAFT)**: _Algorand Contracts_
- **Q4 2022 - Hacken (DRAFT)**: _NEAR Contracts_
- **Q2 2023 - Runtime Verification (IN PROGRESS)**: _Guardian_


## Bug Bounty Program

The Wormhole project operates two bug bounty programs to financially incentivize independent researchers for finding and responsibly disclosing security issues.

- [Self-Hosted Program](https://wormhole.com/bounty/)
  - **Scopes**: Guardian and Smart Contracts
  - **Rewards**: Up to $2,500,000 USDC
  - **KYC**: Required
- [Immunefi-Hosted Program](https://immunefi.com/bounty/wormhole/)
  - **Scopes**: Guardian and Smart Contracts
  - **Rewards**: Up to $2,500,000 USDC
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

- Review the existing unit and integration testing (found in [CONTRIBUTING.md](https://github.com/wormhole-foundation/wormhole/blob/main/CONTRIBUTING.md)) and see what is already being testing for.
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

As the list of chains connected to Wormhole increases, so does the risk that a given connected could introduce risks to the Wormhole network. As a result, Wormhole does have built-in safety features (e.g.: [Governor white-paper](https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0007_governor.md)) to reduce the "blast radius" of such case. That said, a defense in depth strategy is required to do as much as possible to secure the network. As part of this methodology, the Wormhole project recommends that all connected chains current and future implement robust security programs of their own to do their part in managing chain compromise risk to the wormhole network.

Here are a few ways in which connected chains can maintain high security standards:

For source code ensure relevant bits are:

- All open source (required)
- Audited by an independent third party with public audit reports
- Included in a public bug bounty program. The bounty rewards should be sufficiently large to incentivize white-hat mindshare in finding security bugs and responsibly disclosing them
- Version control systems contain adequate access controls and mandatory code review (e.g.: In github, use of branch protection and a minimum of one independent reviewer to merge code)
- Maintaining a [SECURITY.md](https://github.com/wormhole-foundation/wormhole/blob/main/SECURITY.md) in the root of the repository (like this one) to offer guidance and transparency on security relevant topics
- Includes sufficient unit and integration test coverage (including negative tests), which are run on every commit via continuous integration. Ensure that the results of those test runs are visible to the public

Additionally, ensure:

- The Wormhole team has sufficient contact information and an associated call or page tree to reach you in the event of a security incident.
- That Wormhole has the full upgrade authority on relevant bridge contracts to act quickly in the case of a security incident.
- You have an established incident response program in place, with established patterns and playbooks to ensure deterministic outcomes for containment.
- When security issues do occur, please make sure that the chain makes every attempt to inform affected parties and leads with transparency.

## Social Media Monitoring

The Wormhole project maintains a social media monitoring program to stay abreast of important ecosystem developments.

These developments include monitoring services like Twitter for key phrases and patterns such that the Wormhole project is informed of a compromise or vulnerability in a dependancy that could negatively affect Wormhole, its users, or the chains that Wormhole is connected to.

In the case of a large ecosystem development that requires response, the Wormhole project will engage its security incident response program.

## Incident Response

The Wormhole project maintains an incident response program to respond to vulnerabilities or active threats to Wormhole, its users, or the ecosystems it's connected to.  Wormhole can be made aware about a security event from a variety of different sources (eg. bug bounty program, audit finding, security monitoring, social media, etc.)

When a Wormhole project contributor becomes aware of a security event, that contributor immediately holds the role of [incident commander](https://en.wikipedia.org/wiki/Incident_commander) for the issue until they hand off to a more appropriate incident commander.  A contributor does not need to be a "security person" or have any special priviledges to hold the role of incident commander, they simply need to be responsible, communicate effectively, and maintain the following obligations to manage the incident to completion.

The role of the incident commander for Wormhole includes the following minimum obligations:

- Understand what is going on, the severity, and advance the state of the incident.
- Identify and contact the relevant responders needed to address the issue.
- Identify what actions are needed for containment (eg. security patch, contracts deployed, governance ceremony).
- Establish a dedicated real-time communication channel for responders to coordinate (eg. Slack, Telegram, Signal, or Zoom).
- Establish a private incident document, where the problem, timeline, actions, artifacts, lessons learned, etc. can be tracked and shared with responders.
- When an incident is over, host a [retrospective](https://en.wikipedia.org/wiki/Retrospective) with key responders to understand how things could be handled better in the future (this is a no blame session, the goal is objectively about improving Wormhole's readiness and response capability in the future).
- Create issues in relevant ticket trackers for actions based on lessons learned.

## Emergency Shutdown

The Wormhole project has evaluated the concept of having an safety feature, which could allow Wormhole smart contracts to be paused during a state of existential crisis without contract upgrades.  However, after careful consideration the project has chosen to not introduce such a feature at this time.

The rationale for this decision included the following considerations:

- Introduces new trust assumptions for the protocol specific to shutdown events.
- Adds additional engineering overhead to design, develop, and maintain shutdown capability for each supported run-time.
- Adds attack surface to Wormhole related smart contracts, which could introduce new bugs.
- Adds gas cost on every transaction for a feature that will be used maybe once a year.
- Makes assumptions about where in the code existential crisis may surface.

As an alternative approach, Wormhole has evolved its emergency shutdown strategy to leverage the existing upgrade authority via governance to temporarily patch smart contracts for vulnerabilities when/if they occur (eg. temporarily disabling the affected function) and a long-term bug fix is not readily available.

The benefits of such an approach include the following:

- Maintain the same trust assumptions for existential events
- Allow selective shutdown of only the affected code (while leaving everything else fully functional)
- No additional attack surface (only less attack surface during existential crisis)
- No additional gas cost paid by users of Wormhole
- No additional process or keys to distribute or manage for Guardians
- For known shutdown cases, a shutdown contract with disabled capabilities can be pre-deployed on chain, making the governance proposal easier to produce and approve.
- For unknown shutdown cases, a temporary patch can be developed to disable only the affected functionality.

The caveats of such an approach include the following:

- Speed to shutdown is limited by speed to develop the temporary bug fix (only for the unknown cases, known cases won't require development)
- Speed to shutdown is limited by speed at which goverance can be passed to accept the temporary bug fix (slower for unknown cases and faster for known cases)
- Restoring after a shutdown will require a secondary governance action to either repoint the proxy contract to a non-shutdown implementation (known cases) or to revert the temporary patch and apply the long term patch (unknown cases)

## Security Monitoring

The Wormhole project expects all Guardians develop and maintain their own security monitoring strategies.  This expectation is based on the value of having heterogeneous monitoring strategies across the Guardian set as a function of Wormhole's defense in depth approach, increasing the likelihood of detecting fraudulent activity.

Wormhole Guardians should aim to capture all of the following domains with their monitoring strategies:

- Guardian Application, System, and Network Activity
- Gossip Network Activity
- Smart Contract Activity/State
- Transaction/Usage Activity
- Governor Activity

Guardians are encouraged to share monitoring lessons learned with each other to the extent that it increases the ability to detect fraudulent activity on the network.  However, the end state for Wormhole network monitoring is not a homogeneous monitoring strategy, as levels of diversity within the Guardians is an essential property of the Wormhole network.

Lastly, if a Guardian detects a security event via their monitoring strategy they are empowered to engage the above mentioned incident response pattern.
