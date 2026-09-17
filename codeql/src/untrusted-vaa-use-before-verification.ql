/**
 * @name Parsed VAA used before quorum verification
 * @description Parsed signed VAAs from untrusted node boundaries must be verified with the complete guardian set before storage or external delivery; vaa.Unmarshal only checks wire format.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/untrusted-vaa-use-before-verification
 * @tags security
 *       external/cwe/cwe-345
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.controlflow.ControlFlowGraph
import semmle.go.dataflow.DataFlow

predicate isProductionNodeFile(File f) {
  f.getRelativePath().matches("%node/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

bindingset[earlier, later]
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

predicate isVaaUnmarshal(CallExpr call) {
  isProductionNodeFile(call.getFile()) and
  call.getTarget().getName() = "Unmarshal" and
  call.getTarget().getPackage().getName() = "vaa"
}

predicate unmarshalResult(CallExpr unmarshal, Expr e) {
  isVaaUnmarshal(unmarshal) and
  DataFlow::localFlow(DataFlow::extractTupleElement(DataFlow::exprNode(unmarshal), 0), DataFlow::exprNode(e))
}

predicate sameObjectVerify(CallExpr verify, CallExpr unmarshal) {
  isProductionNodeFile(verify.getFile()) and
  exists(SelectorExpr callee |
    verify.getCalleeExpr().stripParens() = callee and
    callee.getSelector().getName() = "Verify" and
    unmarshalResult(unmarshal, callee.getBase())
  )
}

predicate verifySuccessReturnsOnError(CallExpr verify) {
  exists(IfStmt ifs |
    ifs.getInit().getAChild*() = verify and
    ifs.getThen().getAChild*() instanceof ReturnStmt
  )
  or
  exists(IfStmt ifs |
    ifs.getCond().getAChild*() = verify and
    ifs.getThen().getAChild*() instanceof ReturnStmt
  )
}

predicate directSameObjectProofBefore(CallExpr unmarshal, AstNode sink) {
  exists(CallExpr verify, ControlFlow::Node sinkNode |
    sameObjectVerify(verify, unmarshal) and
    verifySuccessReturnsOnError(verify) and
    occursBefore(verify, sink) and
    sinkNode.isFirstNodeOf(sink) and
    DataFlow::exprNode(verify).getBasicBlock().dominates(sinkNode.getBasicBlock())
  )
}

predicate spyHelperProofBefore(CallExpr unmarshal, AstNode sink) {
  exists(CallExpr helper, ControlFlow::Node sinkNode |
    helper.getFile().getRelativePath() = "node/cmd/spy/spy.go" and
    helper.getCalleeName() = "verifyVAA" and
    helper.getNumArgument() >= 1 and
    helper.getEnclosingFunction() = sink.getEnclosingFunction() and
    unmarshalResult(unmarshal, helper.getArgument(0)) and
    occursBefore(helper, sink) and
    sinkNode.isFirstNodeOf(sink) and
    DataFlow::exprNode(helper).getBasicBlock().dominates(sinkNode.getBasicBlock())
  )
}

predicate hasVerificationProofBefore(CallExpr unmarshal, AstNode sink) {
  directSameObjectProofBefore(unmarshal, sink)
  or
  spyHelperProofBefore(unmarshal, sink)
}

predicate isSignedVaaStorageSink(CallExpr sink, CallExpr unmarshal) {
  isProductionNodeFile(sink.getFile()) and
  sink.getTarget().getName() in ["storeSignedVAA", "StoreSignedVAA", "StoreSignedVAABatch"] and
  exists(int i |
    i >= 0 and i < sink.getNumArgument() and
    unmarshalResult(unmarshal, sink.getArgument(i))
  )
}

predicate isSpySubscriberSendSink(SendStmt sink, CallExpr unmarshal) {
  sink.getFile().getRelativePath() = "node/cmd/spy/spy.go" and
  unmarshal.getEnclosingFunction() = sink.getEnclosingFunction() and
  occursBefore(unmarshal, sink)
}

predicate isSecuritySink(AstNode sink, CallExpr unmarshal, string sinkKind) {
  exists(CallExpr call |
    sink = call and
    isSignedVaaStorageSink(call, unmarshal) and
    sinkKind = "stores or queues the parsed signed VAA"
  )
  or
  exists(SendStmt send |
    sink = send and
    isSpySubscriberSendSink(send, unmarshal) and
    sinkKind = "delivers signed VAA bytes to an external spy subscriber"
  )
}

from AstNode sink, CallExpr unmarshal, string sinkKind
where
  isSecuritySink(sink, unmarshal, sinkKind) and
  not hasVerificationProofBefore(unmarshal, sink)
select sink,
  "Verify this parsed VAA with the complete guardian set before treating it as authenticated; vaa.Unmarshal only checks wire format, and this sink " +
    sinkKind + " before quorum verification."
