# 1. ADRs will be the Canonical Source of Truth for architecture decisions

Date: 2024-06-24

## Status

Accepted

## Context

- As WH/SL
- We want a place to memorialize decisions
- Because it helps with context / institutional memory / onboarding

As discussed and agreed to in the Strangelove / Wormhole [project kick-off](https://miro.com/app/board/uXjVK_fZYq0=/?share_link_id=596301298163).

## Decision

To memorialize decisions, we'll use Architecture Decision Records, as [described by Michael Nygard](http://thinkrelevance.com/blog/2011/11/15/documenting-architecture-decisions).

Briefly: ~if~ whenever we make a decision which might reasonably cause a Future Developer (e.g., a new dev, or Six-Months-In-The-Future-Us) to say "Wait—why'd we choose that?", we'll log an ADR to act as a Canonical Source Of Truth, contemporaneously detailing reasoning.

At times, we'll certainly be wrong.

We'll almost certainly backtrack certain ideas.

But—hopefully—we won't bark up the same tree twice.

## Consequences

- You might want some tooling. For a lightweight ADR toolset, see Nat Pryce's [adr-tools](https://github.com/npryce/adr-tools), or `brew install adr-tools` if you're the trusting sort.
- We instantiated w/ the default template (Status, Context, Decision, Consequences). If/when we want to update the ADR format, this gh issue has the rundown: https://github.com/npryce/adr-tools/issues/120.
