/**
 * @name NEAR receipt outcome publication without same-outcome finality proof
 * @description NEAR watcher receipt logs must only be published after the same receipt_outcome block_hash has been proven finalized.
 * @kind problem
 * @problem.severity error
 * @precision high
 * @id wormhole/go/near-finalized-receipt-outcome-before-publication
 * @tags security
 *       external/cwe/cwe-345
 */

import go
import semmle.go.concepts.GeneratedFile

predicate isProductionNearWatcherFile(File f) {
  f.getRelativePath().matches("node/pkg/watchers/near/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate sameVariable(Expr a, Expr b) {
  exists(Entity target |
    a.stripParens().(Ident).refersTo(target) and
    b.stripParens().(Ident).refersTo(target)
  )
}

bindingset[earlier, later]
pragma[inline]
predicate occursBefore(AstNode earlier, AstNode later) {
  exists(string path, int earlierLine, int earlierColumn, int laterLine, int laterColumn |
    earlier.getLocation().hasLocationInfo(path, earlierLine, earlierColumn, _, _) and
    later.getLocation().hasLocationInfo(path, laterLine, laterColumn, _, _) and
    earlier.getEnclosingFunction() = later.getEnclosingFunction() and
    (
      earlierLine < laterLine
      or
      earlierLine = laterLine and earlierColumn < laterColumn
    )
  )
}

bindingset[variable, earlier, later]
predicate variableReassignedBetween(Expr variable, AstNode earlier, AstNode later) {
  exists(
    Assignment assign, Expr lhs, string path, int earlierLine, int earlierColumn, int assignLine,
    int assignColumn, int laterLine, int laterColumn
  |
    lhs = assign.getLhs(_) and
    sameVariable(lhs, variable) and
    assign.getEnclosingFunction() = later.getEnclosingFunction() and
    earlier.getEnclosingFunction() = later.getEnclosingFunction() and
    earlier.getLocation().hasLocationInfo(path, earlierLine, earlierColumn, _, _) and
    assign.getLocation().hasLocationInfo(path, assignLine, assignColumn, _, _) and
    later.getLocation().hasLocationInfo(path, laterLine, laterColumn, _, _) and
    (
      earlierLine < assignLine
      or
      earlierLine = assignLine and earlierColumn < assignColumn
    ) and
    (
      assignLine < laterLine
      or
      assignLine = laterLine and assignColumn < laterColumn
    )
  )
}

predicate isStringLiteral(Expr e, string s) { e.getStringValue() = s }

predicate isGjsonGetCall(CallExpr call, Expr base, string key) {
  call.getCalleeExpr().(SelectorExpr).getSelector().getName() = "Get" and
  (
    base = call.getCalleeExpr().(SelectorExpr).getBase()
    or
    sameVariable(base, call.getCalleeExpr().(SelectorExpr).getBase())
  ) and
  isStringLiteral(call.getArgument(0), key)
}

predicate isProcessWormholeLogCall(CallExpr call, Expr watcherReceiver) {
  isProductionNearWatcherFile(call.getFile()) and
  call.getCalleeName() = "processWormholeLog" and
  call.getTarget() instanceof Method and
  call.getTarget().(Method).getReceiverBaseType().hasQualifiedName(
    "github.com/certusone/wormhole/node/pkg/watchers/near", "Watcher"
  ) and
  exists(SelectorExpr callee |
    callee = call.getCalleeExpr().(SelectorExpr) and
    watcherReceiver = callee.getBase()
  )
}

predicate assignedGjsonGetFor(Expr resultExpr, Expr base, string key, AstNode use, AstNode origin) {
  exists(Assignment assign, Expr lhs, CallExpr getCall |
    lhs = assign.getLhs(0) and
    isGjsonGetCall(getCall, base, key) and
    assign.getRhs(0) = getCall and
    assign.getEnclosingFunction() = use.getEnclosingFunction() and
    occursBefore(assign, use) and
    sameVariable(lhs, resultExpr) and
    not variableReassignedBetween(lhs, assign, use) and
    origin = assign
  )
}

predicate directLoopReadsLogForReceiptOutcome(Expr logExpr, Expr receiptOutcome, AstNode use, AstNode origin) {
  exists(RangeStmt loop, Ident rangeValue, Variable v, CallExpr arrayCall, Expr logs, Expr outcome, AstNode logsOrigin |
    loop.getEnclosingFunction() = use.getEnclosingFunction() and
    loop.getBody() = use.getParent*() and
    rangeValue = loop.getValue() and
    rangeValue.refersTo(v) and
    logExpr.stripParens().(Ident).refersTo(v) and
    loop.getDomain().stripParens() = arrayCall and
    arrayCall.getCalleeExpr().(SelectorExpr).getSelector().getName() = "Array" and
    sameVariable(arrayCall.getCalleeExpr().(SelectorExpr).getBase(), logs) and
    assignedGjsonGetFor(outcome, receiptOutcome, "outcome", use, origin) and
    assignedGjsonGetFor(logs, outcome, "logs", use, logsOrigin)
  )
}

predicate isReceiptDerivedPublication(CallExpr call, Expr receiptOutcome, AstNode origin, Expr watcherReceiver) {
  isProcessWormholeLogCall(call, watcherReceiver) and
  directLoopReadsLogForReceiptOutcome(call.getArgument(5), receiptOutcome, call, origin)
}

predicate isFinalizerIsFinalizedCall(CallExpr call, Expr watcherReceiver) {
  exists(Method method |
    isProductionNearWatcherFile(call.getFile()) and
    call.getCalleeExpr().(SelectorExpr).getSelector().refersTo(method) and
    method.getName() = "isFinalized" and
    method.getReceiverBaseType().hasQualifiedName(
      "github.com/certusone/wormhole/node/pkg/watchers/near", "Finalizer"
    ) and
    call.getNumArgument() = 3 and
    exists(SelectorExpr callee, SelectorExpr finalizer, Field field |
      callee = call.getCalleeExpr().(SelectorExpr) and
      finalizer = callee.getBase().(SelectorExpr) and
      finalizer.refersTo(field) and
      field.getName() = "finalizer" and
      sameVariable(finalizer.getBase(), watcherReceiver)
    )
  )
}

predicate finalityCallUsesBlockHashFor(CallExpr proofCall, Expr receiptOutcome, AstNode hashOrigin) {
  exists(Assignment assign, Expr lhs, CallExpr getCall, CallExpr stringCall |
    assign.getEnclosingFunction() = proofCall.getEnclosingFunction() and
    lhs = assign.getLhs(0) and
    assign.getRhs(0) = getCall and
    isGjsonGetCall(getCall, receiptOutcome, "block_hash") and
    proofCall.getArgument(2).stripParens() = stringCall and
    stringCall.getCalleeExpr().(SelectorExpr).getSelector().getName() = "String" and
    sameVariable(stringCall.getCalleeExpr().(SelectorExpr).getBase(), lhs) and
    occursBefore(assign, proofCall) and
    not variableReassignedBetween(lhs, assign, proofCall) and
    hashOrigin = assign
  )
}

predicate boolResultOfFinalityCall(Expr boolExpr, CallExpr proofCall) {
  exists(Assignment assign, Expr lhs |
    assign.getRhs(0) = proofCall and
    lhs = assign.getLhs(1) and
    sameVariable(lhs, boolExpr)
  )
}

predicate headerResultOfFinalityCall(Expr headerExpr, CallExpr proofCall) {
  exists(Assignment assign, Expr lhs |
    assign.getRhs(0) = proofCall and
    lhs = assign.getLhs(0) and
    sameVariable(lhs, headerExpr)
  )
}

predicate finalityHeaderPassedToPublication(CallExpr proofCall, CallExpr sink) {
  headerResultOfFinalityCall(sink.getArgument(3), proofCall) and
  not variableReassignedBetween(sink.getArgument(3), proofCall, sink)
}

predicate finalityBooleanGuardDominates(Expr boolExpr, AstNode sink, boolean outcome) {
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node sinkNode |
    outcome = true and
    sinkNode.isFirstNodeOf(sink) and
    guard.ensures(DataFlow::exprNode(boolExpr), true) and
    guard.dominates(sinkNode.getBasicBlock())
  )
}

