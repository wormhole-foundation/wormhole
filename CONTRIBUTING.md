# Contributing

Wormhole is an open-source project licensed under the permissive Apache 2 license. Contributions are greatly
appreciated and will be reviewed swiftly.

Wormhole is a mission-critical, high-stakes project. We optimize for quality over quantity. Design processes
and code reviews are our most important tools to accomplish that.

- All new features must first be discussed in a GitHub issue before starting to implement them. For
  complex features, it can be useful to submit a [formal design document](design/template.md).

- Development happens on a long-lived development branch (`dev.v2` and `dev.v1`).
  Every change going into a development branch is reviewed individually (see below). Release branches may be used
  to support in-the-wild releases of Wormhole. We aim to support at most two release
  branches at the same time. Changes can be cherry-picked from the development branch to release branches, but
  never from release branches to a development branch.
- Releases are first tested on a testnet.

- Commits should be small and have a meaningful commit message. One commit should, roughly, be "one idea" and
  be as atomic as possible. A feature can consist of many such commits.
- Feature flags and interface evolution are better than breaking changes and long-lived feature branches.
- We optimize for reading, not for writing - over its lifetime, code is read much more often than written.
  Small commits, meaningful commit messages and useful comments make it easier to review code and improve the
  quality of code review as well as review turnaround times. It's much easier to spot mistakes in small,
  well-defined changes.

