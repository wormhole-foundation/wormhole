# Contributing

Wormhole is an open-source project licensed under the permissive Apache 2 license. Contributions are greatly
appreciated and will be reviewed swiftly.

Wormhole is a mission-critical, high-stakes project. We optimize for quality over quantity. Design processes
and code reviews are our most important tools to accomplish that.

- All new features must first be discussed in a GitHub issue before starting to implement them. For
  complex features, it can be useful to submit a [formal design document](design/template.md).

- Development happens on a long-lived development branch (usually `main` or `dev.<x>` for larger changes).
  Every change going into a development branch is reviewed individually (see below). Release branches branched
  from `main` are used to support in-the-wild releases of Wormhole. We aim to support at most two release
  branches at the same time. Changes can be cherry-picked from the development branch to release branches, but
  never from release branches to a development branch.
  
- Releases are first tested on a testnet. This involves coordination with the mainnet DAO running the nodes.

- Commits should be small and have a meaningful commit message. One commit should, roughly, be "one idea" and
  be as atomic as possible. A feature can consist of many such commits.
  
- Feature flags and interface evolution are better than breaking changes and long-lived feature branches.
  
- We optimize for reading, not for writing - over its lifetime, code is read much more often than written.
  Small commits, meaningful commit messages and useful comments make it easier to review code and improve the
  quality of code review as well as review turnaround times. It's much easier to spot mistakes in small,
  well-defined changes.

Documentation for the in-the-wild deployments lives in the
[wormhole-networks](https://github.com/certusone/wormhole-networks) repository.

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

You should first open a GitHub issue with more details.

<!--
TODO: how to contact the DAO? most of the communication today happens in a Telegram group, we should move this
somewhere better-suited for public inquiries (Discourse forum?)
-->

### Do you support \<random blockchain innovation\>?

Probably :-). At its core, Wormhole is a generic attestation mechanism and is not tied to any particular kind
of communication (like transfers). It is likely that you can use the existing Wormhole contracts to build your
own features on top of, without requiring any changes in Wormhole itself.

Please open a GitHub issue outlining your use case, and we can help you build it!

## Submit change for review

Certus One uses **Gerrit** for code review on [**forge.certus.one**](https://forge.certus.one). Gerrit has the
advantage of dealing with a stack of individual commits, rather than reviewing an entire branch. This makes it
much easier to review large features by breaking them down into smaller pieces, and puts a large emphasis on
clean commits with meaningful commit messages. This workflow helps us write better software.

We also accept contributions via **GitHub PRs**, but we strongly recommend to give Gerrit a try. Gerrit has
somewhat of a learning curve, but offers a much nicer experience (think of it as Vim vs. Notepad).

The GitHub repository is a mirror of the Gerrit repository. GitHub has a global CDN for Git, so if you plan
to clone the Wormhole repo a lot in an automated fashion, please clone it from GitHub.

### Why Gerrit?

With GitHub, if you want to submit three changes A, B and C to be reviewed (in that order), you have two
choices:

- Submit a single PR with carefully rebased commits, and ask the reviewer to actually look at your
  carefully-written commit messages and review each commit individually. However, this is not well-supported
  by the UI, approval can only be given for the whole stack, and rebasing/adding commits breaks it altogether.
  It also doesn't work with the squash merge policy used by many projects.
  
- Submit three individual PRs with different bases. This allows you to approve and merge each change
  individually, but requires you to manually rebase multiple branches on top of each other, which is annoying.

By making it hard to break changes up into smaller pieces, GitHub encourages large, hard-to-review changes.
With Gerrit, the opposite is true - it's **trivial to submit a stack of changes**. You can just put your
changes A, B and C on a single branch:

    C  <-- HEAD
    ↑
    B
    ↑
    A
    ↑
    O <-- origin/main, main
    ↑
    …

... and submit all three using a single `git push origin HEAD:refs/for/main`. Gerrit will create a review
request for A, B and C, and it understands the relation chain between them. C can only be merged after B and
C, and merging C will automatically merge B and C as well.

This means that A can be reviewed, approved and merged before B and C are done. Other team members can then
start building on A and avoid a "big scary merge". This workflow is often called **trunk-based development**.

Other advantages of Gerrit include:

- The ability to **compare different versions of a change**, with inline comments shown in their original place.
  This is very useful when re-reviewing a change.
- Keeping inline comments across rebases (!).
- Very responsive user interface that can be fully driven using keyboard shortcuts.
  GitHub can be slow - opening a PR and showing the diff often takes multiple seconds.
- A view that shows an overview of open comments, their status and a small code snippet.
- Comments can be attached to a selection, not just entire lines. 
  Multiple threads can be attached to the same line.
- The "**attention set**" mechanism with a fine-grained state machine on who needs to take action,
  which avoids sending non-actionable email notifications!
- We run our own infrastructure. Yay decentralization!

### Quickstart

You can log into Gerrit using your Google account. **If you're contributing on behalf of a company, make
sure that your Git email address reflects your affiliation!**

First, add your SSH keys to Gerrit in your [profile settings](https://forge.certus.one/settings/#SSHKeys).
Alternatively, you can generate an HTTP Password and store it in your [Git credentials store of
choice](https://git-scm.com/book/en/v2/Git-Tools-Credential-Storage) - this is particularly useful for
development hosts or corporate environments that can't use SSH or access your key.

**Clone the repo from Gerrit** if you haven't done so already by going to the [repository
page](https://forge.certus.one/admin/repos/wormhole) and using the *"Clone with commit-msg hook"* command. If
you have an existing GitHub checkout you want to convert, you can simply set a new remote and you'll be
prompted to install the hook the first time you push to Gerrit:

    git remote set-url origin ssh://<gerrit-username>@[...]

Then, just commit to a local branch. Every local commit becomes one code review request. Multiple commits on
the same branch will be submitted as a stack (see above). Once you're done, push to the special ref that
creates your code reviews:

    git push origin HEAD:refs/for/main

(replace `main` by a different development branch, where applicable)

That's it! No special tooling needed. You can now go look at your commits in the web UI and add reviewers. If
you want less typing, take a look at these:

- There's an excellent [IntelliJ plugin](https://plugins.jetbrains.com/plugin/7272-gerrit) that allows you to clone,
  check out and even review CLs from inside your IDE.

- The Go project's [git-codereview](https://pkg.go.dev/golang.org/x/review/git-codereview) CLI utility.
