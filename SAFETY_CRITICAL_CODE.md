# Safety-Critical Code Standards

This document outlines coding principles for Wormhole's mission-critical infrastructure. These are aspirational standards representing where we're heading through continuous improvement and preventing regression.

Wormhole secures billions of dollars in assets. A single bug can be catastrophic. Code quality is code security.

These principles apply to all Wormhole code, with special emphasis on Guardian nodes and core protocol implementation. They favor **boredom over beauty**, **predictability over cleverness**, and **testability over conciseness**.

---

## Core Principles

### Parse, Don't Verify

Transform untrusted data into constrained types rather than checking and discarding the proof. Once parsed, the type system enforces your constraints automatically. Invalid states become unrepresentable. Use the compiler as your ally—let it prevent misuse at compile time rather than catching it at runtime.

### Make Invalid States Unrepresentable

Design data structures so that illegal states cannot be constructed. Use sum types (enums/unions) instead of boolean flags. Use bounded types instead of magic sentinel values. If a value can only exist in valid states, you cannot accidentally create an invalid one. The compiler prevents entire classes of bugs.

### Document Assumptions and Invariants

Every function must explicitly document its preconditions, postconditions, and invariants. Assert all preconditions at function entry. Assert all postconditions before return. Assert invariants at boundaries where they matter. Comments should explain *why*, not *what*—the code already shows what.

Use `REQUIRES:` and `ENSURES:` in function documentation to specify preconditions and postconditions. Add inline comments explaining non-obvious design decisions.

### Redundant Safety Over Performance

When in doubt, choose safety. Redundant checks are insurance, not waste. They catch bugs that "should be impossible" and protect against future refactoring.

Add checks at every boundary, even if an earlier check "should" have caught the error. For every property you want to enforce, find at least two different code paths where you can assert it—before writing to disk and after reading from disk, before serialization and after deserialization.

Remove checks only after profiling proves they're a bottleneck, after replacing them with cheaper equivalents, and after documenting the removal with explicit justification.

### Explicit Constraint Propagation

Constraints from dependencies must be explicitly enforced, not implicitly assumed. If you depend on a library that requires 32-byte hashes, assert that at the call site. If your database requires 4K-aligned pages, enforce that with compile-time or runtime checks.

Document external constraints in comments explaining *why* they exist. Make implicit dependencies explicit and searchable. When requirements change, you'll know exactly where to look.

---

## Control Flow and Complexity

### No Recursion

Recursion is forbidden. Recursion makes stack depth unpredictable, which means unbounded memory usage and potential crashes. Recursive code is less friendly to static analysis tools.
Iterative code makes stack usage explicit, visible, and testable.

### Bound Everything

Every loop, queue, retry mechanism, timeout, and I/O operation must have a statically-known upper bound. Unbounded operations are time bombs. They will fail in production.

- Loops must have a maximum iteration count
- Queues must have a maximum capacity and reject additions when full
- Retries must have a maximum attempt count
- Timeouts must be explicit
- Network reads must have a maximum size (prevents DoS via huge requests)
- Disk reads must have a maximum size (prevents resource exhaustion)

Event loops that intentionally run forever must assert this explicitly in comments.

### Simplify Conditionals

Split compound boolean expressions into simple, nested conditions. Avoid complex `if` conditions with multiple `&&` or `||` operators. Each branch should be obvious. Each case should be explicitly handled.

State invariants positively rather than negatively. `if index < length` is clearer than `if !(index >= length)`.

### Centralize Control Flow

When splitting large functions, keep all branching logic (if/switch/match) in the parent function. Push child functions toward pure computation without branching. This separates decision-making from execution and makes both easier to understand and test.

### Function Length Limit

Target: 70 lines per function. If you can't see the entire function without scrolling, you can't fully understand it. Long functions should be split by extracting pure logic into helpers while keeping control flow centralized.

---

## Memory and Resource Management

### Bounded Memory Usage

Prefer static allocation where possible. Pre-allocate buffers at startup when you can predict maximum usage. Where dynamic allocation is necessary, enforce strict bounds and monitoring.

**Collections must not grow unbounded.** Every map, slice, array, or buffer must have:
- A maximum capacity enforced at runtime
- Explicit eviction or cleanup policies when limits are reached
- Alerting when approaching capacity thresholds
- Clear behavior when full (reject, evict oldest, fail explicitly)

Use ring buffers, bounded queues, and object pools. Establish hard limits on cache sizes, connection pools, and pending operations. Memory usage should be predictable and observable—never allow silent, unbounded growth.

### Minimize Variable Scope

Declare variables at the smallest possible scope. Fewer variables in scope means fewer ways to misuse them. Calculate values when needed, not before. Minimize the gap between where data is validated and where it's used (place-of-check to place-of-use distance).

---

## Assertions and Validation

### Assertion Density

Minimum: 2 assertions per non-trivial function. Assertions turn correctness bugs (silent corruption) into liveness bugs (crashes), which are infinitely preferable.

Assert preconditions at function entry. Assert postconditions before return. Assert invariants at boundaries where they matter.

### Assert Positive and Negative Space

Assert what you expect AND what you don't expect. The boundary between valid and invalid data is where bugs hide.

