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

## Rigorous Testing and Review

We believe that rigorous testing that is also transparent is critically important to ensuring the integrity of Wormhole components. It is also critically important that everyone can independantly verify and extend Wormhole test cases, which allow us to further ensure that Wormhole components will operate as expected in both positive and especially negative test case scenarios.

Places to find out more about existing test coverage and how to run those tests:

- **Guardian Node**
    - Tests: `./node/**/*_test.go`
    - Run: `cd node && make test`
- **Ethereum Smart Contracts**
    - Tests: `./ethereum/test/*.[js|sol]`
    - Run: `cd ethereum && make test`
- **Solana Smart Contracts**
    - Tests: `./solana/bridge/program/tests/*.rs`
    - Run: `cd solana && make test`
- **Terra Smart Contracts**
    - Tests: `./terra/test/*`
    - Run: `cd terra && make test`

We additionally subscribe to a number of linting frameworks, including gosec, golint, cargo check and others to avoid obvious pitfalls and language specific bugs to provide instant feedback to developers on Wormhole.

The best place to understand how we invoke these tests via GitHub Actions on every commit can be found via `./.github/workflows/*.yml` and the best place to observe the results of these builds can be found via [https://github.com/certusone/wormhole/actions](https://github.com/certusone/wormhole/actions).  Additionally, these results are also available to anyone who submits a PR to the Wormhole project inline, and the team has a technical requirement that all builds must pass and a minimum of 2 passing code reviews before the PR can be merged into the main branch.