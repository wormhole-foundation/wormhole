/**
 * @name Governance VAA payload must come from checked typed serializer
 * @description Production governance VAA construction must pass CreateGovernanceVAA a payload produced by a checked SDK typed governance serializer or EmptyPayloadVaa.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/governance-vaa-typed-payload
 * @tags security
 *       external/cwe/cwe-20
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.dataflow.DataFlow
import semmle.go.dataflow.GlobalValueNumbering

predicate inNodeProduction(AstNode n) {
  n.getFile().getRelativePath().matches("node/%.go") and
  not n.getFile().getRelativePath().matches("%_test.go") and
  not n.getFile() instanceof GeneratedFile
}

predicate isCreateGovernanceVaa(CallExpr call) {
  call.getTarget().getPackage().getPath() = "github.com/wormhole-foundation/wormhole/sdk/vaa" and
  call.getTarget().getName() = "CreateGovernanceVAA"
}

predicate isEmptyPayloadVaa(CallExpr call) {
  call.getTarget().getPackage().getPath() = "github.com/wormhole-foundation/wormhole/sdk/vaa" and
  call.getTarget().getName() = "EmptyPayloadVaa"
}

predicate isTypedGovernanceSerialize(CallExpr call) {
  exists(Method m, string receiver |
    m = call.getTarget() and
    m.getPackage().getPath() = "github.com/wormhole-foundation/wormhole/sdk/vaa" and
    m.getName() = "Serialize" and
    receiver = m.getReceiverBaseType().getName() and
    receiver.matches("Body%") and
    not receiver.matches("%VAA%")
  )
}

predicate sameExpr(Expr a, Expr b) {
  globalValueNumber(DataFlow::exprNode(a.stripParens())) =
    globalValueNumber(DataFlow::exprNode(b.stripParens()))
  or
  exists(Entity e | a.stripParens().(Ident).refersTo(e) and b.stripParens().(Ident).refersTo(e))
}

predicate assignedTuple(CallExpr producer, Expr bytesLhs, Expr errLhs) {
  exists(Assignment assign |
    assign.getRhs(0) = producer and
    bytesLhs = assign.getLhs(0) and
    errLhs = assign.getLhs(1)
  )
}

predicate isGovernancePayloadProducer(CallExpr producer) {
  isTypedGovernanceSerialize(producer)
  or
  isEmptyPayloadVaa(producer)
}

predicate payloadFlowsToSink(CallExpr producer, CallExpr sink) {
  DataFlow::localFlow(
    DataFlow::extractTupleElement(DataFlow::exprNode(producer), 0),
    DataFlow::exprNode(sink.getArgument(4))
  )
}

predicate errIsIgnored(Expr errLhs) { errLhs.toString() = "_" }

predicate errGuardDominates(Expr errLhs, CallExpr sink) {
  exists(IfStmt guard |
    sameExpr(errLhs, any(Expr e | e = guard.getCond().getAChild*())) and
    guard.getThen().getAChild*() instanceof ReturnStmt and
    DataFlow::exprNode(guard.getCond()).getBasicBlock().dominates(
      DataFlow::exprNode(sink).getBasicBlock()
    )
  )
}

predicate occursBetween(AstNode middle, AstNode first, AstNode last) {
  middle.getEnclosingFunction() = last.getEnclosingFunction() and
  first.getLocation().getStartLine() < middle.getLocation().getStartLine() and
  middle.getLocation().getStartLine() < last.getLocation().getStartLine()
}

predicate payloadReassignedBetween(Expr bytesLhs, CallExpr producer, CallExpr sink) {
  exists(Assignment later |
    occursBetween(later, producer, sink) and
    later.getRhs(0) != producer and
    sameExpr(bytesLhs, later.getLhs(_))
  )
}

predicate payloadMutatedBetween(Expr bytesLhs, CallExpr producer, CallExpr sink) {
  exists(CallExpr mutation |
    occursBetween(mutation, producer, sink) and
    (
      mutation.getCalleeName() = "append" and sameExpr(bytesLhs, mutation.getArgument(0))
      or
      mutation.getCalleeName() = "copy" and sameExpr(bytesLhs, mutation.getArgument(0))
    )
  )
}

predicate hasCheckedStablePayload(CallExpr sink) {
  exists(CallExpr producer, Expr bytesLhs, Expr errLhs |
    isGovernancePayloadProducer(producer) and
    assignedTuple(producer, bytesLhs, errLhs) and
    payloadFlowsToSink(producer, sink) and
    not errIsIgnored(errLhs) and
    errGuardDominates(errLhs, sink) and
    not payloadReassignedBetween(bytesLhs, producer, sink) and
    not payloadMutatedBetween(bytesLhs, producer, sink)
  )
}

predicate hasUncheckedSerializerPayload(CallExpr sink) {
  exists(CallExpr producer, Expr bytesLhs, Expr errLhs |
    isGovernancePayloadProducer(producer) and
    assignedTuple(producer, bytesLhs, errLhs) and
    payloadFlowsToSink(producer, sink) and
    (
      errIsIgnored(errLhs)
      or
      not errGuardDominates(errLhs, sink)
    )
  )
}

from CallExpr sink, string message
where
  inNodeProduction(sink) and
  isCreateGovernanceVaa(sink) and
  not hasCheckedStablePayload(sink) and
  (
    hasUncheckedSerializerPayload(sink) and
    message =
      "check the governance payload serializer error before passing its bytes to CreateGovernanceVAA"
    or
    not hasUncheckedSerializerPayload(sink) and
    message =
      "governance VAA payload must come from a checked SDK typed serializer or EmptyPayloadVaa before CreateGovernanceVAA"
  )
select sink, message
