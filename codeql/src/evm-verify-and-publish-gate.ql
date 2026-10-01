/**
 * @name EVM watcher publication bypasses verifyAndPublish
 * @description EVM watcher MessagePublication values must be published only through (*Watcher).verifyAndPublish so transfer-verifier state updates cannot be bypassed.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/evm-verify-and-publish-gate
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

predicate isMessagePublicationPointer(Type t) {
  t.(PointerType).getBaseType().hasQualifiedName(
    "github.com/certusone/wormhole/node/pkg/common", "MessagePublication"
  )
}

predicate isPublicationChannel(Type t) {
  t.(ChanType).canSend() and
  isMessagePublicationPointer(t.(ChanType).getElementType())
}

predicate isEvmWatcherMsgCField(Field field) {
  field.hasQualifiedName("github.com/certusone/wormhole/node/pkg/watchers/evm", "Watcher", "msgC")
}

predicate isDirectProtectedChannelExpr(Expr channel) {
  exists(SelectorExpr selector, Field field |
    selector = channel.stripParens() and
    selector.refersTo(field) and
    isEvmWatcherMsgCField(field) and
    isPublicationChannel(selector.getType())
  )
}

predicate sameVariable(Expr a, Expr b) {
  exists(Entity target |
    a.stripParens().(Ident).refersTo(target) and
    b.stripParens().(Ident).refersTo(target)
  )
}

predicate occursBefore(AstNode earlier, AstNode later) {
  earlier.getLocation().getStartLine() < later.getLocation().getStartLine()
  or
  earlier.getLocation().getStartLine() = later.getLocation().getStartLine() and
  earlier.getLocation().getStartColumn() < later.getLocation().getStartColumn()
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

predicate isProtectedChannelExprAt(Expr channel, AstNode use) {
  isDirectProtectedChannelExpr(channel)
  or
  exists(Assignment assign, Expr lhs, int index |
    lhs = assign.getLhs(index) and
    sameVariable(channel, lhs) and
    isPublicationChannel(channel.getType()) and
    assign.getEnclosingFunction() = use.getEnclosingFunction() and
    occursBefore(assign, use) and
    variableIsNotReassignedBetween(channel, assign, use) and
    isProtectedChannelExprAt(assign.getRhs(index), assign)
  )
}

predicate isApprovedVerifyAndPublishBody(SendStmt send) {
  exists(MethodDecl method |
    method = send.getEnclosingFunction() and
    method.getName() = "verifyAndPublish" and
    method.getReceiverBaseType().hasQualifiedName(
      "github.com/certusone/wormhole/node/pkg/watchers/evm", "Watcher"
    )
  )
}

predicate sendChannelIsParameter(SendStmt send, Parameter parameter) {
  send.getChannel().stripParens().(Ident).refersTo(parameter)
}

predicate helperCalledWithProtectedChannel(SendStmt send, Parameter parameter, CallExpr call) {
  sendChannelIsParameter(send, parameter) and
  parameter.getIndex() >= 0 and
  call.getTarget().getFuncDecl() = parameter.getFunction() and
  isProductionEvmWatcherFile(call.getFile()) and
  isProtectedChannelExprAt(call.getArgument(parameter.getIndex()), call)
}

predicate isDirectOrAliasBypass(SendStmt send) {
  isProductionEvmWatcherFile(send.getFile()) and
  isProtectedChannelExprAt(send.getChannel(), send)
}

predicate isThinHelperBypass(SendStmt send) {
  isProductionEvmWatcherFile(send.getFile()) and
  exists(Parameter parameter, CallExpr call |
    helperCalledWithProtectedChannel(send, parameter, call) and
    isPublicationChannel(send.getChannel().getType())
  )
}

string bypassKind(SendStmt send) {
  isThinHelperBypass(send) and
  result = "this thin helper send is reachable from a call that passes the watcher's msgC channel"
  or
  isDirectProtectedChannelExpr(send.getChannel()) and
  result = "this send writes to the watcher's msgC field directly"
  or
  isProtectedChannelExprAt(send.getChannel(), send) and
  not isDirectProtectedChannelExpr(send.getChannel()) and
  result = "this send writes through a local alias of the watcher's msgC channel"
}

from SendStmt send, string kind
where
  kind = bypassKind(send) and
  (isDirectOrAliasBypass(send) or isThinHelperBypass(send)) and
  not isApprovedVerifyAndPublishBody(send)
select send,
  "EVM watcher publications must go through (*Watcher).verifyAndPublish; " + kind +
    " and can bypass transfer-verifier state updates."
