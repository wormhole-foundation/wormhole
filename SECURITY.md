# Security

## Bug Bounty Program

We operate a **[bug bounty program](https://immunefi.com/bounty/wormhole/)** to financially incentivize independent researchers (with up to $10,000,000 USDC) to find and responsibly disclose security issues in Wormhole.

If you find a security issue in wormhole, we ask that you immediately **[report the bug](https://immunefi.com/bounty/wormhole/)** to our security team.

## 3rd Party Security Audits

We engage 3rd party firms to conduct independent security audits of Wormhole.  At any given time, we likely have multiple audit streams in progress.

As these 3rd party audits are completed and issues are sufficiently addressed, we make those audit reports public.

- **[January, 10, 2022 - Neodyme](https://storage.googleapis.com/wormhole-audits/2022-01-10_neodyme.pdf)**
    - **Scopes**: *Ethereum Contracts, Solana Contracts, Terra Contracts, Guardian, and Solitaire*
- **[July 1, 2022 - Kudelski](https://storage.googleapis.com/wormhole-audits/2022-07-01_kudelski.pdf)**
    - **Scopes**: *Ethereum Contracts, Solana Contracts, Terra Contracts, and Guardian*

## White-Hat Hacking on Wormhole

We want to lower the bar for White-hat hackers to find security bugs in Wormhole.  Why? The easier we make this process, the more likely it will be for white-hats to find bugs in Wormhole and responsibly disclose them, helping to secure the network.

Here's a list of strategies that we've found helpful to hackers getting started on Wormhole:

- Review the existing unit and integration testing (found in CONTRIBUTING.md) and see what we're already testing for.
    * Check out places were we might be missing test coverage entirely.  This could be a ripe spot to look for something we missed.
    * Check out places were we have unit/integration tests, but we lack sufficient [negative test](https://en.wikipedia.org/wiki/Negative_testing) coverage.
- Review our different smart contract implementations (eg. Solana, EVM, CosmWasm) and attempt to understand how and why they are different.
    * Does one chain have a safety check that another chain doesn't?
    * Does one chain have a specific set of nuances gotchas that that we missed?
- Consider going beyond the source code
    * Review the deployed contracts on chain, is some sort of odd that we missed? 

We'll continue to iterate on this list of white-hat bootstrap strategies as we grow our lessons learned internally hacking on Wormhole and from other white-hats who have been successful via our bug bounty program.

It's important to remember that this is an iterative process, if you spend the time to come up with a new test case, but didn't actually find a bug, we'd be extremely appreciative if you'd be willing to send a [pull request](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request) with additional positive and negative test cases.  This process has shown repeatedly to improve your ability to understand Wormhole, and will increase your odds of success.

## Guidance to Chain Integrators

As the list of chains connected to Wormhole increases, so does the risk that a given connected could introduce risks to the Wormhole network.  As a result, Wormhole does have a built-in safety feature (see [Governor white-paper](https://github.com/certusone/wormhole/blob/dev.v2/whitepapers/0007_governor.md)) to try and coreduce the impact of such case.  That said, a defense in depth strategy is required to do as much as possible to secure the network.  As part of this methodology, the Wormhole project recommends that all connected chains current and future implement robust security programs of their own to do their part in managing chain compromise risk to the wormhole network.

Here are a few ways in which connected chains can help can ensure safety of the Wormhole network by maintaining high security standards:

- Ensure that all relevant source code is open source.
- Ensure that all relevant source code is audited by an independant third party and that audit reports are made available to the public.
- Ensure that all relevant source code is included in a public bug bounty program and that the bounty rewards are sufficiently large to incentivize white-hat mindshare in finding security bugs in your code and responsibly disclosing them.
- Ensure that all relevant source code makes use of branch protection and have a minimum of one independant reviewer to merge code.
- Ensure that all relevant source code maintains a SECURITY.md in the root of the repository (like this one) to offer guidance and transparency on security relevant topics.
- Ensure that all relevant source code has sufficient unit test and integration test coverage, which is run on every commit via continuous integration.  Additionally, ensure that the results of those test runs are visible to the public.
- Ensure that the Wormhole team has sufficient contact information and an associated call or page tree to reach you in the event of a security incident.
- Ensure that Wormhole has the full upgrade authority on relevant bridge contracts to act quickly in the case of security incident.
- Ensure that you have an established incident response program in place, with established patterns and playbooks to ensure deterministic outcomes for process.
- Ensure that when security issues do occur, that the chain makes every attempt to inform affected parties and lead with transparency.