/**
 * @name EVM CCL handling mutates signed message fields
 * @description EVM custom-consistency-level handling must preserve signed MessagePublication fields after observation and update only release metadata such as effectiveCL or additionalBlocks.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/evm-ccl-signed-message-immutability
 * @tags security
 *       external/cwe/cwe-345
 */

import go
import semmle.go.concepts.GeneratedFile

predicate isProductionEvmWatcherFile(File f) {
  f.getRelativePath().matches("node/pkg/watchers/evm/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate isCclHandle(MethodDecl m) {
  isProductionEvmWatcherFile(m.getFile()) and
  m.getName() = "cclHandleMessage" and
  m.getReceiverBaseType().hasQualifiedName(
    "github.com/certusone/wormhole/node/pkg/watchers/evm", "Watcher"
  )
}

predicate sameVariable(Expr a, Expr b) {
  exists(Entity target |
    a.stripParens().(Ident).refersTo(target) and
    b.stripParens().(Ident).refersTo(target)
  )
}

bindingset[earlier]
bindingset[later]
pragma[inline]
predicate occursBefore(AstNode earlier, AstNode later) {
  exists(string path, int earlierLine, int earlierColumn, int laterLine, int laterColumn |
    earlier.getLocation().hasLocationInfo(path, earlierLine, earlierColumn, _, _) and
    later.getLocation().hasLocationInfo(path, laterLine, laterColumn, _, _) and
    earlier.getEnclosingFunction() = later.getEnclosingFunction() and
    (earlierLine < laterLine or earlierLine = laterLine and earlierColumn < laterColumn)
  )
}

predicate variableIsNotReassignedBetween(Expr variable, Assignment source, AstNode use) {
  not exists(Assignment overwrite, Expr lhs |
    lhs = overwrite.getLhs(_) and
    sameVariable(variable, lhs) and
    overwrite.getEnclosingFunction() = use.getEnclosingFunction() and
    occursBefore(source, overwrite) and
    occursBefore(overwrite, use)
  )
}

predicate isCclHandlePendingParameter(Expr e, AstNode use) {
  exists(MethodDecl m, Parameter pe |
    isCclHandle(m) and
    pe = m.getParameter(1) and
    use.getEnclosingFunction() = m and
    e.stripParens().(Ident).refersTo(pe)
  )
}

predicate isHelperParameterFromProtectedPending(Expr e, AstNode use) {
  exists(CallExpr call, Parameter p |
    call.getTarget().getFuncDecl() = p.getFunction() and
    p.getIndex() >= 0 and
    isCclHandle(call.getEnclosingFunction()) and
    isPendingRootAt(call.getArgument(p.getIndex()), call) and
    use.getEnclosingFunction() = p.getFunction() and
    e.stripParens().(Ident).refersTo(p)
  )
}

predicate isHelperParameterFromProtectedMessage(Expr e, AstNode use) {
  exists(CallExpr call, Parameter p |
    call.getTarget().getFuncDecl() = p.getFunction() and
    p.getIndex() >= 0 and
    isCclHandle(call.getEnclosingFunction()) and
    isMessageRootAt(call.getArgument(p.getIndex()), call) and
    use.getEnclosingFunction() = p.getFunction() and
    e.stripParens().(Ident).refersTo(p)
  )
}

predicate isHelperParameterFromProtectedPayload(Expr e, AstNode use) {
  exists(CallExpr call, Parameter p |
    call.getTarget().getFuncDecl() = p.getFunction() and
    p.getIndex() >= 0 and
    isCclHandle(call.getEnclosingFunction()) and
    isPayloadRootAt(call.getArgument(p.getIndex()), call) and
    use.getEnclosingFunction() = p.getFunction() and
    e.stripParens().(Ident).refersTo(p)
  )
}

predicate isPendingRootAt(Expr e, AstNode use) {
  isCclHandlePendingParameter(e, use)
  or
  isHelperParameterFromProtectedPending(e, use)
  or
  exists(Assignment assign, Expr lhs, Expr rhs |
    lhs = assign.getLhs(_) and
    rhs = assign.getRhs(_) and
    sameVariable(e, lhs) and
    assign.getEnclosingFunction() = use.getEnclosingFunction() and
    occursBefore(assign, use) and
    variableIsNotReassignedBetween(e, assign, use) and
    isPendingRootAt(rhs, assign)
  )
}

predicate isMessageField(Expr e, Expr pending) {
  exists(SelectorExpr sel |
    e.stripParens() = sel and
    sel.getSelector().getName() = "message" and
    sameVariable(sel.getBase(), pending)
  )
}

predicate isMessageRootAt(Expr e, AstNode use) {
  isHelperParameterFromProtectedMessage(e, use)
  or
  exists(Expr pending |
    isPendingRootAt(pending, use) and
    isMessageField(e, pending)
  )
  or
  exists(Assignment assign, Expr lhs, Expr rhs |
    lhs = assign.getLhs(_) and
    rhs = assign.getRhs(_) and
    sameVariable(e, lhs) and
    assign.getEnclosingFunction() = use.getEnclosingFunction() and
    occursBefore(assign, use) and
    variableIsNotReassignedBetween(e, assign, use) and
    isMessageRootAt(rhs, assign)
  )
}

predicate isPayloadField(Expr e, Expr message) {
  exists(SelectorExpr sel |
    e.stripParens() = sel and
    sel.getSelector().getName() = "Payload" and
    (sel.getBase().stripParens() = message.stripParens() or sameVariable(sel.getBase(), message))
  )
}

predicate isPayloadRootAt(Expr e, AstNode use) {
  isHelperParameterFromProtectedPayload(e, use)
  or
  exists(Expr message |
    isMessageRootAt(message, use) and
    isPayloadField(e, message)
  )
  or
  exists(Assignment assign, Expr lhs, Expr rhs |
    lhs = assign.getLhs(_) and
    rhs = assign.getRhs(_) and
    sameVariable(e, lhs) and
    assign.getEnclosingFunction() = use.getEnclosingFunction() and
    occursBefore(assign, use) and
    variableIsNotReassignedBetween(e, assign, use) and
    (
      isPayloadRootAt(rhs, assign)
      or exists(SliceExpr slice | rhs.stripParens() = slice and isPayloadRootAt(slice.getBase(), assign))
    )
  )
}

predicate isSignedField(string field) {
  field in ["Timestamp", "Nonce", "EmitterChain", "EmitterAddress", "Payload", "Sequence", "ConsistencyLevel"]
}

predicate signedFieldWrite(AstNode sink, string field) {
  exists(Assignment assign, SelectorExpr lhs |
    sink = lhs and
    lhs = assign.getLhs(_).stripParens() and
    field = lhs.getSelector().getName() and
    isSignedField(field) and
    isMessageRootAt(lhs.getBase(), assign)
  )
  or
  exists(IncDecStmt inc, SelectorExpr operand |
    sink = operand and
    operand = inc.getOperand().stripParens() and
    field = operand.getSelector().getName() and
    isSignedField(field) and
    isMessageRootAt(operand.getBase(), inc)
  )
  or
  exists(Assignment assign, SelectorExpr lhs |
    sink = lhs and
    lhs = assign.getLhs(_).stripParens() and
    lhs.getSelector().getName() = "message" and
    isPendingRootAt(lhs.getBase(), assign) and
    field = "MessagePublication"
  )
}

predicate payloadByteWrite(AstNode sink) {
  exists(Assignment assign, IndexExpr lhs |
    sink = lhs and
    lhs = assign.getLhs(_).stripParens() and
    isPayloadRootAt(lhs.getBase(), assign)
  )
  or
  exists(CallExpr call |
    sink = call and
    call.getCalleeName() = "copy" and
    isPayloadRootAt(call.getArgument(0), call)
  )
}

from AstNode sink, string field
where
  signedFieldWrite(sink, field)
  or payloadByteWrite(sink) and field = "Payload bytes"
select sink,
  "CCL handling must not mutate signed MessagePublication field '" + field +
    "' after observation; update release metadata such as effectiveCL/additionalBlocks instead so guardians sign the original EVM message body."
