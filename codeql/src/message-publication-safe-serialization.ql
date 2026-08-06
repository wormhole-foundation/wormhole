/**
 * @name MessagePublication serialized with deprecated unsafe format
 * @description Production MessagePublication serialization must use MarshalBinary and UnmarshalBinary because the deprecated helpers omit Unreliable and verificationState.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/message-publication-safe-serialization
 * @tags security
 *       external/cwe/cwe-345
 */

import go
import semmle.go.concepts.GeneratedFile

predicate isProductionNodeFile(File f) {
  f.getRelativePath().matches("node/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate isMessagePublicationDeclaration(Function target) {
  target.getFuncDecl().getFile().getRelativePath() = "node/pkg/common/chainlock.go"
}

predicate isDeprecatedMessagePublicationMarshalTarget(Function target) {
  isMessagePublicationDeclaration(target) and
  target instanceof Method and
  target.getName() = "Marshal" and
  target.(Method).getReceiverBaseType().getName() = "MessagePublication"
}

predicate isDeprecatedMessagePublicationUnmarshalTarget(Function target) {
  isMessagePublicationDeclaration(target) and
  not target instanceof Method and
  target.getName() = "UnmarshalMessagePublication"
}

predicate callTargets(CallExpr call, Function target) {
  target = call.getTarget()
  or
  call.getCalleeExpr().stripParens().(SelectorExpr).refersTo(target)
}

predicate isDeprecatedMessagePublicationMarshal(CallExpr call) {
  isProductionNodeFile(call.getFile()) and
  exists(Function target |
    callTargets(call, target) and
    isDeprecatedMessagePublicationMarshalTarget(target)
  )
}

predicate isDeprecatedMessagePublicationUnmarshal(CallExpr call) {
  isProductionNodeFile(call.getFile()) and
  exists(Function target |
    callTargets(call, target) and
    isDeprecatedMessagePublicationUnmarshalTarget(target)
  )
}

predicate isOldParameterReference(Expr expr, FuncDecl function) {
  exists(Parameter oldParameter, int i |
    oldParameter = function.getParameter(i) and
    oldParameter.getName() = "isOld" and
    expr.stripParens().(Ident).refersTo(oldParameter)
  )
}

predicate conditionIsOld(Expr condition, FuncDecl function) {
  isOldParameterReference(condition, function)
  or
  exists(EqualityTestExpr equality, Expr oldOperand, Expr trueOperand |
    equality = condition.stripParens() and
    equality.getPolarity() = true and
    equality.hasOperands(oldOperand, trueOperand) and
    isOldParameterReference(oldOperand, function) and
    trueOperand.stripParens().getBoolValue() = true
  )
}

predicate isInTrueBranchOfIsOld(Expr expr, FuncDecl function) {
  exists(IfStmt ifStmt |
    conditionIsOld(ifStmt.getCond(), function) and
    expr = ifStmt.getThen().getAChild*()
  )
}

predicate isLegacyGovernorOldRead(CallExpr call) {
  call.getFile().getRelativePath() = "node/pkg/db/governor.go" and
  call.getEnclosingFunction().getName() = "UnmarshalPendingTransfer" and
  isDeprecatedMessagePublicationUnmarshal(call) and
  isInTrueBranchOfIsOld(call, call.getEnclosingFunction())
}

predicate isCallCalleeSelector(SelectorExpr sel) {
  exists(CallExpr call | call.getCalleeExpr().stripParens() = sel)
}

predicate capturesDeprecatedMessagePublicationHelper(SelectorExpr sel, string helper) {
  isProductionNodeFile(sel.getFile()) and
  not isCallCalleeSelector(sel) and
  exists(Function target |
    sel.refersTo(target) and
    (
      isDeprecatedMessagePublicationMarshalTarget(target) and helper = "Marshal"
      or
      isDeprecatedMessagePublicationUnmarshalTarget(target) and helper = "UnmarshalMessagePublication"
    )
  )
}

from AstNode node, string replacement, string deprecated
where
  exists(CallExpr call |
    node = call and
    isDeprecatedMessagePublicationMarshal(call) and
    replacement = "MarshalBinary" and
    deprecated = "Marshal"
  )
  or
  exists(CallExpr call |
    node = call and
    isDeprecatedMessagePublicationUnmarshal(call) and
    not isLegacyGovernorOldRead(call) and
    replacement = "UnmarshalBinary" and
    deprecated = "UnmarshalMessagePublication"
  )
  or
  exists(SelectorExpr sel |
    node = sel and
    capturesDeprecatedMessagePublicationHelper(sel, deprecated) and
    (
      deprecated = "Marshal" and replacement = "MarshalBinary"
      or
      deprecated = "UnmarshalMessagePublication" and replacement = "UnmarshalBinary"
    )
  )
select node,
  "MessagePublication current-format serialization must use " + replacement + "; deprecated " +
    deprecated +
    " omits Unreliable and verificationState and is allowed only for explicit old Governor migration reads."
