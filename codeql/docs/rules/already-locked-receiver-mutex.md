# AlreadyLocked Receiver Mutex

Call documented Wormhole `AlreadyLocked` helpers only while holding the required write lock on the same receiver instance.

## Why This Matters

Methods named `AlreadyLocked` encode a caller-side locking precondition. They are intentionally written without acquiring their own mutex because callers are expected to already hold the receiver's write lock while mutating shared Accountant, Governor, or CCQ state. Calling one of these helpers without the exact receiver lock can race concurrent state updates; calling it after an intervening non-deferred `Unlock` can also violate the precondition even when an earlier lock appears in the same function.

## Examples

### Violation

```go
func publish(a *Accountant, transfer *Transfer) error {
	return a.publishTransferAlreadyLocked(transfer)
}
```

```go
func publishAfterUnlock(a *Accountant, transfer *Transfer) error {
	a.pendingTransfersLock.Lock()
	a.pendingTransfersLock.Unlock()
	return a.publishTransferAlreadyLocked(transfer)
}
```

```go
func publishWrongReceiver(a, other *Accountant, transfer *Transfer) error {
	a.pendingTransfersLock.Lock()
	defer a.pendingTransfersLock.Unlock()
	return other.publishTransferAlreadyLocked(transfer)
}
```

### Fix

```go
func publish(a *Accountant, transfer *Transfer) error {
	a.pendingTransfersLock.Lock()
	defer a.pendingTransfersLock.Unlock()
	return a.publishTransferAlreadyLocked(transfer)
}
```

```go
func publishAfterRelock(a *Accountant, transfer *Transfer) error {
	a.pendingTransfersLock.Lock()
	a.pendingTransfersLock.Unlock()
	a.pendingTransfersLock.Lock()
	defer a.pendingTransfersLock.Unlock()
	return a.publishTransferAlreadyLocked(transfer)
}
```

## What The Rule Checks

The rule reports production Go calls under `node/pkg/accountant/`, `node/pkg/governor/`, and `node/cmd/ccq/` when a documented `AlreadyLocked` helper is called without a proven same-function write-lock precondition on the same receiver. Direct `go` and `defer` helper invocations are always reported because execution is not guaranteed to occur while the syntactic lock remains held.

Exact method-to-mutex mappings are:

| Package | Receiver type | AlreadyLocked method | Required receiver mutex |
| --- | --- | --- | --- |
| `accountant` | `*Accountant` | `publishTransferAlreadyLocked` | `pendingTransfersLock` |
| `accountant` | `*Accountant` | `addPendingTransferAlreadyLocked` | `pendingTransfersLock` |
| `accountant` | `*Accountant` | `deletePendingTransferAlreadyLocked` | `pendingTransfersLock` |
| `governor` | `*ChainGovernor` | `parseMsgAlreadyLocked` | `mutex` |
| `governor` | `*ChainGovernor` | `loadFromDBAlreadyLocked` | `mutex` |
| `ccq` | `*PendingResponses` | `updateMetricsAlreadyLocked` | `mu` |

A call is considered protected when an ordinary synchronous same-function call to `receiver.<mutex>.Lock()` occurs before the `AlreadyLocked` call and the lock's basic block dominates the helper call. Deferred and goroutine-launched lock calls do not count. The receiver must be the same unmodified receiver value: locking `a.pendingTransfersLock` does not protect `other.publishTransferAlreadyLocked(...)`, and rebinding `a` or a locked alias before the helper invalidates the proof.

The rule treats non-deferred unlocks as lock-precondition breakers. A later ordinary synchronous same-branch relock can re-establish the precondition only when no textually intervening `break`, `continue`, or `goto` can bypass it. This is a conservative structured-control approximation rather than a complete path-sensitive lock-state analysis.

Nested `AlreadyLocked` calls inherit the enclosing helper's precondition in a bounded way. A call from inside one of the documented `AlreadyLocked` methods is allowed when it is made on the enclosing method receiver, or on a one-hop local alias assigned from that still-unmodified receiver. The inheritance is rejected if the receiver parameter is reassigned before the nested call, or if the alias is reassigned after being initialized from the receiver.

The rule recognizes ordered same-function receiver aliases and local value equivalence for direct mutex accesses such as:

```go
func publish(a *Accountant, transfer *Transfer) error {
	acct := a
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()
	return a.publishTransferAlreadyLocked(transfer)
}
```

It also handles reassignment-sensitive cases conservatively for nested precondition inheritance:

```go
func (a *Accountant) publishTransferAlreadyLocked(transfer *Transfer) error {
	acct := a
	acct = otherAccountant()
	return acct.addPendingTransferAlreadyLocked(transfer) // not covered by inherited precondition
}
```

## Exclusions

The rule ignores tests, generated files, files outside `node/pkg/accountant/`, `node/pkg/governor/`, and `node/cmd/ccq/`, unrelated method names, unrelated receiver types, read locks, deferred unlocks that run after the helper call, and locks on different receiver instances or different mutex fields.

## Limitations

This is a bounded receiver-lock rule, not a full interprocedural lock-state proof. It does not model lock acquisition in callers, wrappers, closures, goroutines, function values, interfaces, reflection, unsafe pointer manipulation, container-stored receivers, or deep alias chains. It requires a same-function dominating `Lock` unless the call is a bounded nested call inside a documented `AlreadyLocked` method. It also does not prove that the protected helper itself avoids unlocking internally; it only checks call-site preconditions for the documented helper set.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/already-locked-receiver-mutex.md): records the exact method-to-mutex policy and lifecycle evidence.
- [Rule query](../../src/already-locked-receiver-mutex.ql): defines the production scope, exact method-to-mutex mappings, same-receiver lock matching, unlock/relock handling, and bounded nested precondition inheritance.
- [Go `sync.Mutex` documentation](https://pkg.go.dev/sync#Mutex): describes lock and unlock semantics for Go mutexes.
- [Wormhole Accountant package](https://github.com/wormhole-foundation/wormhole/tree/main/node/pkg/accountant), [Governor package](https://github.com/wormhole-foundation/wormhole/tree/main/node/pkg/governor), and [CCQ command package](https://github.com/wormhole-foundation/wormhole/tree/main/node/cmd/ccq): source areas covered by this rule.

## Maintainer Notes

The CodeQL ID is `wormhole/go/already-locked-receiver-mutex`. Keep the method-to-mutex table synchronized with `isAlreadyLockedMethod` and `isAlreadyLockedCall` in the query. If new `AlreadyLocked` helpers are added, add explicit method-to-mutex mappings and fixtures before broadening by naming convention alone. If alias or interprocedural support is expanded, preserve the same-receiver requirement and add reassignment, unlock-without-relock, and nested-helper regression cases.
