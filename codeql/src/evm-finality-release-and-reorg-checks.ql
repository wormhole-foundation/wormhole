/**
 * @name EVM pending publication lacks finality or reorg checks
 * @description EVM watcher pending-message release must wait for the effective consistency level and height threshold, then refetch and prove the same receipt transaction/block before verifyAndPublish.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/evm-finality-release-and-reorg-checks
 * @tags security
 *       external/cwe/cwe-345
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.controlflow.ControlFlowGraph

predicate isProductionEvmWatcherFile(File f) {
  f.getRelativePath().matches("node/pkg/watchers/evm/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

bindingset[earlier]
bindingset[later]
pragma[inline]
predicate occursBefore(AstNode earlier, AstNode later) {
  exists(string path, int earlierLine, int earlierColumn, int laterLine, int laterColumn |
    earlier.getLocation().hasLocationInfo(path, earlierLine, earlierColumn, _, _) and
    later.getLocation().hasLocationInfo(path, laterLine, laterColumn, _, _) and
    earlier.getEnclosingFunction() = later.getEnclosingFunction() and
    (earlierLine < laterLine or earlierLine = laterLine and earlierColumn < laterColumn)
  )
}

predicate sameVariable(Expr a, Expr b) {
  exists(Entity target |
    a.stripParens().(Ident).refersTo(target) and
    b.stripParens().(Ident).refersTo(target)
  )
}

predicate sameValue(Expr a, Expr b) {
  sameVariable(a, b)
}

predicate selectorNamed(Expr e, Expr base, string name) {
  exists(SelectorExpr sel |
    e.stripParens() = sel and
    sel.getSelector().getName() = name and
    sameVariable(sel.getBase(), base)
  )
}

predicate pendingMessageExpr(Expr msg, Expr pending) {
  selectorNamed(msg, pending, "message")
}

predicate pendingEffectiveClExpr(Expr e, Expr pending) {
  selectorNamed(e, pending, "effectiveCL")
}

predicate pendingHeightExpr(Expr e, Expr pending) {
  selectorNamed(e, pending, "height")
}

predicate pendingAdditionalBlocksExpr(Expr e, Expr pending) {
  selectorNamed(e, pending, "additionalBlocks")
}

predicate blockNumberExpr(Expr e, Expr event) {
  selectorNamed(e, event, "Number")
}

predicate blockNumberSourceExpr(Expr e, Expr event) {
  blockNumberExpr(e, event)
  or exists(CallExpr call, SelectorExpr callee |
    e.stripParens() = call and
    callee = call.getCalleeExpr().(SelectorExpr) and
    callee.getSelector().getName() = "Uint64" and
    blockNumberExpr(callee.getBase(), event)
  )
}

predicate pendingMessageTxIdExpr(Expr e, Expr pending) {
  exists(SelectorExpr txSel, SelectorExpr msgSel |
    e.stripParens() = txSel and
    txSel.getSelector().getName() = "TxID" and
    txSel.getBase().stripParens() = msgSel and
    msgSel.getSelector().getName() = "message" and
    sameVariable(msgSel.getBase(), pending)
  )
}

predicate receiptTxHashExpr(Expr e, Expr receipt) {
  selectorNamed(e, receipt, "TxHash")
}

predicate receiptBlockHashExpr(Expr e, Expr receipt) {
  selectorNamed(e, receipt, "BlockHash")
}

predicate keyBlockHashExpr(Expr e, Expr key) {
  exists(SelectorExpr sel |
    e.stripParens() = sel and
    sel.getSelector().getName() = "BlockHash"
    and sameValue(sel.getBase(), key)
  )
}

predicate bytesToHashOfPendingTxId(Expr e, Expr pending) {
  exists(CallExpr call |
    e.stripParens() = call and
    call.getCalleeExpr().(SelectorExpr).getSelector().getName() = "BytesToHash" and
    pendingMessageTxIdExpr(call.getArgument(0), pending)
  )
}

predicate localAssignedFromBytesToHashOfPendingTxId(Expr e, Expr pending) {
  exists(Assignment assign, Expr lhs, Expr source |
    lhs = assign.getLhs(_) and
    bytesToHashOfPendingTxId(source, pending) and
    source = assign.getRhs(_) and
    sameVariable(lhs, e) and
    occursBefore(assign, e)
  )
}

predicate txHashOfPendingTxId(Expr e, Expr pending) {
  pendingMessageTxIdExpr(e, pending)
  or bytesToHashOfPendingTxId(e, pending)
  or localAssignedFromBytesToHashOfPendingTxId(e, pending)
}

predicate assignmentReceivesTupleElement(Assignment assign, CallExpr call, int index, Expr lhs) {
  assign.getRhs(0) = call and
  lhs = assign.getLhs(index)
}

predicate pendingMapOfReceiver(Expr domain, Expr receiver) {
  exists(SelectorExpr sel |
    domain.stripParens() = sel and
    sel.getSelector().getName() = "pending" and
    sameVariable(sel.getBase(), receiver)
  )
}

predicate pendingRangeKey(Expr pending, Expr key, Expr receiver) {
  exists(RangeStmt loop, Ident rangeKey, Ident rangeValue, Variable keyVar, Variable valueVar |
    rangeKey = loop.getKey() and
    rangeValue = loop.getValue() and
    pendingMapOfReceiver(loop.getDomain(), receiver) and
    rangeKey.refersTo(keyVar) and
    rangeValue.refersTo(valueVar) and
    pending.(Ident).refersTo(valueVar) and
    key.(Ident).refersTo(keyVar)
  )
}

predicate pendingRangeKey(Expr pending, Expr key) {
  exists(RangeStmt loop, Ident rangeKey, Ident rangeValue, Variable keyVar, Variable valueVar |
    rangeKey = loop.getKey() and
    rangeValue = loop.getValue() and
    rangeKey.refersTo(keyVar) and
    rangeValue.refersTo(valueVar) and
    pending.(Ident).refersTo(valueVar) and
    key.(Ident).refersTo(keyVar)
  )
}

predicate conditionContains(Expr condition, Expr child) {
  child = condition or child = condition.getAChild*()
}

predicate consistencyLevelConstant(Expr e, string name) {
  e.stripParens().(Ident).getName() = name
  or e.stripParens().(SelectorExpr).getSelector().getName() = name
}

predicate variableAssignedFrom(Expr variableUse, Expr rhs, Assignment assign) {
  exists(Expr lhs |
    lhs = assign.getLhs(_) and
    rhs = assign.getRhs(_) and
    sameVariable(lhs, variableUse)
  )
}

predicate currentConsistencyLevelExpr(Expr e, AstNode before) {
  exists(Variable v, Assignment finalizedAssign, Assignment safeAssign |
    e.stripParens().(Ident).refersTo(v) and
    e.stripParens().(Ident).getName() = "thisConsistencyLevel" and
    variableAssignedFrom(e, _, finalizedAssign) and
    consistencyLevelConstant(finalizedAssign.getRhs(_), "ConsistencyLevelFinalized") and
    variableAssignedFrom(e, _, safeAssign) and
    consistencyLevelConstant(safeAssign.getRhs(_), "ConsistencyLevelSafe") and
    finalizedAssign.getEnclosingFunction() = before.getEnclosingFunction() and
    safeAssign.getEnclosingFunction() = before.getEnclosingFunction() and
    occursBefore(finalizedAssign, before) and
    occursBefore(safeAssign, before)
  )
}

predicate conditionRejectsConsistencyMismatch(Expr condition, Expr pending, AstNode before, Expr proof) {
  exists(NotExpr notExpr, CallExpr call |
    conditionContains(condition, notExpr) and
    proof = notExpr and
    notExpr.getOperand().stripParens() = call and
    call.getCalleeExpr().(Ident).getName() = "consistencyLevelMatches" and
    currentConsistencyLevelExpr(call.getArgument(0), before) and
    pendingEffectiveClExpr(call.getArgument(1), pending)
  )
}

predicate pendingReleaseHeightThreshold(Expr e, Expr pending) {
  exists(AddExpr added |
    e.stripParens() = added and
    pendingHeightExpr(added.getAnOperand(), pending) and
    pendingAdditionalBlocksExpr(added.getAnOperand(), pending)
  )
}

predicate currentBlockNumberExpr(Expr e, AstNode before) {
  exists(Assignment assign, Expr event |
    e.stripParens().(Ident).getName() = "blockNumberU" and
    variableAssignedFrom(e, _, assign) and
    blockNumberSourceExpr(assign.getRhs(_), event) and
    occursBefore(assign, before)
  )
}

predicate conditionRejectsHeightNotReached(Expr condition, Expr pending, AstNode before, Expr proof) {
  exists(RelationalComparisonExpr cmp |
    conditionContains(condition, cmp) and
    proof = cmp and
    cmp.isStrict() and
    pendingReleaseHeightThreshold(cmp.getGreaterOperand(), pending) and
    currentBlockNumberExpr(cmp.getLesserOperand(), before)
  )
}

predicate receiptFetchForPending(CallExpr fetch) {
  fetch.getCalleeExpr().(SelectorExpr).getSelector().getName() = "TransactionReceipt"
}

predicate receiptFetchForSink(CallExpr sink, Expr pending, Expr receipt, Expr err, CallExpr fetch) {
  exists(Assignment assign, Expr fetchedReceipt |
    receiptFetchForPending(fetch) and
    assignmentReceivesTupleElement(assign, fetch, 0, fetchedReceipt) and
    assignmentReceivesTupleElement(assign, fetch, 1, err) and
    not err.(Ident).getName() = "_" and
    txHashOfPendingTxId(fetch.getArgument(1), pending) and
    sameValue(fetchedReceipt, receipt) and
    occursBefore(fetch, sink)
  )
}

predicate notFoundErrorSentinel(Expr e) {
  e.(SelectorExpr).getSelector().getName() = "ErrNoResult"
  or e.(SelectorExpr).getSelector().getName() = "NotFound"
}

predicate conditionContainsNotFoundCheck(Expr condition, Expr err, Expr proof) {
  exists(CallExpr call, Expr checkedErr, SelectorExpr callee |
    conditionContains(condition, call) and
    proof = call and
    callee = call.getCalleeExpr().(SelectorExpr) and
    callee.getSelector().getName() = "Is" and
    sameValue(call.getArgument(0), err) and
    checkedErr = call.getArgument(1) and
    notFoundErrorSentinel(checkedErr)
  )
}

predicate conditionComparesToNil(Expr condition, Expr value, Expr proof) {
  exists(EqualityTestExpr eq, Expr operand, Expr nil |
    conditionContains(condition, eq) and
    proof = eq and
    exprRefersToNil(nil) and
    eq.hasOperands(operand, nil) and
    sameValue(operand, value)
  )
}

predicate conditionComparesErrNonNil(Expr condition, Expr err, Expr proof) {
  exists(NeqExpr neq, Expr operand, Expr nil |
    conditionContains(condition, neq) and
    proof = neq and
    exprRefersToNil(nil) and
    neq.hasOperands(operand, nil) and
    sameValue(operand, err)
  )
}

predicate conditionRejectsTxHashMismatch(Expr condition, Expr receipt, Expr pending, Expr proof) {
  exists(NeqExpr eq, Expr left, Expr right |
    conditionContains(condition, eq) and
    proof = eq and
    eq.hasOperands(left, right) and
    (
      receiptTxHashExpr(left, receipt) and txHashOfPendingTxId(right, pending)
      or receiptTxHashExpr(right, receipt) and txHashOfPendingTxId(left, pending)
    )
  )
}

predicate conditionRejectsBlockHashMismatch(Expr condition, Expr receipt, Expr key, Expr proof) {
  exists(NeqExpr eq, Expr left, Expr right |
    conditionContains(condition, eq) and
    proof = eq and
    eq.hasOperands(left, right) and
    (
      receiptBlockHashExpr(left, receipt) and keyBlockHashExpr(right, key)
      or receiptBlockHashExpr(right, receipt) and keyBlockHashExpr(left, key)
    )
  )
}

predicate guardDominatesSink(ControlFlow::ConditionGuardNode guard, AstNode sink) {
  sink instanceof Expr and guard.dominates(DataFlow::exprNode(sink.(Expr)).getBasicBlock())
  or exists(ControlFlow::Node sinkNode |
    sinkNode.isFirstNodeOf(sink) and guard.dominates(sinkNode.getBasicBlock())
  )
}

predicate failsClosedIfBefore(IfStmt ifStmt, AstNode sink, Expr proof) {
  ifStmt.getEnclosingFunction() = sink.getEnclosingFunction() and
  occursBefore(ifStmt, sink) and
  (
    exists(ContinueStmt cont | cont = ifStmt.getThen().getAChild())
    or exists(ReturnStmt ret | ret = ifStmt.getThen().getAChild())
  ) and
  exists(ControlFlow::ConditionGuardNode guard |
    guard.ensures(DataFlow::exprNode(proof), false) and
    guardDominatesSink(guard, sink)
  )
}

predicate hasConsistencyProof(CallExpr sink, Expr pending) {
  exists(IfStmt ifStmt, Expr proof |
    conditionRejectsConsistencyMismatch(ifStmt.getCond(), pending, sink, proof) and
    failsClosedIfBefore(ifStmt, sink, proof)
  )
}

predicate hasHeightProof(CallExpr sink, Expr pending) {
  exists(IfStmt ifStmt, Expr proof |
    conditionRejectsHeightNotReached(ifStmt.getCond(), pending, sink, proof) and
    failsClosedIfBefore(ifStmt, sink, proof)
  )
}

predicate hasNotFoundRejection(CallExpr sink, Expr err) {
  exists(IfStmt ifStmt, Expr proof |
    conditionContainsNotFoundCheck(ifStmt.getCond(), err, proof) and
    failsClosedIfBefore(ifStmt, sink, proof)
  )
}

predicate hasGenericErrRejection(CallExpr sink, Expr err) {
  exists(IfStmt ifStmt, Expr proof |
    conditionComparesErrNonNil(ifStmt.getCond(), err, proof) and
    failsClosedIfBefore(ifStmt, sink, proof)
  )
}

predicate hasNilReceiptRejection(CallExpr sink, Expr receipt) {
  exists(IfStmt ifStmt, Expr proof |
    conditionComparesToNil(ifStmt.getCond(), receipt, proof) and
    failsClosedIfBefore(ifStmt, sink, proof)
  )
}

predicate hasTxHashProof(CallExpr sink, Expr receipt, Expr pending) {
  exists(IfStmt ifStmt, Expr proof |
    conditionRejectsTxHashMismatch(ifStmt.getCond(), receipt, pending, proof) and
    failsClosedIfBefore(ifStmt, sink, proof)
  )
}

predicate hasBlockHashProof(CallExpr sink, Expr receipt, Expr key) {
  exists(IfStmt ifStmt, Expr proof |
    conditionRejectsBlockHashMismatch(ifStmt.getCond(), receipt, key, proof) and
    failsClosedIfBefore(ifStmt, sink, proof)
  )
}

predicate isPendingVerifyAndPublishSink(CallExpr call, Expr pending, Expr key, Expr receipt) {
  isProductionEvmWatcherFile(call.getFile()) and
  call.getCalleeExpr().(SelectorExpr).getSelector().getName() = "verifyAndPublish" and
  pendingMessageExpr(call.getArgument(0), pending) and
  pendingRangeKey(pending, key) and
  receipt = call.getArgument(3)
}

string missingProof(CallExpr sink, Expr pending, Expr key, Expr receipt) {
  not pendingRangeKey(pending, key, sink.getCalleeExpr().(SelectorExpr).getBase()) and
    result = "same-watcher w.pending range"
  or not hasConsistencyProof(sink, pending) and result = "effective consistency-level match"
  or not hasHeightProof(sink, pending) and result = "height threshold"
  or not exists(CallExpr fetch, Expr err | receiptFetchForSink(sink, pending, receipt, err, fetch)) and
    result = "receipt refetch"
  or exists(CallExpr fetch, Expr err |
    receiptFetchForSink(sink, pending, receipt, err, fetch) and
    not hasNotFoundRejection(sink, err) and
    result = "not-found/orphaned receipt rejection"
  )
  or exists(CallExpr fetch, Expr err |
    receiptFetchForSink(sink, pending, receipt, err, fetch) and
    not hasGenericErrRejection(sink, err) and
    result = "generic receipt error rejection"
  )
  or exists(CallExpr fetch, Expr err |
    receiptFetchForSink(sink, pending, receipt, err, fetch) and
    not hasNilReceiptRejection(sink, receipt) and
    result = "nil receipt rejection"
  )
  or exists(CallExpr fetch, Expr err |
    receiptFetchForSink(sink, pending, receipt, err, fetch) and
    not hasTxHashProof(sink, receipt, pending) and
    result = "tx-hash match"
  )
  or exists(CallExpr fetch, Expr err |
    receiptFetchForSink(sink, pending, receipt, err, fetch) and
    not hasBlockHashProof(sink, receipt, key) and
    result = "block-hash match"
  )
}

from CallExpr sink, Expr pending, Expr key, Expr receipt, string proof
where
  isPendingVerifyAndPublishSink(sink, pending, key, receipt) and
  proof = missingProof(sink, pending, key, receipt)
select sink,
  "EVM pending-message release must reach the effective consistency level and height threshold, refetch the receipt, and fail closed on not-found, generic error, nil receipt, tx-hash mismatch, and block-hash mismatch before verifyAndPublish."
    + " Missing proof: " + proof + "."
