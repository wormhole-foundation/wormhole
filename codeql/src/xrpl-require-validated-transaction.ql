/**
 * @name XRPL parser call without validated transaction proof
 * @description XRPL parser entry points can construct Wormhole observations and must only be called after proving the source transaction is in a validated ledger.
 * @kind problem
 * @problem.severity error
 * @precision high
 * @id wormhole/go/xrpl-require-validated-transaction
 * @tags security
 *       external/cwe/cwe-345
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.controlflow.ControlFlowGraph
import semmle.go.dataflow.GlobalValueNumbering

predicate isProductionXrplWatcherFile(File f) {
  f.getRelativePath().matches("node/pkg/watchers/xrpl/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate isParserEntryPoint(CallExpr call) {
  isProductionXrplWatcherFile(call.getFile()) and
  exists(MethodDecl method |
    method = call.getTarget().getFuncDecl() and
    method.getName() in ["ParseTransactionStream", "ParseTxResponse"] and
    method.getReceiverBaseType().getName() = "Parser" and
    isProductionXrplWatcherFile(method.getFile())
  )
}

predicate isValidatedFieldReadFor(SelectorExpr validated, Expr tx) {
  exists(Field field |
    validated.refersTo(field) and
    field.getName() = "Validated" and
    globalValueNumber(DataFlow::exprNode(validated.getBase())) = globalValueNumber(DataFlow::exprNode(tx))
  )
}

predicate wrapsTxResponseValue(CompositeLit wrapper, Expr tx) {
  exists(int i, KeyValueExpr fieldInit |
    fieldInit = wrapper.getElement(i) and
    fieldInit.getKey().(Ident).getName() = "TxResponse" and
    globalValueNumber(DataFlow::exprNode(fieldInit.getValue())) = globalValueNumber(DataFlow::exprNode(tx))
  )
}

predicate addressWrapsTxResponseValue(AddressExpr wrapper, Expr tx) {
  wrapsTxResponseValue(wrapper.getOperand().stripParens().(CompositeLit), tx)
}

predicate wrapperExprWrapsTxResponseValue(Expr wrapper, Expr tx) {
  wrapsTxResponseValue(wrapper.stripParens().(CompositeLit), tx)
  or
  addressWrapsTxResponseValue(wrapper.stripParens().(AddressExpr), tx)
}

predicate isParseTxResponseWrapperFor(CallExpr call, Expr wrapper, Expr tx) {
  call.getTarget().getFuncDecl().getName() = "ParseTxResponse" and
  wrapperExprWrapsTxResponseValue(wrapper, tx) and
  DataFlow::localFlow(DataFlow::exprNode(wrapper), DataFlow::exprNode(call.getArgument(0)))
}

predicate isParsedTransactionArgument(CallExpr call, Expr tx) {
  call.getTarget().getFuncDecl().getName() = "ParseTransactionStream" and
  globalValueNumber(DataFlow::exprNode(call.getArgument(0))) = globalValueNumber(DataFlow::exprNode(tx))
  or
  exists(Expr wrapper | isParseTxResponseWrapperFor(call, wrapper, tx))
}

predicate isValidatedProofExprFor(Expr proof, Expr tx, SelectorExpr validated) {
  proof = validated and
  isValidatedFieldReadFor(validated, tx)
  or
  exists(Assignment assign |
    assign.getRhs(0) = validated and
    assign.getLhs(0) = proof and
    isValidatedFieldReadFor(validated, tx)
  )
}

predicate occursAfterGuardBeforeCall(AstNode node, SelectorExpr validated, CallExpr call) {
  node.getEnclosingFunction() = call.getEnclosingFunction() and
  (
    node.getLocation().getStartLine() > validated.getLocation().getStartLine()
    or
    node.getLocation().getStartLine() = validated.getLocation().getStartLine() and
    node.getLocation().getStartColumn() > validated.getLocation().getStartColumn()
  ) and
  (
    node.getLocation().getStartLine() < call.getLocation().getStartLine()
    or
    node.getLocation().getStartLine() = call.getLocation().getStartLine() and
    node.getLocation().getStartColumn() < call.getLocation().getStartColumn()
  )
}

predicate sameLocalVariable(Expr a, Expr b) {
  exists(Entity target |
    a.stripParens().(Ident).refersTo(target) and
    b.stripParens().(Ident).refersTo(target)
  )
}

predicate fieldReadFor(SelectorExpr selector, Expr base, string fieldName) {
  exists(Field field |
    selector.refersTo(field) and
    field.getName() = fieldName and
    (
      globalValueNumber(DataFlow::exprNode(selector.getBase())) = globalValueNumber(DataFlow::exprNode(base))
      or
      sameLocalVariable(selector.getBase(), base)
    )
  )
}

predicate transactionMutatedAfterGuardBeforeCall(Expr tx, SelectorExpr validated, CallExpr call) {
  exists(Assignment assign, Expr lhs, SelectorExpr mutatedField, Field mutatedFieldDecl |
    lhs = assign.getLhs(_) and
    occursAfterGuardBeforeCall(assign, validated, call) and
    (
      sameLocalVariable(lhs, tx)
      or
      mutatedField = lhs.stripParens() and
      mutatedField.refersTo(mutatedFieldDecl) and
      mutatedFieldDecl.getName() = "Validated" and
      sameLocalVariable(mutatedField.getBase(), tx)
    )
  )
}

predicate wrapperMutatedAfterGuardBeforeCall(Expr wrapper, SelectorExpr validated, CallExpr call) {
  exists(Assignment assign, Expr lhs |
    lhs = assign.getLhs(_) and
    occursAfterGuardBeforeCall(assign, validated, call) and
    (
      sameLocalVariable(lhs, call.getArgument(0)) and
      assign.getLocation().getStartLine() > wrapper.getLocation().getStartLine()
      or
      fieldReadFor(lhs.stripParens().(SelectorExpr), call.getArgument(0), "TxResponse")
      or
      fieldReadFor(lhs.stripParens().(SelectorExpr), call.getArgument(0), "Validated")
      or
      exists(SelectorExpr txResponseField |
        fieldReadFor(txResponseField, call.getArgument(0), "TxResponse") and
        fieldReadFor(lhs.stripParens().(SelectorExpr), txResponseField, "Validated")
      )
      or
      sameLocalVariable(lhs, wrapper) and
      assign.getLocation().getStartLine() > wrapper.getLocation().getStartLine()
      or
      fieldReadFor(lhs.stripParens().(SelectorExpr), wrapper, "TxResponse")
      or
      fieldReadFor(lhs.stripParens().(SelectorExpr), wrapper, "Validated")
      or
      exists(SelectorExpr wrappedTxResponseField |
        fieldReadFor(wrappedTxResponseField, wrapper, "TxResponse") and
        fieldReadFor(lhs.stripParens().(SelectorExpr), wrappedTxResponseField, "Validated")
      )
    )
  )
}

predicate proofInvalidatedAfterGuardBeforeCall(Expr tx, SelectorExpr validated, CallExpr call) {
  transactionMutatedAfterGuardBeforeCall(tx, validated, call)
  or
  exists(Expr wrapper | isParseTxResponseWrapperFor(call, wrapper, tx) |
    wrapperMutatedAfterGuardBeforeCall(wrapper, validated, call)
  )
}

predicate callHasValidatedProof(CallExpr call) {
  exists(Expr tx, Expr proof, Expr proofRead, SelectorExpr validated, ControlFlow::ConditionGuardNode guard |
    isParsedTransactionArgument(call, tx) and
    isValidatedProofExprFor(proof, tx, validated) and
    globalValueNumber(DataFlow::exprNode(proofRead)) = globalValueNumber(DataFlow::exprNode(proof)) and
    guard.ensures(DataFlow::exprNode(proofRead), true) and
    guard.dominates(DataFlow::exprNode(call).getBasicBlock()) and
    not proofInvalidatedAfterGuardBeforeCall(tx, validated, call)
  )
}

from CallExpr call
where isParserEntryPoint(call) and not callHasValidatedProof(call)
select call, "XRPL transaction is parsed before proving Validated == true."
