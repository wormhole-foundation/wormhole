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