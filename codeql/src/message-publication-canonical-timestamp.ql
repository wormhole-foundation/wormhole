/**
 * @name MessagePublication timestamp bypasses canonical VAA conversion
 * @description MessagePublication timestamps derived through Unix-second conversion must use vaa.TimeFromUnix and fail closed on conversion errors before publication.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/message-publication-canonical-timestamp
 * @tags security
 *       external/cwe/cwe-190
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.controlflow.ControlFlowGraph
import semmle.go.dataflow.GlobalValueNumbering

predicate isProductionNodeFile(File f) {
  f.getRelativePath().matches("node/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate isCommonMessagePublicationType(Type t) {
  t.getName() = "MessagePublication" and
  t.getPackage().getPath().matches("%/node/pkg/common")
}

predicate isMessagePublicationLiteral(CompositeLit lit) {
  isProductionNodeFile(lit.getFile()) and
  isCommonMessagePublicationType(lit.getType())
}

predicate isTimestampField(KeyValueExpr field, Expr timestampValue) {
  field.getKey().(Ident).getName() = "Timestamp" and
  timestampValue = field.getValue()
}

predicate isMessagePublicationTimestampSink(KeyValueExpr field, Expr timestampValue) {
  exists(CompositeLit lit, int i |
    isMessagePublicationLiteral(lit) and
    field = lit.getElement(i) and
    isTimestampField(field, timestampValue)
  )
}

predicate isTimeUnixCall(CallExpr call) {
  call.getCalleeExpr().(SelectorExpr).getSelector().getName() = "Unix" and
  call.getTarget().getQualifiedName() = "time.Unix"
}

predicate isTimeNowCall(CallExpr call) {
  call.getCalleeExpr().(SelectorExpr).getSelector().getName() = "Now" and
  call.getTarget().getQualifiedName() = "time.Now"
}

predicate isLocalWallClockUnixSeconds(CallExpr call) {
  call.getCalleeExpr().(SelectorExpr).getSelector().getName() = "Unix" and
  exists(CallExpr nowCall |
    isTimeNowCall(nowCall) and
    call.getCalleeExpr().(SelectorExpr).getBase() = nowCall
  )
}

predicate isLocalWallClockRoundTrip(CallExpr call) {
  isTimeUnixCall(call) and
  exists(CallExpr secondsCall |
    isLocalWallClockUnixSeconds(secondsCall) and
    DataFlow::localFlow(DataFlow::exprNode(secondsCall), DataFlow::exprNode(call.getArgument(0)))
  )
}

predicate isVaaTimeFromUnixCall(CallExpr call) {
  call.getCalleeExpr().(SelectorExpr).getSelector().getName() = "TimeFromUnix" and
  call.getTarget().getFuncDecl().getName() = "TimeFromUnix" and
  call.getTarget().getFuncDecl().getFile().getRelativePath().matches("%sdk/vaa/%.go")
}

DataFlow::Node timeFromUnixValue(CallExpr call) {
  isVaaTimeFromUnixCall(call) and
  result = DataFlow::extractTupleElement(DataFlow::exprNode(call), 0)
}

predicate assignmentReceivesTupleElement(Assignment assign, CallExpr call, int index, Expr lhs) {
  assign.getRhs(0) = call and
  lhs = assign.getLhs(index)
}

predicate pairedTimeFromUnixErrorExpr(CallExpr call, Expr err) {
  exists(Assignment assign |
    assign.getEnclosingFunction() = call.getEnclosingFunction() and
    assignmentReceivesTupleElement(assign, call, 1, err) and
    not err.(Ident).getName() = "_"
  )
}

predicate neqNilExprFor(Expr condition, Expr err) {
  exists(NeqExpr neq, Expr nil |
    condition = neq and
    exprRefersToNil(nil) and
    (
      exists(Entity target |
        neq.getLeftOperand().(Ident).refersTo(target) and err.(Ident).refersTo(target)
      ) and
      neq.getRightOperand() = nil
      or
      exists(Entity target |
        neq.getRightOperand().(Ident).refersTo(target) and err.(Ident).refersTo(target)
      ) and
      neq.getLeftOperand() = nil
    )
  )
}

predicate eqNilExprFor(Expr condition, Expr err) {
  exists(EqExpr eq, Expr nil |
    condition = eq and
    exprRefersToNil(nil) and
    (
      exists(Entity target |
        eq.getLeftOperand().(Ident).refersTo(target) and err.(Ident).refersTo(target)
      ) and
      eq.getRightOperand() = nil
      or
      exists(Entity target |
        eq.getRightOperand().(Ident).refersTo(target) and err.(Ident).refersTo(target)
      ) and
      eq.getLeftOperand() = nil
    )
  )
}

predicate errorReassignedBeforeGuard(Expr err, AstNode guardUse) {
  exists(Assignment assign, Ident lhs, Entity target |
    lhs = assign.getLhs(_).stripParens() and
    lhs.refersTo(target) and
    err.(Ident).refersTo(target) and
    assign.getLocation().getStartLine() > err.getLocation().getStartLine() and
    assign.getLocation().getStartLine() < guardUse.getLocation().getStartLine()
  )
}

predicate guardProvesNilBefore(Expr err, AstNode use) {
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node useNode, Expr errRead, Expr nil |
    exprRefersToNil(nil) and
    useNode.isFirstNodeOf(use) and
    errRead.getLocation().getStartLine() > err.getLocation().getStartLine() and
    globalValueNumber(DataFlow::exprNode(errRead)) = globalValueNumber(DataFlow::exprNode(err)) and
    not errorReassignedBeforeGuard(err, errRead) and
    guard.ensuresEq(DataFlow::exprNode(errRead), DataFlow::exprNode(nil)) and
    guard.dominates(useNode.getBasicBlock())
  )
  or
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node useNode, Expr condition |
    neqNilExprFor(condition, err) and
    condition.getLocation().getStartLine() > err.getLocation().getStartLine() and
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
  or
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node useNode, Expr condition |
    eqNilExprFor(condition, err) and
    condition.getLocation().getStartLine() > err.getLocation().getStartLine() and
    not errorReassignedBeforeGuard(err, condition) and
    guard.ensures(DataFlow::exprNode(condition), true) and
    (
      use instanceof Expr and
      guard.dominates(DataFlow::exprNode(use.(Expr)).getBasicBlock())
      or
      useNode.isFirstNodeOf(use) and
      guard.dominates(useNode.getBasicBlock())
    )
  )
}

predicate timeFromUnixErrorRejectedBefore(CallExpr call, AstNode use) {
  exists(Expr err |
    pairedTimeFromUnixErrorExpr(call, err) and
    guardProvesNilBefore(err, use)
  )
}

predicate timestampHasUnsafeUnixProvenance(Expr timestampValue, CallExpr unixCall) {
  isTimeUnixCall(unixCall) and
  not isLocalWallClockRoundTrip(unixCall) and
  DataFlow::localFlow(DataFlow::exprNode(unixCall), DataFlow::exprNode(timestampValue))
}

predicate returnsOnlyTimeUnix(FuncDecl f, CallExpr unixCall) {
  isProductionNodeFile(f.getFile()) and
  exists(ReturnStmt ret |
    ret.getEnclosingFunction() = f and
    DataFlow::localFlow(DataFlow::exprNode(unixCall), DataFlow::exprNode(ret.getExpr(0)))
  ) and
  not exists(ReturnStmt ret |
    ret.getEnclosingFunction() = f and
    not exists(CallExpr returnedUnix |
      isTimeUnixCall(returnedUnix) and
      not isLocalWallClockRoundTrip(returnedUnix) and
      DataFlow::localFlow(DataFlow::exprNode(returnedUnix), DataFlow::exprNode(ret.getExpr(0)))
    )
  ) and
  isTimeUnixCall(unixCall) and
  not isLocalWallClockRoundTrip(unixCall)
}

predicate timestampHasUnsafeUnixWrapperProvenance(Expr timestampValue, CallExpr unixCall) {
  exists(CallExpr wrapperCall, FuncDecl wrapper |
    wrapperCall.getTarget().getFuncDecl() = wrapper and
    returnsOnlyTimeUnix(wrapper, unixCall) and
    DataFlow::localFlow(DataFlow::exprNode(wrapperCall), DataFlow::exprNode(timestampValue))
  )
}

predicate timestampHasUncheckedTimeFromUnixProvenance(Expr timestampValue, KeyValueExpr field, CallExpr timeFromUnixCall) {
  DataFlow::localFlow(timeFromUnixValue(timeFromUnixCall), DataFlow::exprNode(timestampValue)) and
  not timeFromUnixErrorRejectedBefore(timeFromUnixCall, field)
}

from KeyValueExpr field, Expr timestampValue, CallExpr report
where
  isMessagePublicationTimestampSink(field, timestampValue) and
  (
    timestampHasUnsafeUnixProvenance(timestampValue, report)
    or
    timestampHasUnsafeUnixWrapperProvenance(timestampValue, report)
    or
    timestampHasUncheckedTimeFromUnixProvenance(timestampValue, field, report)
  )
select report,
  "MessagePublication timestamp from chain-derived Unix seconds must be validated with vaa.TimeFromUnix and must not publish on conversion error; this timestamp appears to bypass the VAA uint32 wire-format check."
