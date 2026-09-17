/**
 * @name Delegate observation quorum bucket keyed by non-canonical digest
 * @description Delegate observation quorum buckets must be keyed by the reconstructed MessagePublication VAA signing digest, not by serialized observations, per-guardian metadata, or composite keys.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/delegate-consensus-canonical-digest
 * @tags security
 *       external/cwe/cwe-345
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.dataflow.DataFlow

predicate isProductionProcessorFile(File f) {
  f.getRelativePath().matches("node/pkg/processor/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate isDelegateStateObservationsSelector(Expr e) {
  exists(SelectorExpr observations, SelectorExpr delegateState |
    e.stripParens() = observations and
    observations.getSelector().getName() = "observations" and
    delegateState = observations.getBase().stripParens() and
    delegateState.getSelector().getName() = "delegateState"
  )
}

predicate isDelegateBucketBase(Expr base) {
  isDelegateStateObservationsSelector(base)
  or
  exists(Expr source |
    isDelegateStateObservationsSelector(source) and
    DataFlow::localFlow(DataFlow::exprNode(source), DataFlow::exprNode(base.stripParens()))
  )
  or
  exists(SelectorExpr observations, MethodDecl method |
    base.stripParens() = observations and
    observations.getSelector().getName() = "observations" and
    observations.getEnclosingFunction() = method and
    method.getReceiverBaseType().getName() = "delegateState"
  )
}

predicate isDelegateBucketIndex(IndexExpr idx) {
  isProductionProcessorFile(idx.getFile()) and
  isDelegateBucketBase(idx.getBase())
}

predicate isMessagePublicationMethod(CallExpr call, string name) {
  call.getCalleeName() = name and
  call.getTarget().getPackage().getPath().matches("%/node/pkg/common")
}

predicate isCreateDigestCall(CallExpr call) {
  isMessagePublicationMethod(call, "CreateDigest")
}

predicate isCreateVaaCall(CallExpr call) {
  isMessagePublicationMethod(call, "CreateVAA")
}

predicate isCanonicalSigningDigestCall(CallExpr signing) {
  signing.getCalleeName() = "SigningDigest" and
  signing.getTarget().getPackage().getPath().matches("%/node/pkg/common") and
  exists(SelectorExpr callee, CallExpr createVaa |
    signing.getCalleeExpr().stripParens() = callee and
    isCreateVaaCall(createVaa) and
    DataFlow::localFlow(
      DataFlow::exprNode(createVaa), DataFlow::exprNode(callee.getBase().stripParens())
    )
  )
}

predicate isEncodedCanonicalSigningDigest(CallExpr encoded) {
  encoded.getCalleeName() = "EncodeToString" and
  exists(CallExpr bytes, SelectorExpr bytesCallee, CallExpr signing |
    encoded.getArgument(0).stripParens() = bytes and
    bytes.getCalleeName() = "Bytes" and
    bytes.getCalleeExpr().stripParens() = bytesCallee and
    bytesCallee.getBase().stripParens() = signing and
    isCanonicalSigningDigestCall(signing)
  )
}

predicate canonicalDigestExpr(Expr e) {
  exists(CallExpr call | e.stripParens() = call and isCreateDigestCall(call))
  or
  exists(CallExpr call | e.stripParens() = call and isEncodedCanonicalSigningDigest(call))
  or
  exists(CallExpr helper, FuncDecl target, ReturnStmt ret |
    e.stripParens() = helper and
    helper.getTarget().getFuncDecl() = target and
    ret.getEnclosingFunction() = target and
    ret.getNumExpr() = 1 and
    canonicalDigestExpr(ret.getExpr(0))
  )
}

predicate hasAllowedDigestFlow(Expr key) {
  exists(CallExpr call |
    canonicalDigestExpr(call) and
    DataFlow::localFlow(DataFlow::exprNode(call), DataFlow::exprNode(key.stripParens()))
  )
}

predicate isDelegateBucketKeyHelper(CallExpr call, Expr key) {
  exists(MethodDecl method, Parameter keyParam, IndexExpr idx, Ident keyUse |
    call.getTarget().getFuncDecl() = method and
    method.getReceiverBaseType().getName() = "delegateState" and
    keyParam = method.getParameter(0) and
    idx.getEnclosingFunction() = method and
    isDelegateBucketBase(idx.getBase()) and
    idx.getIndex().stripParens() = keyUse and
    keyUse.refersTo(keyParam) and
    key = call.getArgument(0)
  )
}

predicate isDelegateBucketKeyHelperParameter(IndexExpr idx, Expr key) {
  exists(MethodDecl method, Parameter keyParam, Ident keyUse |
    idx.getEnclosingFunction() = method and
    method.getReceiverBaseType().getName() = "delegateState" and
    keyParam = method.getParameter(0) and
    key.stripParens() = keyUse and
    keyUse.refersTo(keyParam)
  )
}

predicate isUnsafeDigestOrSerializationCall(CallExpr call) {
  call.getCalleeName() in [
      "MarshalBinary", "Marshal", "Keccak256Hash", "MessageIDString", "Sprintf", "Sprint", "Sprintln"
    ]
}

predicate isNonVaaFieldRead(SelectorExpr sel) {
  sel.getSelector().getName() in [
      "IsReobservation", "Unreliable", "verificationState", "TxID", "TxHash", "GuardianAddr",
      "GuardianAddress", "Signature", "Signatures"
    ]
}

predicate hasUnsafeKeyFlow(Expr key) {
  exists(CallExpr call |
    isUnsafeDigestOrSerializationCall(call) and
    DataFlow::localFlow(DataFlow::exprNode(call), DataFlow::exprNode(key.stripParens()))
  )
  or
  exists(SelectorExpr sel |
    isNonVaaFieldRead(sel) and
    DataFlow::localFlow(DataFlow::exprNode(sel), DataFlow::exprNode(key.stripParens()))
  )
  or
  exists(AddExpr added |
    hasAllowedDigestFlow(added.getAnOperand()) and
    DataFlow::localFlow(DataFlow::exprNode(added), DataFlow::exprNode(key.stripParens()))
  )
  or
  exists(Assignment assign, Ident lhs, Ident keyIdent, Variable v, CallExpr unsafe |
    assign.getLhs(_) = lhs and
    key.stripParens() = keyIdent and
    lhs.refersTo(v) and
    keyIdent.refersTo(v) and
    isUnsafeDigestOrSerializationCall(unsafe) and
    assign.getRhs(_).getAChild*() = unsafe and
    assign.getLocation().getStartLine() < key.getLocation().getStartLine()
  )
}

from AstNode sink, Expr key
where
  (
    exists(IndexExpr idx |
      sink = key and
      isDelegateBucketIndex(idx) and
      key = idx.getIndex() and
      not isDelegateBucketKeyHelperParameter(idx, key)
    )
    or
    exists(CallExpr call | sink = key and isDelegateBucketKeyHelper(call, key))
  ) and
  (not hasAllowedDigestFlow(key) or hasUnsafeKeyFlow(key))
select sink,
  "Delegate observation quorum bucket key must be the reconstructed MessagePublication VAA signing digest; use CreateDigest or equivalent SigningDigest and exclude per-guardian/non-VAA fields such as IsReobservation, TxID, guardian address, signatures, and serialized delegate observations."
