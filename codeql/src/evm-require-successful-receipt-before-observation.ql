/**
 * @name EVM watcher observes or publishes without a successful receipt check
 * @description EVM watcher receipt-log parsing must be backed by the same receipt proven successful before parsing; verifyAndPublish must receive a receipt argument proven successful before publication.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/evm-require-successful-receipt-before-observation
 * @tags security
 *       external/cwe/cwe-345
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.controlflow.ControlFlowGraph
import semmle.go.dataflow.DataFlow
import semmle.go.dataflow.GlobalValueNumbering

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
    (
      earlierLine < laterLine
      or
      earlierLine = laterLine and earlierColumn < laterColumn
    )
  )
}

predicate sameVariable(Expr a, Expr b) {
  exists(Entity target |
    a.stripParens().(Ident).refersTo(target) and
    b.stripParens().(Ident).refersTo(target)
  )
}

predicate isReceiptStatusSelectorFor(Expr statusExpr, Expr receipt) {
  exists(SelectorExpr status |
    statusExpr.stripParens() = status and
    status.getSelector().getName() = "Status" and
    sameVariable(status.getBase(), receipt)
  )
}

predicate isReceiptStatusSuccessfulExpr(Expr success) {
  exists(SelectorExpr sel |
    success.stripParens() = sel and
    sel.getSelector().getName() = "ReceiptStatusSuccessful"
  )
}

predicate conditionComparesReceiptStatusToSuccess(Expr condition, Expr receipt, boolean successOutcome) {
  exists(EqualityTestExpr equality, Expr status, Expr success |
    equality = condition.stripParens() and
    equality.hasOperands(status, success) and
    isReceiptStatusSelectorFor(status, receipt) and
    isReceiptStatusSuccessfulExpr(success) and
    (
      equality.getPolarity() = true and successOutcome = true
      or
      equality.getPolarity() = false and successOutcome = false
    )
  )
}

bindingset[receipt, earlier, later]
pragma[inline]
predicate receiptReassignedBetween(Expr receipt, AstNode earlier, AstNode later) {
  exists(Assignment assign, Expr lhs |
    lhs = assign.getLhs(_) and
    sameVariable(lhs, receipt) and
    assign.getEnclosingFunction() = later.getEnclosingFunction() and
    earlier.getEnclosingFunction() = later.getEnclosingFunction() and
    occursBefore(earlier, assign) and
    occursBefore(assign, later)
  )
}

predicate receiptExplicitlyProvenSuccessfulBefore(Expr receipt, AstNode observationOrigin, AstNode use) {
  exists(ControlFlow::ConditionGuardNode guard, Expr condition, boolean outcome |
    condition.getEnclosingFunction() = use.getEnclosingFunction() and
    observationOrigin.getEnclosingFunction() = use.getEnclosingFunction() and
    conditionComparesReceiptStatusToSuccess(condition, receipt, outcome) and
    occursBefore(condition, use) and
    not receiptReassignedBetween(receipt, condition, use) and
    not receiptReassignedBetween(receipt, observationOrigin, condition) and
    guard.ensures(DataFlow::exprNode(condition), outcome) and
    guard.dominates(DataFlow::exprNode(use.(Expr)).getBasicBlock())
  )
  or
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node useNode, Expr condition, boolean outcome |
    condition.getEnclosingFunction() = use.getEnclosingFunction() and
    observationOrigin.getEnclosingFunction() = use.getEnclosingFunction() and
    conditionComparesReceiptStatusToSuccess(condition, receipt, outcome) and
    useNode.isFirstNodeOf(use) and
    occursBefore(condition, use) and
    not receiptReassignedBetween(receipt, condition, use) and
    not receiptReassignedBetween(receipt, observationOrigin, condition) and
    guard.ensures(DataFlow::exprNode(condition), outcome) and
    guard.dominates(useNode.getBasicBlock())
  )
}

predicate isParseLogMessagePublishedCall(CallExpr call) {
  isProductionEvmWatcherFile(call.getFile()) and
  call.getCalleeExpr().(SelectorExpr).getSelector().getName() = "ParseLogMessagePublished" and
  call.getTarget().getFuncDecl().getName() = "ParseLogMessagePublished"
}

predicate receiptBackedBySuccessfulStatus(Expr receipt, AstNode observationOrigin, AstNode use) {
  receiptExplicitlyProvenSuccessfulBefore(receipt, observationOrigin, use)
}

predicate assignmentReceivesTupleElement(Assignment assign, CallExpr call, int index, Expr lhs) {
  assign.getRhs(0) = call and
  lhs = assign.getLhs(index)
}

predicate returnsNilErrorAt(ReturnStmt ret, int errorIndex) {
  ret.getNumExpr() > errorIndex and
  exprRefersToNil(ret.getExpr(errorIndex))
}

bindingset[ret, errorIndex]
pragma[inline]
predicate successfulReturnProvesReceiptStatus(ReturnStmt ret, int errorIndex) {
  returnsNilErrorAt(ret, errorIndex) and
  exists(
    Expr receipt, Expr condition, boolean outcome, ControlFlow::ConditionGuardNode guard,
    ControlFlow::Node returnNode
  |
    receipt = ret.getExpr(0) and
    conditionComparesReceiptStatusToSuccess(condition, receipt, outcome) and
    condition.getEnclosingFunction() = ret.getEnclosingFunction() and
    occursBefore(condition, ret) and
    not receiptReassignedBetween(receipt, condition, ret) and
    returnNode.isFirstNodeOf(ret) and
    guard.ensures(DataFlow::exprNode(condition), outcome) and
    guard.dominates(returnNode.getBasicBlock())
  )
}

bindingset[f]
pragma[inline]
predicate isSafeMessageEventsFunction(FuncDecl f, int messagesIndex, int errorIndex) {
  isProductionEvmWatcherFile(f.getFile()) and
  f.getName() = "MessageEventsForTransaction" and
  f.getFile().getRelativePath() in [
      "node/pkg/watchers/evm/by_transaction.go", "node/pkg/watchers/evm/stubs.go"
    ] and
  (
    messagesIndex = 1 and errorIndex = 2
    or
    messagesIndex = 2 and errorIndex = 3
  ) and
  exists(ReturnStmt ret |
    ret.getEnclosingFunction() = f and
    returnsNilErrorAt(ret, errorIndex)
  ) and
  not exists(ReturnStmt ret |
    ret.getEnclosingFunction() = f and
    returnsNilErrorAt(ret, errorIndex) and
    not successfulReturnProvesReceiptStatus(ret, errorIndex)
  )
}

predicate messageEventsTuple(
  CallExpr helper, Assignment assign, Expr receipt, Expr messages, Expr err, int messagesIndex,
  int errorIndex
) {
  isSafeMessageEventsFunction(helper.getTarget().getFuncDecl(), messagesIndex, errorIndex) and
  assign.getEnclosingFunction() = helper.getEnclosingFunction() and
  assignmentReceivesTupleElement(assign, helper, 0, receipt) and
  assignmentReceivesTupleElement(assign, helper, messagesIndex, messages) and
  assignmentReceivesTupleElement(assign, helper, errorIndex, err) and
  not err.stripParens().(Ident).getName() = "_"
}

predicate neqNilExprFor(Expr condition, Expr err) {
  exists(NeqExpr neq, Expr nil |
    condition = neq and
    exprRefersToNil(nil) and
    (
      exists(Entity target |
        neq.getLeftOperand().stripParens().(Ident).refersTo(target) and
        err.stripParens().(Ident).refersTo(target)
      ) and
      neq.getRightOperand() = nil
      or
      exists(Entity target |
        neq.getRightOperand().stripParens().(Ident).refersTo(target) and
        err.stripParens().(Ident).refersTo(target)
      ) and
      neq.getLeftOperand() = nil
    )
  )
}

predicate errorReassignedBeforeGuard(Expr err, AstNode guardUse) {
  exists(Assignment assign, Ident lhs, Entity target |
    lhs = assign.getLhs(_).stripParens() and
    lhs.refersTo(target) and
    err.stripParens().(Ident).refersTo(target) and
    occursBefore(err, assign) and
    occursBefore(assign, guardUse)
  )
}

predicate guardProvesNilBefore(Expr err, AstNode use) {
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node useNode, Expr errRead, Expr nil |
    exprRefersToNil(nil) and
    useNode.isFirstNodeOf(use) and
    occursBefore(err, errRead) and
    globalValueNumber(DataFlow::exprNode(errRead)) = globalValueNumber(DataFlow::exprNode(err)) and
    not errorReassignedBeforeGuard(err, errRead) and
    guard.ensuresEq(DataFlow::exprNode(errRead), DataFlow::exprNode(nil)) and
    guard.dominates(useNode.getBasicBlock())
  )
  or
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node useNode, Expr condition |
    neqNilExprFor(condition, err) and
    occursBefore(err, condition) and
    not errorReassignedBeforeGuard(err, condition) and
    guard.ensures(DataFlow::exprNode(condition), false) and
    (
      use instanceof Expr and
      guard.dominates(DataFlow::exprNode(use.(Expr)).getBasicBlock())
      or
      useNode.isFirstNodeOf(use) and
      guard.dominates(useNode.getBasicBlock())
    )
  )
}

bindingset[message, messages, sink]
pragma[inline]
predicate messageReadFromReturnedSlice(Expr message, Expr messages, CallExpr sink) {
  exists(RangeStmt loop, Ident rangeValue, Ident messageRead, Variable v |
    sameVariable(loop.getDomain(), messages) and
    rangeValue = loop.getValue() and
    rangeValue.refersTo(v) and
    messageRead.refersTo(v) and
    messageRead = message.stripParens() and
    loop.getBody() = sink.getParent*() and
    not receiptReassignedBetween(messageRead, loop, sink)
  )
  or
  exists(IndexExpr idx |
    idx = message.stripParens() and
    sameVariable(idx.getBase(), messages)
  )
}

bindingset[messages, earlier, later]
pragma[inline]
predicate sliceElementAssignedBetween(Expr messages, AstNode earlier, AstNode later) {
  exists(Assignment assign, IndexExpr index |
    index = assign.getLhs(_).stripParens() and
    sameVariable(index.getBase(), messages) and
    assign.getEnclosingFunction() = later.getEnclosingFunction() and
    occursBefore(earlier, assign) and
    occursBefore(assign, later)
  )
}

bindingset[sink, message, receipt]
pragma[inline]
predicate backedBySuccessfulMessageEventsTuple(CallExpr sink, Expr message, Expr receipt) {
  exists(
    CallExpr helper, Assignment assign, Expr returnedReceipt, Expr messages, Expr err,
    int messagesIndex, int errorIndex
  |
    messageEventsTuple(
      helper, assign, returnedReceipt, messages, err, messagesIndex, errorIndex
    ) and
    assign.getEnclosingFunction() = sink.getEnclosingFunction() and
    sameVariable(receipt, returnedReceipt) and
    messageReadFromReturnedSlice(message, messages, sink) and
    occursBefore(assign, sink) and
    not receiptReassignedBetween(returnedReceipt, assign, sink) and
    not receiptReassignedBetween(messages, assign, sink) and
    not sliceElementAssignedBetween(messages, assign, sink) and
    guardProvesNilBefore(err, sink)
  )
}

predicate isEvmWatcherVerifyAndPublishCall(CallExpr call, Expr messageArg, Expr receiptArg) {
  isProductionEvmWatcherFile(call.getFile()) and
  call.getCalleeExpr().(SelectorExpr).getSelector().getName() = "verifyAndPublish" and
  call.getTarget().getFuncDecl().getName() = "verifyAndPublish" and
  call.getTarget().getFuncDecl().getFile().getRelativePath().matches("node/pkg/watchers/evm/%.go") and
  messageArg = call.getArgument(0) and
  receiptArg = call.getArgument(3)
}

predicate isReceiptLogsSelectorFor(Expr logs, Expr receipt) {
  exists(SelectorExpr sel |
    logs.stripParens() = sel and
    sel.getSelector().getName() = "Logs" and
    sameVariable(sel.getBase(), receipt)
  )
}

predicate exprIsReceiptLogsForFrom(Expr logs, Expr receipt, AstNode observationOrigin, AstNode use) {
  isReceiptLogsSelectorFor(logs, receipt) and
  observationOrigin = use
  or
  exists(Assignment assign, Expr lhs, Expr rhs |
    lhs = assign.getLhs(0) and
    assign.getEnclosingFunction() = use.getEnclosingFunction() and
    sameVariable(logs, lhs) and
    isReceiptLogsSelectorFor(rhs, receipt) and
    assign.getRhs(0) = rhs and
    occursBefore(assign, use) and
    not receiptReassignedBetween(lhs, assign, use) and
    observationOrigin = assign
  )
}

predicate exprReadsLogFromReceipt(Expr logExpr, Expr receipt, AstNode observationOrigin, AstNode use) {
  exists(RangeStmt loop, Ident rangeValue, Ident logRead, Variable v |
    isProductionEvmWatcherFile(use.getFile()) and
    loop.getEnclosingFunction() = use.getEnclosingFunction() and
    exprIsReceiptLogsForFrom(loop.getDomain(), receipt, observationOrigin, use) and
    rangeValue = loop.getValue() and
    rangeValue.refersTo(v) and
    logRead.refersTo(v) and
    (logRead = logExpr or logRead = logExpr.getAChild()) and
    loop.getBody() = use.getParent*()
  )
  or
  exists(IndexExpr idx |
    isProductionEvmWatcherFile(use.getFile()) and
    idx.getEnclosingFunction() = use.getEnclosingFunction() and
    (idx = logExpr.stripParens() or idx = logExpr.getAChild()) and
    exprIsReceiptLogsForFrom(idx.getBase(), receipt, observationOrigin, use)
  )
}

predicate parseCallReadsLogFromReceipt(CallExpr call, Expr receipt, AstNode observationOrigin) {
  isParseLogMessagePublishedCall(call) and
  exprReadsLogFromReceipt(call.getArgument(0), receipt, observationOrigin, call)
}

predicate helperParameterParsed(FuncDecl f, Parameter p) {
  exists(CallExpr parse, Ident pRead |
    isProductionEvmWatcherFile(f.getFile()) and
    parse.getEnclosingFunction() = f and
    isParseLogMessagePublishedCall(parse) and
    pRead.refersTo(p) and
    (pRead = parse.getArgument(0) or pRead = parse.getArgument(0).getAChild())
  )
}

predicate helperCallParsesLogFromReceipt(CallExpr call, Expr receipt, AstNode observationOrigin) {
  exists(int i, FuncDecl f, Parameter p |
    isProductionEvmWatcherFile(call.getFile()) and
    f = call.getTarget().getFuncDecl() and
    isProductionEvmWatcherFile(f.getFile()) and
    p = f.getParameter(i) and
    helperParameterParsed(f, p) and
    exprReadsLogFromReceipt(call.getArgument(i), receipt, observationOrigin, call)
  )
}

from CallExpr sink, Expr receipt, AstNode observationOrigin, string kind
where
  (
    exists(CallExpr call, Expr message |
      sink = call and
      observationOrigin = call and
      isEvmWatcherVerifyAndPublishCall(call, message, receipt) and
      kind = "publishes with (*Watcher).verifyAndPublish"
    )
    or
    exists(CallExpr call |
      sink = call and
      (
        parseCallReadsLogFromReceipt(call, receipt, observationOrigin)
        or
        helperCallParsesLogFromReceipt(call, receipt, observationOrigin)
      ) and
      kind = "parses LogMessagePublished events from receipt.Logs"
    )
  ) and
  not receiptBackedBySuccessfulStatus(receipt, observationOrigin, sink) and
  not exists(Expr message |
    isEvmWatcherVerifyAndPublishCall(sink, message, receipt) and
    backedBySuccessfulMessageEventsTuple(sink, message, receipt)
  )
select sink,
  "EVM watcher observation must prove the same transaction receipt has Status == ReceiptStatusSuccessful before it " +
    kind + "; nil, tx-hash, block-hash, or finality checks do not prove receipt success."
