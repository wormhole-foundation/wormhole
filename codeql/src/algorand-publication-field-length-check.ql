/**
 * @name Algorand publication field decoded without exact length check
 * @description Algorand publishMessage nonce and sequence bytes must be proven exactly 8 bytes before binary.BigEndian.Uint64 decoding.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/algorand-publication-field-length-check
 * @tags security
 *       external/cwe/cwe-20
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.controlflow.ControlFlowGraph
import semmle.go.dataflow.GlobalValueNumbering

predicate isProductionAlgorandWatcherFile(File f) {
  (
    f.getRelativePath().matches("node/pkg/watchers/algorand/%.go")
    or
    f.getRelativePath().matches("pkg/watchers/algorand/%.go")
  ) and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate isAlgorandPublicationFunction(FuncDecl f) {
  isProductionAlgorandWatcherFile(f.getFile()) and
  exists(AstNode n |
    n.getEnclosingFunction() = f and
    (
      exists(Ident id | id = n and id.getName() in ["publishMessage", "MessagePublication"])
      or
      exists(SelectorExpr sel, Field field |
        sel = n and
        sel.refersTo(field) and
        field.getName() in ["Nonce", "Sequence"]
      )
    )
  )
}

predicate isApplicationArgsTwo(IndexExpr idx) {
  idx.getIndex().getIntValue() = 2 and
  exists(SelectorExpr args, Field field |
    args = idx.getBase().stripParens() and
    args.refersTo(field) and
    field.getName() = "ApplicationArgs"
  )
}

predicate isLogsZero(IndexExpr idx) {
  idx.getIndex().getIntValue() = 0 and
  exists(SelectorExpr logs, Field field |
    logs = idx.getBase().stripParens() and
    logs.refersTo(field) and
    field.getName() = "Logs"
  )
}

predicate isSequenceBytesExpr(Expr e) {
  exists(ConversionExpr conv, IndexExpr idx |
    e.stripParens() = conv and
    idx = conv.getOperand().stripParens() and
    isLogsZero(idx)
  )
}

predicate isPublicationFieldStorageExpr(Expr e) {
  isApplicationArgsTwo(e.stripParens().(IndexExpr))
  or
  isLogsZero(e.stripParens().(IndexExpr))
}

predicate isExactPublicationFieldExpr(Expr e) {
  isApplicationArgsTwo(e.stripParens().(IndexExpr))
  or
  isSequenceBytesExpr(e)
}

predicate publicationFieldFlowsTo(Expr sinkArg, Expr source) {
  isExactPublicationFieldExpr(source) and
  (
    DataFlow::localFlow(DataFlow::exprNode(source), DataFlow::exprNode(sinkArg.stripParens()))
    or
    exists(Assignment assign, Expr lhs |
      assign.getRhs(_) = source and
      lhs = assign.getLhs(_) and
      sameLocalVariable(lhs, sinkArg) and
      assign.getLocation().getStartLine() < sinkArg.getLocation().getStartLine()
    )
  )
}

predicate isBinaryBigEndianUint64Call(CallExpr call) {
  call.getTarget().getName() = "Uint64" and
  call.getTarget().getPackage().getPath() = "encoding/binary" and
  exists(SelectorExpr uint64Sel, Expr ord |
    uint64Sel = call.getCalleeExpr().stripParens() and
    uint64Sel.getSelector().getName() = "Uint64" and
    ord = uint64Sel.getBase().stripParens() and
    isBinaryBigEndianOrderExpr(ord, call)
  )
}

predicate isBinaryBigEndianSelector(Expr e) {
  exists(SelectorExpr bigEndianSel |
    bigEndianSel = e.stripParens() and
    bigEndianSel.getSelector().getName() = "BigEndian"
  )
}

predicate isBinaryBigEndianOrderExpr(Expr ord, AstNode use) {
  isBinaryBigEndianSelector(ord)
  or
  exists(Assignment assign, Expr lhs, Expr rhs |
    rhs = assign.getRhs(_) and
    isBinaryBigEndianSelector(rhs) and
    lhs = assign.getLhs(_) and
    sameLocalVariable(lhs, ord) and
    assign.getEnclosingFunction() = use.getEnclosingFunction() and
    assign.getLocation().getStartLine() < use.getLocation().getStartLine()
  )
}

predicate sameLocalValue(Expr a, Expr b) {
  globalValueNumber(DataFlow::exprNode(a.stripParens())) =
    globalValueNumber(DataFlow::exprNode(b.stripParens()))
  or
  DataFlow::localFlow(DataFlow::exprNode(a.stripParens()), DataFlow::exprNode(b.stripParens()))
  or
  DataFlow::localFlow(DataFlow::exprNode(b.stripParens()), DataFlow::exprNode(a.stripParens()))
}

predicate samePublicationFieldValue(Expr a, Expr b) {
  sameLocalValue(a, b)
  or
  isApplicationArgsTwo(a.stripParens().(IndexExpr)) and isApplicationArgsTwo(b.stripParens().(IndexExpr))
  or
  isLogsZero(a.stripParens().(IndexExpr)) and isSequenceBytesExpr(b)
  or
  isSequenceBytesExpr(a) and isLogsZero(b.stripParens().(IndexExpr))
  or
  isSequenceBytesExpr(a) and isSequenceBytesExpr(b)
}

predicate isLenCallFor(CallExpr lenCall, Expr value) {
  lenCall.getCalleeName() = "len" and
  lenCall.getNumArgument() = 1 and
  samePublicationFieldValue(lenCall.getArgument(0), value)
}

predicate isEight(Expr e) { e.getIntValue() = 8 }

predicate sameLocalVariableAssigned(Expr lhs, Expr value) {
  sameLocalVariable(lhs, value)
}

predicate valueReassignedBetween(Expr value, Expr condition, AstNode use) {
  exists(Assignment assign, Expr lhs |
    lhs = assign.getLhs(_) and
    sameLocalVariableAssigned(lhs, value) and
    assign.getLocation().getStartLine() > condition.getLocation().getStartLine() and
    assign.getLocation().getStartLine() < use.getLocation().getStartLine()
  )
  or
  exists(Assignment assign, Expr lhs |
    lhs = assign.getLhs(_) and
    isPublicationFieldStorageExpr(lhs) and
    samePublicationFieldValue(lhs, value) and
    assign.getLocation().getStartLine() > condition.getLocation().getStartLine() and
    assign.getLocation().getStartLine() < use.getLocation().getStartLine()
  )
}

predicate exactLengthConditionFor(Expr condition, Expr value, boolean truth) {
  truth = true and
  exists(EqExpr eq, CallExpr lenCall |
    condition = eq and
    (
      isLenCallFor(lenCall, value) and lenCall = eq.getLeftOperand() and isEight(eq.getRightOperand())
      or
      isLenCallFor(lenCall, value) and lenCall = eq.getRightOperand() and isEight(eq.getLeftOperand())
    )
  )
  or
  truth = false and
  exists(NeqExpr neq, CallExpr lenCall |
    condition = neq and
    (
      isLenCallFor(lenCall, value) and lenCall = neq.getLeftOperand() and isEight(neq.getRightOperand())
      or
      isLenCallFor(lenCall, value) and lenCall = neq.getRightOperand() and isEight(neq.getLeftOperand())
    )
  )
}

predicate exactLengthGuardDominates(Expr value, AstNode use) {
  exists(ControlFlow::ConditionGuardNode guard, Expr condition, boolean truth |
    exactLengthConditionFor(condition, value, truth) and
    not valueReassignedBetween(value, condition, use) and
    guard.ensures(DataFlow::exprNode(condition), truth) and
    (
      use instanceof Expr and
      guard.dominates(DataFlow::exprNode(use.(Expr)).getBasicBlock())
      or
      exists(ControlFlow::Node useNode |
        useNode.isFirstNodeOf(use) and
        guard.dominates(useNode.getBasicBlock())
      )
    )
  )
}

predicate isDirectUnguardedPublicationDecode(CallExpr call) {
  isProductionAlgorandWatcherFile(call.getFile()) and
  isAlgorandPublicationFunction(call.getEnclosingFunction()) and
  isBinaryBigEndianUint64Call(call) and
  exists(Expr fieldSource |
    publicationFieldFlowsTo(call.getArgument(0), fieldSource) and
    not exactLengthGuardDominates(call.getArgument(0), call)
  )
}

predicate parameterFlowsToUint64(Parameter parameter, CallExpr uint64Call) {
  exists(FuncDecl helper, Ident parameterRead |
    helper = parameter.getFunction() and
    parameterRead.refersTo(parameter) and
    parameterRead.getEnclosingFunction() = helper and
    isBinaryBigEndianUint64Call(uint64Call) and
    uint64Call.getEnclosingFunction() = helper and
    DataFlow::localFlow(
      DataFlow::exprNode(parameterRead), DataFlow::exprNode(uint64Call.getArgument(0))
    )
  )
}

predicate helperParameterHasExactGuard(Parameter parameter, CallExpr uint64Call) {
  exists(FuncDecl helper, Ident parameterRead |
    helper = parameter.getFunction() and
    parameterRead.refersTo(parameter) and
    parameterRead.getEnclosingFunction() = helper and
    exactLengthGuardDominates(parameterRead, uint64Call)
  )
}

predicate assignmentReceivesTupleElement(Assignment assign, CallExpr call, int index, Expr lhs) {
  assign.getRhs(0) = call and
  lhs = assign.getLhs(index)
}

predicate exprRefersToNilLocal(Expr e) { e.(Ident).getName() = "nil" }

predicate sameLocalVariable(Expr a, Expr b) {
  exists(Entity target |
    a.stripParens().(Ident).refersTo(target) and
    b.stripParens().(Ident).refersTo(target)
  )
}

predicate neqNilExprFor(Expr condition, Expr err) {
  exists(NeqExpr neq, Expr nil |
    condition = neq and
    exprRefersToNilLocal(nil) and
    (
      sameLocalVariable(neq.getLeftOperand(), err) and neq.getRightOperand() = nil
      or
      sameLocalVariable(neq.getRightOperand(), err) and neq.getLeftOperand() = nil
    )
  )
}

predicate errorRejectedBefore(Expr err, AstNode use) {
  exists(ControlFlow::ConditionGuardNode guard, Expr condition |
    neqNilExprFor(condition, err) and
    condition.getLocation().getStartLine() > err.getLocation().getStartLine() and
    guard.ensures(DataFlow::exprNode(condition), false) and
    (
      use instanceof Expr and
      guard.dominates(DataFlow::exprNode(use.(Expr)).getBasicBlock())
      or
      exists(ControlFlow::Node useNode |
        useNode.isFirstNodeOf(use) and
        guard.dominates(useNode.getBasicBlock())
      )
    )
  )
}

predicate pairedHelperErrorExpr(CallExpr call, Expr err) {
  exists(Assignment assign |
    assign.getEnclosingFunction() = call.getEnclosingFunction() and
    assignmentReceivesTupleElement(assign, call, 1, err)
  )
}

predicate pairedHelperResultExpr(CallExpr call, Expr decoded) {
  exists(Assignment assign |
    assign.getEnclosingFunction() = call.getEnclosingFunction() and
    assignmentReceivesTupleElement(assign, call, 0, decoded)
  )
}

predicate exprContainsSameLocalValue(Expr outer, Expr value) {
  sameLocalValue(outer, value)
  or
  exists(Expr child |
    child = outer.getAChild*() and
    sameLocalValue(child, value)
  )
  or
  exists(Ident id |
    id = outer.getAChild*() and
    sameLocalVariable(id, value)
  )
}

predicate publicationUsesHelperResult(CompositeLit publication, Expr decoded) {
  exists(int i, KeyValueExpr field, Expr fieldValue |
    field = publication.getElement(i) and
    field.getKey().(Ident).getName() in ["Nonce", "Sequence"] and
    fieldValue = field.getValue() and
    exprContainsSameLocalValue(fieldValue, decoded)
  )
}

predicate helperResultPublicationUse(CallExpr call, CompositeLit publication) {
  exists(Expr decoded |
    pairedHelperResultExpr(call, decoded) and
    publication.getEnclosingFunction() = call.getEnclosingFunction() and
    publication.getLocation().getStartLine() > call.getLocation().getStartLine() and
    publication.getType().getName() = "MessagePublication" and
    publicationUsesHelperResult(publication, decoded)
  )
}

predicate helperResultPublicationUseNotRejected(CallExpr call) {
  exists(Expr err, CompositeLit publication |
    pairedHelperErrorExpr(call, err) and
    helperResultPublicationUse(call, publication) and
    not errorRejectedBefore(err, publication)
  )
}

predicate isThinHelperPublicationDecode(CallExpr call, Parameter parameter, CallExpr internalUint64) {
  exists(FuncDecl helper |
  helper = parameter.getFunction() and
  isProductionAlgorandWatcherFile(call.getFile()) and
  isAlgorandPublicationFunction(call.getEnclosingFunction()) and
  call.getTarget().getFuncDecl() = helper and
  parameterFlowsToUint64(parameter, internalUint64) and
  isProductionAlgorandWatcherFile(helper.getFile()) and
  exists(int i, Expr fieldSource |
    parameter = helper.getParameter(i) and
    publicationFieldFlowsTo(call.getArgument(i), fieldSource)
  )
  )
}

predicate isUnguardedThinHelperPublicationDecode(CallExpr call) {
  exists(Parameter parameter, CallExpr internalUint64 |
    isThinHelperPublicationDecode(call, parameter, internalUint64) and
    (
      not helperParameterHasExactGuard(parameter, internalUint64)
      or
      (
        exists(CompositeLit publication | helperResultPublicationUse(call, publication)) and
        helperResultPublicationUseNotRejected(call)
      )
    )
  )
}

from CallExpr call
where isDirectUnguardedPublicationDecode(call) or isUnguardedThinHelperPublicationDecode(call)
select call, "Algorand publishMessage nonce/sequence bytes must be proven exactly 8 bytes before Uint64 decoding; malformed lengths can panic the watcher before the observation is skipped."
