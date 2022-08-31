# Security

## Bug Bounty Program

We operate a **[bug bounty program](https://immunefi.com/bounty/wormhole/)** to financially incentivize independent researchers (with up to $10,000,000 USDC) to find and responsibly disclose security issues in Wormhole.

If you find a security issue in wormhole, we ask that you immediately **[report the bug](https://immunefi.com/bounty/wormhole/)** to our security team.

## 3rd Party Security Audits

We engage 3rd party firms to conduct independent security audits of Wormhole. At any given time, we likely have multiple audit streams in progress.

As these 3rd party audits are completed and issues are sufficiently addressed, we make those audit reports public.

- **[January 10, 2022 - Neodyme](https://storage.googleapis.com/wormhole-audits/2022-01-10_neodyme.pdf)**
  - **Scopes**: _Ethereum Contracts, Solana Contracts, Terra Contracts, Guardian, and Solitaire_
- **[July 1, 2022 - Kudelski](https://storage.googleapis.com/wormhole-audits/2022-07-01_kudelski.pdf)**
  - **Scopes**: _Ethereum Contracts, Solana Contracts, Terra Contracts, and Guardian_
- **[August 16, 2022 - Kudelski](https://storage.googleapis.com/wormhole-audits/2022-08-16_kudelski.pdf)**
  - **Scopes**: _Algorand Contracts_

## White-Hat Hacking on Wormhole

We want to lower the bar for White-hat hackers to find security bugs in Wormhole. Why? The easier we make this process, the more likely it will be for white-hats to find bugs in Wormhole and responsibly disclose them, helping to secure the network.

Here's a list of strategies we've found helpful for getting started on Wormhole:

- Review the existing unit and integration testing (found in [CONTRIBUTING.md](https://github.com/wormhole-foundation/wormhole/blob/dev.v2/CONTRIBUTING.md)) and see what we're already testing for.
  - Check out places were we might be missing test coverage entirely. This could be a ripe spot to look for something we missed.
  - Check out places were we have unit/integration tests, but we lack sufficient [negative test](https://en.wikipedia.org/wiki/Negative_testing) coverage.
- Review our different smart contract implementations (eg. Solana, EVM, CosmWasm, Move) and attempt to understand how and why they are different.
  - Does one chain have a safety check that another chain doesn't?
  - Does one chain have a specific set of nuances / gotchas that that were missed on another chain?
- Consider going beyond the source code
  - Review the deployed contracts on chain. Is something odd that we missed?

We'll continue to iterate on this list of white-hat bootstrap strategies as we grow our lessons learned internally hacking on Wormhole and from other white-hats who have been successful via our bug bounty program.

It's important to remember this is an iterative process. If you spend the time to come up with a new test case, but didn't actually find a bug, we'd be extremely appreciative if you'd be willing to send a [pull request](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request) with additional positive and negative test cases. This process has shown repeatedly to improve your ability to understand Wormhole, and will increase your odds of success.

## Guidance to Chain Integrators

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
- When security issues do occur, that the chain makes every attempt to inform affected parties and leads with transparency.
