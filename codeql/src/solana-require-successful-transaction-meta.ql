/**
 * @name Solana transaction parsed without successful metadata proof
 * @description Solana watcher transaction paths must validate transaction metadata is present and successful before reading metadata, extracting transactions, or processing observations.
 * @kind problem
 * @problem.severity error
 * @precision high
 * @id wormhole/go/solana-require-successful-transaction-meta
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

predicate sameLocalVariable(Expr a, Expr b) {
  exists(Ident ai, Ident bi, Variable v |
    a.stripParens() = ai and
    b.stripParens() = bi and
    ai.refersTo(v) and
    bi.refersTo(v)
  )
}

predicate sameValue(Expr a, Expr b) {
  globalValueNumber(DataFlow::exprNode(a.stripParens())) =
    globalValueNumber(DataFlow::exprNode(b.stripParens()))
  or
  DataFlow::localFlow(DataFlow::exprNode(a.stripParens()), DataFlow::exprNode(b.stripParens()))
  or
  DataFlow::localFlow(DataFlow::exprNode(b.stripParens()), DataFlow::exprNode(a.stripParens()))
  or
  sameLocalVariable(a, b)
}

predicate before(AstNode earlier, AstNode later) {
  earlier.getLocation().getStartLine() < later.getLocation().getStartLine()
  or
  earlier.getLocation().getStartLine() = later.getLocation().getStartLine() and
  earlier.getLocation().getStartColumn() < later.getLocation().getStartColumn()
}

predicate fieldReadFor(SelectorExpr selector, Expr base, string fieldName) {
  exists(Field field |
    selector.refersTo(field) and
    field.getName() = fieldName and
    sameValue(selector.getBase(), base)
  )
}

predicate sameFieldRead(Expr a, Expr b, string fieldName) {
  exists(SelectorExpr asel, SelectorExpr bsel, Field afield, Field bfield |
    a.stripParens() = asel and
    b.stripParens() = bsel and
    asel.refersTo(afield) and
    bsel.refersTo(bfield) and
    afield.getName() = fieldName and
    bfield.getName() = fieldName and
    sameValue(asel.getBase(), bsel.getBase())
  )
}

predicate sameMetaValue(Expr a, Expr b) {
  sameValue(a, b)
  or
  sameFieldRead(a, b, "Meta")
}

predicate isProcessTransactionSink(CallExpr call, Expr meta) {
  isProductionSolanaWatcherFile(call.getFile()) and
  call.getTarget().getFuncDecl().getName() = "processTransaction" and
  (
    call.getNumArgument() > 3 and
    meta = call.getArgument(3)
    or
    call.getNumArgument() = 2 and
    meta = call.getArgument(1)
  )
}

predicate isTransactionExtractionSink(CallExpr call) {
  isProductionSolanaWatcherFile(call.getFile()) and
  call.getCalleeName() in ["GetTransaction", "GetParsedTransaction"] and
  exists(SelectorExpr callee, SelectorExpr transactionField |
    call.getCalleeExpr().stripParens() = callee and
    callee.getBase().stripParens() = transactionField and
    exists(Expr response | fieldReadFor(transactionField, response, "Transaction"))
  )
  or
  isProductionSolanaWatcherFile(call.getFile()) and
  call.getCalleeName() in ["GetTransaction", "GetParsedTransaction"] and
  exists(SelectorExpr callee |
    call.getCalleeExpr().stripParens() = callee and
    callee.getBase().getType().getName().matches("%Transaction%")
  )
}

predicate extractionResponse(CallExpr call, Expr response) {
  exists(SelectorExpr callee, SelectorExpr transactionField |
    isTransactionExtractionSink(call) and
    call.getCalleeExpr().stripParens() = callee and
    callee.getBase().stripParens() = transactionField and
    fieldReadFor(transactionField, response, "Transaction")
  )
  or
  exists(SelectorExpr callee |
    isTransactionExtractionSink(call) and
    call.getCalleeExpr().stripParens() = callee and
    response = callee.getBase()
  )
}

predicate isMetadataUseSink(SelectorExpr use, Expr meta) {
  isProductionSolanaWatcherFile(use.getFile()) and
  exists(Field field |
    use.refersTo(field) and
    field.getName() in [
      "LogMessages", "InnerInstructions", "PreBalances", "PostBalances", "PreTokenBalances",
      "PostTokenBalances", "Rewards", "LoadedAddresses", "ReturnData", "ComputeUnitsConsumed"
    ] and
    meta = use.getBase()
  )
}

predicate isSink(AstNode sink, Expr meta) {
  exists(CallExpr call | sink = call and isProcessTransactionSink(call, meta))
  or
  exists(SelectorExpr use | sink = use and isMetadataUseSink(use, meta))
}

predicate assignmentToSameLocal(Assignment assign, Expr meta) {
  exists(Ident lhs, Ident metaIdent, Variable v |
    assign.getLhs(_) = lhs and
    meta.stripParens() = metaIdent and
    lhs.refersTo(v) and
    metaIdent.refersTo(v)
  )
}

predicate metaReassignedBetween(Expr meta, AstNode proof, AstNode sink) {
  exists(Assignment assign |
    assignmentToSameLocal(assign, meta) and
    assign.getEnclosingFunction() = sink.getEnclosingFunction() and
    before(proof, assign) and
    before(assign, sink)
  )
}

predicate localReassignedBetween(Expr local, AstNode proof, AstNode sink) {
  exists(Assignment assign |
    assignmentToSameLocal(assign, local) and
    assign.getEnclosingFunction() = sink.getEnclosingFunction() and
    before(proof, assign) and
    before(assign, sink)
  )
}

predicate assignmentReceivesTupleElement(Assignment assign, CallExpr call, int index, Expr lhs) {
  assign.getRhs(0) = call and
  lhs = assign.getLhs(index)
}

predicate pairedValidatorErrorExpr(CallExpr call, Expr err) {
  exists(Assignment assign |
    assign.getEnclosingFunction() = call.getEnclosingFunction() and
    assignmentReceivesTupleElement(assign, call, 0, err)
  )
}

predicate isValidateTransactionMetaCall(CallExpr call, Expr meta) {
  isProductionSolanaWatcherFile(call.getFile()) and
  call.getTarget().getFuncDecl().getName() = "validateTransactionMeta" and
  call.getNumArgument() = 1 and
  sameMetaValue(call.getArgument(0), meta)
}

predicate guardProvesErrorNilBefore(Expr err, AstNode use) {
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node useNode, Expr errRead, Expr nil |
    exprRefersToNil(nil) and
    useNode.isFirstNodeOf(use) and
    sameValue(errRead, err) and
    guard.ensuresEq(DataFlow::exprNode(errRead), DataFlow::exprNode(nil)) and
    guard.dominates(useNode.getBasicBlock())
  )
  or
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node useNode, NeqExpr neq, Expr errRead, Expr nil |
    exprRefersToNil(nil) and
    useNode.isFirstNodeOf(use) and
    (
      neq.getLeftOperand() = errRead and neq.getRightOperand() = nil
      or
      neq.getRightOperand() = errRead and neq.getLeftOperand() = nil
    ) and
    sameValue(errRead, err) and
    guard.ensures(DataFlow::exprNode(neq), false) and
    guard.dominates(useNode.getBasicBlock())
  )
}

predicate hasValidatorProofBefore(Expr meta, AstNode sink) {
  exists(CallExpr validator, Expr err |
    isValidateTransactionMetaCall(validator, meta) and
    before(validator, sink) and
    pairedValidatorErrorExpr(validator, err) and
    guardProvesErrorNilBefore(err, sink) and
    not metaReassignedBetween(meta, validator, sink) and
    not localReassignedBetween(err, validator, sink)
  )
}

predicate directNilProofBefore(Expr meta, AstNode sink, AstNode proof) {
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node sinkNode, NeqExpr neq, Expr nil, Expr metaRead |
    exprRefersToNil(nil) and
    proof = neq and
    sinkNode.isFirstNodeOf(sink) and
    (
      neq.getLeftOperand() = metaRead and neq.getRightOperand() = nil
      or
      neq.getRightOperand() = metaRead and neq.getLeftOperand() = nil
    ) and
    sameValue(metaRead, meta) and
    guard.ensures(DataFlow::exprNode(neq), true) and
    guard.dominates(sinkNode.getBasicBlock())
  )
}

predicate directErrNilProofBefore(Expr meta, AstNode sink, AstNode proof) {
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node sinkNode, EqExpr eq, Expr nil, SelectorExpr errRead |
    exprRefersToNil(nil) and
    proof = eq and
    sinkNode.isFirstNodeOf(sink) and
    (
      eq.getLeftOperand() = errRead and eq.getRightOperand() = nil
      or
      eq.getRightOperand() = errRead and eq.getLeftOperand() = nil
    ) and
    fieldReadFor(errRead, meta, "Err") and
    guard.ensures(DataFlow::exprNode(eq), true) and
    guard.dominates(sinkNode.getBasicBlock())
  )
}

predicate hasDirectEquivalentProofBefore(Expr meta, AstNode sink) {
  exists(AstNode nilProof, AstNode errProof |
    directNilProofBefore(meta, sink, nilProof) and
    directErrNilProofBefore(meta, sink, errProof) and
    not metaReassignedBetween(meta, nilProof, sink) and
    not metaReassignedBetween(meta, errProof, sink)
  )
}

predicate hasSuccessfulMetaProofBefore(Expr meta, AstNode sink) {
  hasValidatorProofBefore(meta, sink)
  or
  hasDirectEquivalentProofBefore(meta, sink)
}

predicate hasExtractionMetaProofBefore(CallExpr call) {
  exists(Expr response, SelectorExpr meta |
    extractionResponse(call, response) and
    fieldReadFor(meta, response, "Meta") and
    hasSuccessfulMetaProofBefore(meta, call)
  )
}

from AstNode sink
where
  exists(Expr meta | isSink(sink, meta) and not hasSuccessfulMetaProofBefore(meta, sink))
  or
  exists(CallExpr call | sink = call and isTransactionExtractionSink(call) and not hasExtractionMetaProofBefore(call))
select sink, "Solana transaction metadata must be validated successful before transaction parsing or metadata use."
