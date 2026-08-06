/**
 * @name Solana message account parsed without constructor validation
 * @description Solana watcher message account data must be created by NewMessageAccountData before parsing or processing so the discriminator and length checks are enforced.
 * @kind problem
 * @problem.severity error
 * @precision high
 * @id wormhole/go/solana-message-account-validation
 * @tags security
 *       external/cwe/cwe-20
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.controlflow.ControlFlowGraph
import semmle.go.dataflow.GlobalValueNumbering

predicate isProductionSolanaWatcherFile(File f) {
  f.getRelativePath().matches("node/pkg/watchers/solana/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate hasSolanaWatcherTarget(CallExpr call, string name) {
  isProductionSolanaWatcherFile(call.getFile()) and
  call.getTarget().getFuncDecl().getFile().getRelativePath().matches("node/pkg/watchers/solana/%.go") and
  call.getTarget().getFuncDecl().getName() = name
}

predicate isMessageAccountConstructorCall(CallExpr call) {
  hasSolanaWatcherTarget(call, "NewMessageAccountData")
}

predicate isParserOrProcessorImplementation(FuncDecl f) {
  isProductionSolanaWatcherFile(f.getFile()) and
  f.getName() in ["ParseMessagePublicationAccount", "processMessageAccount"]
}

predicate isMessageAccountSink(CallExpr call, Expr arg) {
  hasSolanaWatcherTarget(call, "ParseMessagePublicationAccount") and
  not isParserOrProcessorImplementation(call.getEnclosingFunction()) and
  arg = call.getArgument(0)
  or
  hasSolanaWatcherTarget(call, "processMessageAccount") and
  arg = call.getArgument(1)
}

DataFlow::Node constructorValue(CallExpr call) {
  isMessageAccountConstructorCall(call) and
  result = DataFlow::extractTupleElement(DataFlow::exprNode(call), 0)
}

predicate assignmentReceivesTupleElement(Assignment assign, CallExpr call, int index, Expr lhs) {
  assign.getRhs(0) = call and
  lhs = assign.getLhs(index)
}

predicate pairedConstructorErrorExpr(CallExpr call, Expr err) {
  exists(Assignment assign |
    assign.getEnclosingFunction() = call.getEnclosingFunction() and
    assignmentReceivesTupleElement(assign, call, 1, err)
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
}

predicate constructorErrorRejectedBefore(CallExpr ctor, AstNode use) {
  exists(Expr err |
    pairedConstructorErrorExpr(ctor, err) and
    guardProvesNilBefore(err, use)
  )
}

predicate returnsNilError(ReturnStmt ret) {
  ret.getNumExpr() >= 2 and
  exprRefersToNil(ret.getExpr(1))
}

predicate returnValueHasCheckedConstructorProvenance(ReturnStmt ret) {
  exists(CallExpr ctor |
    DataFlow::localFlow(constructorValue(ctor), DataFlow::exprNode(ret.getExpr(0))) and
    constructorErrorRejectedBefore(ctor, ret)
  )
}

predicate returnValueHasCheckedProvenance(ReturnStmt ret, int helperDepth) {
  (
    helperDepth in [0 .. 1] and
    returnValueHasCheckedConstructorProvenance(ret)
  )
  or
  exists(CallExpr helper, int nestedDepth |
    helperDepth in [1 .. 1] and
    nestedDepth = helperDepth - 1 and
    DataFlow::localFlow(
      DataFlow::extractTupleElement(DataFlow::exprNode(helper), 0), DataFlow::exprNode(ret.getExpr(0))
    ) and
    isSafeMessageAccountFactoryAtDepth(helper.getTarget().getFuncDecl(), nestedDepth) and
    helperErrorRejectedBefore(helper, ret)
  )
}

predicate isSafeMessageAccountFactoryAtDepth(FuncDecl f, int helperDepth) {
  helperDepth in [0 .. 1] and
  isProductionSolanaWatcherFile(f.getFile()) and
  not f.getName() = "NewMessageAccountData" and
  exists(ReturnStmt ret |
    ret.getEnclosingFunction() = f and
    returnsNilError(ret)
  ) and
  not exists(ReturnStmt ret |
    ret.getEnclosingFunction() = f and
    returnsNilError(ret) and
    not returnValueHasCheckedProvenance(ret, helperDepth)
  )
}

predicate isSafeMessageAccountFactory(FuncDecl f) {
  isSafeMessageAccountFactoryAtDepth(f, 1)
}

DataFlow::Node helperValue(CallExpr call) {
  isSafeMessageAccountFactory(call.getTarget().getFuncDecl()) and
  result = DataFlow::extractTupleElement(DataFlow::exprNode(call), 0)
}

predicate pairedHelperErrorExpr(CallExpr call, Expr err) {
  exists(Assignment assign |
    assign.getEnclosingFunction() = call.getEnclosingFunction() and
    assign.getRhs(0) = call and
    assign.getLhs(1) = err
  )
}

predicate localValueOrPointerRoundTripFlow(DataFlow::Node source, DataFlow::Node sink) {
  DataFlow::localFlow(source, sink)
  or
  exists(Assignment assign, AddressExpr addr, StarExpr deref |
    assign.getRhs(0) = addr and
    deref = sink.asExpr().stripParens() and
    exists(Entity target |
      assign.getLhs(0).(Ident).refersTo(target) and deref.getAChild().(Ident).refersTo(target)
    ) and
    DataFlow::localFlow(source, DataFlow::exprNode(addr.getOperand()))
  )
}

predicate helperErrorRejectedBefore(CallExpr helper, AstNode use) {
  exists(Expr err |
    pairedHelperErrorExpr(helper, err) and
    guardProvesNilBefore(err, use)
  )
}

predicate hasSuccessfulConstructorProvenance(Expr arg, CallExpr sink) {
  exists(CallExpr ctor |
    localValueOrPointerRoundTripFlow(constructorValue(ctor), DataFlow::exprNode(arg)) and
    constructorErrorRejectedBefore(ctor, sink)
  )
  or
  exists(CallExpr helper |
    localValueOrPointerRoundTripFlow(helperValue(helper), DataFlow::exprNode(arg)) and
    helperErrorRejectedBefore(helper, sink)
  )
}

from CallExpr sink, Expr arg
where
  isMessageAccountSink(sink, arg) and
  not hasSuccessfulConstructorProvenance(arg, sink)
select sink, "Solana message account data must be created by NewMessageAccountData before parsing or processing."
