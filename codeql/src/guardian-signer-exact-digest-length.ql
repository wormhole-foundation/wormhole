/**
 * @name Guardian signer signs non-exact digest length
 * @description GuardianSigner.Sign implementations must reject non-32-byte digest input before passing the digest to a local or remote signing primitive.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/guardian-signer-exact-digest-length
 * @tags security
 *       external/cwe/cwe-345
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.dataflow.DataFlow

predicate isProductionGuardianSignerFile(File f) {
  f.getRelativePath().matches("node/pkg/guardiansigner/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate isGuardianSignerSignMethod(MethodDecl m) {
  isProductionGuardianSignerFile(m.getFile()) and
  m.getName() = "Sign" and
  m.getFunction().(Method).implements(
    "github.com/certusone/wormhole/node/pkg/guardiansigner", "GuardianSigner", "Sign"
  ) and
  // Currently documented as an unsafe/test helper and not returned by production URI constructors.
  not m.getReceiverBaseType().getName() = "GeneratedSigner"
}

Parameter signerDigestParam(MethodDecl m) {
  isGuardianSignerSignMethod(m) and
  result = m.getParameter(1)
}

predicate exprReceivesSignerDigest(MethodDecl m, Expr e) {
  DataFlow::localFlow(DataFlow::parameterNode(signerDigestParam(m)), DataFlow::exprNode(e.stripParens()))
  or
  exists(SliceExpr slice |
    e.stripParens() = slice and
    DataFlow::localFlow(
      DataFlow::parameterNode(signerDigestParam(m)), DataFlow::exprNode(slice.getBase().stripParens())
    )
  )
}

predicate isLocalSigningPrimitiveArg(CallExpr call, Expr arg) {
  call.getCalleeName() = "Sign" and
  call.getTarget().getQualifiedName().matches("github.com/ethereum/go-ethereum/crypto.Sign") and
  arg = call.getArgument(0)
}

predicate isKmsSigningPrimitiveArg(CallExpr call, Expr arg) {
  call.getCalleeName() = "Sign" and
  call.getTarget().(Method).hasQualifiedName("github.com/aws/aws-sdk-go-v2/service/kms", "Client", "Sign") and
  exists(CompositeLit input, KeyValueExpr field |
    input = call.getArgument(1).(AddressExpr).getOperand() and
    field = input.getElement(_) and
    field.getKey().(Ident).getName() = "Message" and
    arg = field.getValue()
  )
}

predicate isSigningPrimitiveDigestArg(MethodDecl m, CallExpr call, Expr arg) {
  call.getEnclosingFunction() = m and
  (isLocalSigningPrimitiveArg(call, arg) or isKmsSigningPrimitiveArg(call, arg)) and
  exprReceivesSignerDigest(m, arg)
}

predicate isExact32(Expr e) { e.getIntValue() = 32 or e.getExactValue().toInt() = 32 }

predicate isLenOfSignerDigest(MethodDecl m, CallExpr lenCall) {
  lenCall.getCalleeName() = "len" and
  lenCall.getTarget().getQualifiedName() = "len" and
  exprReceivesSignerDigest(m, lenCall.getArgument(0))
}

predicate isExactLenNeq32Guard(MethodDecl m, IfStmt guard) {
  exists(NeqExpr cmp, CallExpr lenCall, Expr length |
    guard.getEnclosingFunction() = m and
    cmp = guard.getCond().stripParens() and
    isLenOfSignerDigest(m, lenCall) and
    (
      cmp.getLeftOperand() = lenCall and length = cmp.getRightOperand()
      or
      cmp.getRightOperand() = lenCall and length = cmp.getLeftOperand()
    ) and
    isExact32(length)
  )
}

predicate thenBranchFailsClosed(IfStmt guard) {
  exists(ReturnStmt ret |
    ret.getParent*() = guard.getThen() and
    ret.getNumExpr() = 2 and
    ret.getExpr(0).toString() = "nil" and
    not ret.getExpr(1).toString() = "nil"
  )
}

predicate dominates(AstNode before, AstNode after) {
  exists(ControlFlow::Node beforeNode, ControlFlow::Node afterNode |
    beforeNode.isFirstNodeOf(before) and
    afterNode.isFirstNodeOf(after) and
    beforeNode.getBasicBlock().dominates(afterNode.getBasicBlock())
  )
}

predicate hasExactFailClosedDigestGuard(MethodDecl m, CallExpr primitive) {
  exists(IfStmt guard |
    isExactLenNeq32Guard(m, guard) and
    thenBranchFailsClosed(guard) and
    dominates(guard, primitive)
  )
}

from MethodDecl m, CallExpr primitive, Expr arg
where
  isGuardianSignerSignMethod(m) and
  isSigningPrimitiveDigestArg(m, primitive, arg) and
  not hasExactFailClosedDigestGuard(m, primitive)
select m,
  "GuardianSigner.Sign implementation reaches a signing primitive without first rejecting non-32-byte digest input; add a dominating len(hash) == 32 fail-closed check before signing."
