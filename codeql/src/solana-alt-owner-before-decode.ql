/**
 * @name Solana ALT account decoded without owner proof
 * @description Solana watcher code must prove an RPC-fetched address lookup table account exists and is owned by the address lookup table program before decoding its bytes.
 * @kind problem
 * @problem.severity error
 * @precision high
 * @id wormhole/go/solana-alt-owner-before-decode
 * @tags security
 *       external/cwe/cwe-20
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.controlflow.ControlFlowGraph
import semmle.go.dataflow.GlobalValueNumbering

predicate isProductionNodeFile(File f) {
  f.getRelativePath().matches("node/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f.getRelativePath().matches("node/hack/%") and
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

predicate isGetAccountInfoCall(CallExpr call) {
  call.getCalleeName() in ["GetAccountInfo", "GetAccountInfoWithOpts"]
}

predicate rpcAccountInfoResult(Expr info) {
  exists(Assignment assign, CallExpr rpcCall |
    assign.getRhs(0) = rpcCall and
    isGetAccountInfoCall(rpcCall) and
    assign.getLhs(0) = info
  )
}

predicate valueFieldForInfo(Expr value, Expr info) {
  exists(SelectorExpr sel, Field field |
    value.stripParens() = sel and
    sel.refersTo(field) and
    field.getName() = "Value" and
    sameValue(sel.getBase(), info)
  )
}

predicate ownerFieldForInfo(Expr owner, Expr info) {
  exists(SelectorExpr sel, Field field, Expr value |
    owner.stripParens() = sel and
    sel.refersTo(field) and
    field.getName() = "Owner" and
    valueFieldForInfo(value, info) and
    sameValue(sel.getBase(), value)
  )
}

predicate isAltProgramIdExpr(Expr e) { e.stripParens().(Ident).getName() = "addressLookupTableProgramID" }

predicate isOwnerEqualsAltProgramCall(CallExpr call, Expr info) {
  call.getCalleeName() = "Equals" and
  call.getNumArgument() = 1 and
  exists(SelectorExpr callee |
    call.getCalleeExpr().stripParens() = callee and
    (
      ownerFieldForInfo(callee.getBase(), info) and isAltProgramIdExpr(call.getArgument(0))
      or
      isAltProgramIdExpr(callee.getBase()) and ownerFieldForInfo(call.getArgument(0), info)
    )
  )
}

predicate isGetBinaryOnInfo(CallExpr call, Expr info) {
  call.getCalleeName() = "GetBinary" and
  exists(SelectorExpr callee |
    call.getCalleeExpr().stripParens() = callee and
    sameValue(callee.getBase(), info)
  )
}

predicate isAltDecodeSink(CallExpr sink, Expr info) {
  isProductionNodeFile(sink.getFile()) and
  sink.getCalleeName() = "DecodeAddressLookupTableState" and
  sink.getNumArgument() = 1 and
  rpcAccountInfoResult(info) and
  exists(CallExpr getBinary |
    sink.getArgument(0).stripParens() = getBinary and
    isGetBinaryOnInfo(getBinary, info)
  )
}

predicate assignmentToSameLocal(Assignment assign, Expr local) {
  exists(Ident lhs, Ident localIdent, Variable v |
    assign.getLhs(_) = lhs and
    local.stripParens() = localIdent and
    lhs.refersTo(v) and
    localIdent.refersTo(v)
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

predicate hasNilProofBefore(Expr info, AstNode sink) {
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node sinkNode, Expr valueRead, Expr nil |
    exprRefersToNil(nil) and
    valueFieldForInfo(valueRead, info) and
    sinkNode.isFirstNodeOf(sink) and
    guard.ensuresNeq(DataFlow::exprNode(valueRead), DataFlow::exprNode(nil)) and
    guard.dominates(sinkNode.getBasicBlock()) and
    not localReassignedBetween(info, valueRead, sink)
  )
}

predicate hasOwnerProofBefore(Expr info, AstNode sink) {
  exists(ControlFlow::ConditionGuardNode guard, ControlFlow::Node sinkNode, CallExpr equalsCall |
    isOwnerEqualsAltProgramCall(equalsCall, info) and
    sinkNode.isFirstNodeOf(sink) and
    guard.ensures(DataFlow::exprNode(equalsCall), true) and
    guard.dominates(sinkNode.getBasicBlock()) and
    not localReassignedBetween(info, equalsCall, sink)
  )
}

from CallExpr sink, Expr info
where
  isAltDecodeSink(sink, info) and
  (not hasNilProofBefore(info, sink) or not hasOwnerProofBefore(info, sink))
select sink,
  "Solana ALT account data from RPC is decoded or used before proving the same account result is non-nil and owned by the address lookup table program; add fail-closed Value != nil and Owner.Equals(addressLookupTableProgramID) checks before decoding or resolving lookups."