predicate hasSameOutcomeFinalityProof(
  Expr receiptOutcome, AstNode observationOrigin, AstNode sink, Expr watcherReceiver
) {
  exists(CallExpr proofCall, Expr boolExpr, AstNode hashOrigin |
    proofCall.getEnclosingFunction() = sink.getEnclosingFunction() and
    observationOrigin.getEnclosingFunction() = sink.getEnclosingFunction() and
    isFinalizerIsFinalizedCall(proofCall, watcherReceiver) and
    finalityCallUsesBlockHashFor(proofCall, receiptOutcome, hashOrigin) and
    occursBefore(proofCall, sink) and
    occursBefore(hashOrigin, proofCall) and
    boolResultOfFinalityCall(boolExpr, proofCall) and
    finalityBooleanGuardDominates(boolExpr, sink, true) and
    finalityHeaderPassedToPublication(proofCall, sink) and
    not variableReassignedBetween(boolExpr, proofCall, sink) and
    not variableReassignedBetween(receiptOutcome, proofCall, sink) and
    not variableReassignedBetween(receiptOutcome, observationOrigin, proofCall)
  )
}

from CallExpr sink, Expr receiptOutcome, AstNode observationOrigin, Expr watcherReceiver
where
  isReceiptDerivedPublication(sink, receiptOutcome, observationOrigin, watcherReceiver) and
  not hasSameOutcomeFinalityProof(receiptOutcome, observationOrigin, sink, watcherReceiver)
select sink,
  "NEAR receipt_outcome logs must not be published before proving that the same receipt_outcome.block_hash is finalized."