Documentation for the in-the-wild deployments lives in the
[wormhole-networks](https://github.com/certusone/wormhole-networks) repository.

See [DEVELOP.md](./DEVELOP.md) for more information on how to run the development environment.

## Proposing Changes
The Wormhole project understands that contributions come in all shapes and sizes and from both from casual contributions (eg. fix a typo) to advanced contributions (eg. add a new chain integration).  Understanding this, the Wormhole project aims to offer the following paths to make contributions to Wormhole.
### GitHub Pull Request (aka: Casual Contribution)
**Objective:** Allow any casual contribution to the project using the default tooling provided by GitHub.

**Examples of casual contribution**
- Adjustments to documentation
- Adjustments to configuration parameters
- Fixing a script or otherwise non-mission critical piece of code

**Process**

1. Fork the Wormhole project ([GitHub Official Documentation](https://docs.github.com/en/get-started/quickstart/fork-a-repo)).
1. Clone the forked repository ([GitHub Official Documentation](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository#cloning-a-repository))
1. Change directory into the Wormhole repository
    - `cd ./wormhole`
1. Create a new branch
    - `git checkout -b NAME_OF_FEATURE_BRANCH`
1. Make changes to relevant files
1. Add your changes
      - `git add FILE_YOU_CHANGED`
1. Commit your changes
      - `git commit -m "docs: updated contributing process"`
      - *See below for commit message convention guidance*
1. Push your changes to GitHub
      - `git push origin NAME_OF_FEATURE_BRANCH"`
1. On the GitHub PR, Propose 2 core contributors -OR- Wait for 2 core contributors to review and provide feedback.
1. Iterate with the reviewers to ensure that the necessary changes are made and once all changes are made and reviews are complete a core contributor will merge the code into the dev.v2 branch.

### Gerrit Change Set (aka: Advanced Contribution)
**Objective:** Allow advanced contributors, often core contributors, to make major changes to the project using a more advanced review toolchain, allowing for more efficient and iterative change sets and preventing re-reviews of code, which has already been reviewed.

**Examples of Advanced contribution**
- Adding a new chain integration
- Adding a new core feature to the protocol
- Initial draft of a protocol specification/whitepaper
- Any non-config changes to the Guardian/Smart Contracts

**Process**

1. Fork the Wormhole project ([GitHub Official Documentation](https://docs.github.com/en/get-started/quickstart/fork-a-repo))
1. INSERT_STEPS_TO_GET_SETUP_USING_GERRIT_POSSIBLY_ADDING_A_GERRIT_INSTANCE_FOR_WORMHOLE
1. Copy the "Clone with commit-msg hook" command from Gerrit to pull down the repository.
1. Change directory into the Wormhole repository
    - `cd ./wormhole`
1. Create a new branch
    - `git checkout -b NAME_OF_FEATURE_BRANCH`
1. Make changes to relevant files
1. Add your changes
      - `git add FILE_YOU_CHANGED`
1. Commit your changes
      - `git commit -m "docs: updated contributing process"`
      - *See below for commit message convention guidance*
1. Push your changes to Gerrit
      - `git push origin HEAD:refs/for/main"`
1. On the Gerrit Change Set, Propose 2 core contributors -OR- Wait for 2 core contributors to review, provided feedback, and approve your code.
1. Iterate with the reviewers to ensure that the necessary changes are made and once all changes are made and reviews are complete a core contributor will merge the code into the dev.v2 branch.

## Code Review Principles

- **Must** be reviewed by two independant core contributors to Wormhole, with the same level of rigor and attention to detail.
- **Must** follow the commit naming convention below in this document.

## Contributions FAQ

### Can you add \<random blockchain\>?

The answer is... maybe? The following things are needed in order to fully support a chain in Wormhole:

- The Wormhole mainnet is governed by a DAO. Wormhole's design is symmetric - every guardian node needs to run
  a node or light client for every chain supported by Wormhole. This adds up, and the barrier to support new
  chains is pretty high. Your proposal should clearly outline the value proposition of supporting the new chain.
  **Convincing the DAO to run nodes for your chain is the first step in supporting a new chain.**
- The chain needs to support smart contracts capable of verifying 19 individual secp256k1 signatures.

- The smart contract needs to be built and audited. In some cases, existing contracts can be used, like with
  EVM-compatible chains.
- Support for observing the chain needs to be added to guardiand.

- Web wallet integration needs to be built to actually interact with Wormhole.

The hard parts are (1) convincing the DAO to run the nodes, and (2) convincing the core development team to
either build the integration, or work with an external team to build it.

Please do not open a GitHub issue about new networks - this repository is only a reference implementation for
Wormhole, just like go-ethereum is a reference implementation for Ethereum. Instead, reach out to the
[Wormhole Network](https://wormholenetwork.com).

### Do you support \<random blockchain innovation\>?

Probably :-). At its core, Wormhole is a generic attestation mechanism and is not tied to any particular kind
of communication (like transfers). It is likely that you can use the existing Wormhole contracts to build your
own features on top of, without requiring any changes in Wormhole itself.

Please open a GitHub issue outlining your use case, and we can help you build it!

# Pre-Commit checks

Run `./scripts/lint.sh -d format` and `./scripts/lint.sh lint`.

# Commit Naming Convention

When making commits on Wormhole, it's advised to prefix the commit with the component name.

Example Component Names (generally the root folder of the change in the Wormhole repo):
- node
- ethereum
- sdk
- solana

Example Full Commit Text:
- sdk/js-proto*: 0.0.4 version bump
- node: docs for running a spy against mainnet
- node: Fix formatting with go 1.19

Example Full Commits:
- https://github.com/wormhole-foundation/wormhole/commit/5cc2c071572daab876db2fd82e9d16dc4c34aa11
- https://github.com/wormhole-foundation/wormhole/commit/eeb1682fba9530a8cd8755b53639ba3daefeda36

Resources for writing good commit messages:

- https://www.freecodecamp.org/news/how-to-write-better-git-commit-messages/
- https://cbea.ms/git-commit/
- https://reflectoring.io/meaningful-commit-messages/

## IDE Integration

### Golang formatting

You must format your code with `goimports` before submitting.
You can install it with `go install golang.org/x/tools/cmd/goimports@latest` and run it with `goimports -d ./`.
You can enable it in VSCode with the following in your `settings.json`.

```json
  "go.useLanguageServer": true,
  "go.formatTool": "goimports",
  "[go]": {
    "editor.defaultFormatter": "golang.go",
    "editor.formatOnSaveMode": "file",
    "editor.codeActionsOnSave": {
      "source.fixAll": true,
      "source.organizeImports": true
    }
  },
```

# Testing

We believe automated tests ensure the integrity of all Wormhole components. Anyone can verify or extend them transparently and they fit nicely with our software development lifecycle. This ensures Wormhole components operate as expected in both expected and failure cases.

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
- **Cosmwasm Smart Contracts**
  - Tests: `./cosmwasm/test/*`
  - Run: `cd cosmwasm && make test`
- **Algorand Smart Contracts**
  - Tests: `./algorand/test/*`
  - Run: `cd algorand && make test`

The best place to understand how we invoke these tests via GitHub Actions on every commit can be found via `./.github/workflows/*.yml` and the best place to observe the results of these builds can be found via [https://github.com/wormhole-foundation/wormhole/actions](https://github.com/wormhole-foundation/wormhole/actions).