When iterating over a collection can change control flow, ensure that the collection is non-empty.

### Split Compound Assertions

Don't combine multiple checks into one assertion. Split them so that failures give precise information about what went wrong.

### Use Compile-Time Assertions

Assert at compile time whenever possible. Compile-time assertions check design integrity before the program runs. Use them for constant relationships, type sizes, and configuration invariants.

---

## Testing Requirements

### Design for Testability

If code is hard to test, the design is wrong. Testability is a design constraint, not an afterthought.

Separate I/O from logic. Write pure functions that can be tested without touching the network, disk, or database. Keep I/O in thin wrapper functions that are simple enough to be obviously correct.

Depend on abstractions (interfaces/traits), not concrete implementations. Pass data into functions rather than services when possible. If a function is hard to test, split it until the logic is pure and the I/O is trivial.

### Table-Driven Tests

Table-driven tests are mandatory for testing multiple cases. They're more extensible, easier to review, and less error-prone than copy-pasted test functions.

Define test cases as data structures with inputs and expected outputs. Loop over the cases. Adding new test cases becomes adding data, not duplicating code.

### Negative Tests Are Mandatory

Every success case must have a corresponding failure case. If you test that valid input succeeds, you must test that invalid input fails.

Test edges:
- Zero values and maximum values
- Off-by-one boundaries
- Empty collections and null/none values
- Overflows and underflows
- All error paths

### Error Handling Tests

Every error path must be tested. Error paths are not edge cases—they're the most important cases.

Test that errors propagate correctly. Test that cleanup happens on error. Test that state remains consistent despite errors.

---

## Code Style and Standards

### Explicit Types

Use explicitly-sized integer types (`i32`, `u64`, `int32`, `uint64`) rather than architecture-dependent types (`int`, `usize`, `size_t`). Explicit sizes prevent subtle portability bugs and make data layout obvious.

### Strict Linting

Enable all compiler warnings. Treat warnings as errors. Fix them immediately. Compiler warnings are free bug reports.

Configure the strictest linting available for your language. If a linter complains, it's probably right. Exceptions must be justified with inline comments at the violation site.

### Naming Conventions

- Use descriptive names without abbreviations
- Add units to variable names (put units last: `timeout_ms` not `ms_timeout`)
- Order qualifiers by significance (most important first: `config_guardian_set_size`)
- Use proper capitalization for acronyms and common terms

### Express Intent Explicitly

Use explicit operations that show you've considered edge cases. For division, use functions that clarify rounding behavior (exact division vs floor vs ceiling). For library calls, specify options explicitly rather than relying on defaults—defaults can change.

### TODO Comments Must Link Issues

In-line TODO comments are only allowed if they include a full GitHub issue URL. TODOs without trackable issues don't get prioritized or fixed—they rot.

The full URL makes the TODO trackable (one click to full context), unambiguous (no confusion about which repo), accountable (someone owns the issue), and actionable (clear what needs to be done and why).

If it's worth noting, it's worth tracking in an issue. If it's not worth an issue, fix it now instead of leaving a TODO.

---

## Dependencies and External Code

### Minimize Dependencies

Every dependency is a liability that increases attack surface, introduces supply chain risk, adds compilation time, and brings transitive dependencies.

Before adding a dependency, ask:
- Can we implement this in under 200 lines?
- Is this dependency actively maintained?
- Has it been security audited?
- How many transitive dependencies does it pull in?

### Pin Exact Versions

Lock all dependencies to exact versions. Verify checksums. Update dependencies deliberately in isolated changes with review and testing, not as side effects of feature work.

### Never Rely on Defaults

Explicitly specify all library options rather than relying on defaults. Defaults can change between versions. Explicit configuration prevents surprises and documents your intent.

---

## Security Practices

**Defense in Depth:** Multiple independent layers of validation are better than one. Validate early and often. Check at boundaries. Trust nothing from external sources.

**Fail Fast and Loud:** Detect violations as early as possible. Crash rather than continue with corrupt state. Make failures visible and debuggable.

**Least Privilege:** Code should have access only to the resources it needs. Minimize scope and capabilities. Constrain inputs and validate outputs.

---

## The Path Forward

These standards are strict because the stakes are high. Apply them to new code immediately. Gradually refactor existing code to meet these standards. Use CI to enforce what can be automated. Use code review to enforce the rest.

Remember:
- Boredom beats beauty
- Simplicity takes work
- Constraints breed creativity
- Prevention is cheaper than debugging

Write code that's boring. Write code that works. Write code you'll be proud of in production.

---

## Resources

These principles draw from decades of experience in safety-critical systems:

- [NASA's Power of Ten](https://spinroot.com/gerard/pdf/P10.pdf) — Rules for developing safety-critical code
- [TigerBeetle's TIGER STYLE](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/TIGER_STYLE.md) — Coding style for a financial database
- [Boredom Over Beauty](https://blog.asymmetric.re/boredom-over-beauty-why-code-quality-is-code-security/) — Why code quality is code security
- [Parse, Don't Validate](https://lexi-lambda.github.io/blog/2019/11/05/parse-don-t-validate/) — Type-driven design principles
