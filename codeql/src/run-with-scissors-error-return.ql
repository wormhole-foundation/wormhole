/**
 * @name RunWithScissors runnable sends directly to wrapper error channel
 * @description Runnables passed to common.RunWithScissors should return fatal errors and let the wrapper forward them without blocking, not send directly to the same error channel.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/run-with-scissors-error-return
 * @tags security
 *       external/cwe/cwe-667
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.dataflow.GlobalValueNumbering

predicate isProductionNodeGoFile(File f) {
  f.getRelativePath().matches("node/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate isRunWithScissorsCall(CallExpr call) {
  isProductionNodeGoFile(call.getFile()) and
  call.getTarget().getName() = "RunWithScissors" and
  call.getTarget().getPackage().getName() = "common" and
  call.getTarget().getPackage().getPath() in [
    "github.com/certusone/wormhole/node/pkg/common",
    "github.com/wormhole-foundation/wormhole/node/pkg/common"
  ]
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

predicate sameReceiverField(Expr wrapperChannel, Expr candidateChannel) {
  exists(SelectorExpr wrapperSel, SelectorExpr candidateSel, Field field |
    wrapperChannel.stripParens() = wrapperSel and
    candidateChannel.stripParens() = candidateSel and
    wrapperSel.refersTo(field) and
    candidateSel.refersTo(field) and
    sameValue(wrapperSel.getBase(), candidateSel.getBase())
  )
}

predicate sameChannel(Expr wrapperChannel, Expr candidateChannel) {
  sameValue(wrapperChannel, candidateChannel)
  or
  sameReceiverField(wrapperChannel, candidateChannel)
}

predicate exprStrictlyBefore(Expr earlier, Expr later) {
  earlier.getLocation().getStartLine() < later.getLocation().getStartLine()
  or
  earlier.getLocation().getStartLine() = later.getLocation().getStartLine() and
  earlier.getLocation().getStartColumn() < later.getLocation().getStartColumn()
}

predicate assignmentStrictlyBefore(Assignment earlier, Expr later) {
  earlier.getLocation().getStartLine() < later.getLocation().getStartLine()
  or
  earlier.getLocation().getStartLine() = later.getLocation().getStartLine() and
  earlier.getLocation().getStartColumn() < later.getLocation().getStartColumn()
}

predicate assignmentToLocal(Assignment assign, LocalVariable v) {
  exists(Ident lhs | assign.getLhs(_) = lhs and lhs.refersTo(v))
}

predicate localFunctionValueRunnable(CallExpr call, FuncLit lit) {
  exists(Ident runnable, LocalVariable v, Assignment assign |
    call.getArgument(3) = runnable and
    runnable.refersTo(v) and
    assignmentToLocal(assign, v) and
    assign.getRhs() = lit and
    assign.getEnclosingFunction() = call.getEnclosingFunction() and
    assignmentStrictlyBefore(assign, call) and
    not exists(Assignment overwrite |
      assignmentToLocal(overwrite, v) and
      overwrite.getEnclosingFunction() = call.getEnclosingFunction() and
      exprStrictlyBefore(assign.getRhs(), overwrite.getRhs()) and
      assignmentStrictlyBefore(overwrite, call)
    )
  )
}

predicate methodValueRunnable(CallExpr call, SelectorExpr runnable, Method method) {
  call.getArgument(3) = runnable and
  runnable.getSelector().refersTo(method)
}

predicate runnableBody(CallExpr call, FuncDef body) {
  isRunWithScissorsCall(call) and
  (
    body = call.getArgument(3).(FuncLit)
    or
    exists(FuncLit lit | localFunctionValueRunnable(call, lit) and body = lit)
    or
    exists(SelectorExpr runnable, Method method |
      methodValueRunnable(call, runnable, method) and
      body = method.getFuncDecl()
    )
  )
}

predicate methodReceiverFieldSend(CallExpr call, SendStmt send) {
  exists(
    SelectorExpr runnable, Method method, SelectorExpr wrapperSel, SelectorExpr sendSel, Field field,
    ReceiverVariable receiver
  |
    isRunWithScissorsCall(call) and
    methodValueRunnable(call, runnable, method) and
    call.getArgument(1).stripParens() = wrapperSel and
    send.getChannel().stripParens() = sendSel and
    wrapperSel.refersTo(field) and
    sendSel.refersTo(field) and
    sameValue(wrapperSel.getBase(), runnable.getBase()) and
    receiver.isReceiverOf(method.getFuncDecl()) and
    sendSel.getBase().(Ident).refersTo(receiver) and
    send.getEnclosingFunction() = method.getFuncDecl()
  )
}

predicate directRunnableSend(CallExpr call, SendStmt send) {
  exists(FuncDef body |
    runnableBody(call, body) and
    send.getEnclosingFunction() = body and
    sameChannel(call.getArgument(1), send.getChannel())
  )
  or
  methodReceiverFieldSend(call, send)
}

predicate helperParameterSend(CallExpr runCall, CallExpr helperCall, SendStmt send) {
  exists(FuncDef body, FuncDecl helper, Parameter parameter, Ident parameterRead, int i |
    runnableBody(runCall, body) and
    helperCall.getEnclosingFunction() = body and
    helperCall.getTarget().getFuncDecl() = helper and
    helper.getParameter(i) = parameter and
    sameChannel(runCall.getArgument(1), helperCall.getArgument(i)) and
    parameterRead.refersTo(parameter) and
    parameterRead.getEnclosingFunction() = helper and
    send.getEnclosingFunction() = helper and
    sameValue(parameterRead, send.getChannel())
  )
}

predicate helperReceiverSend(CallExpr runCall, CallExpr helperCall, SendStmt send) {
  exists(
    FuncDef body, SelectorExpr helperSelector, Method helper, SelectorExpr wrapperSel,
    SelectorExpr sendSel, Field field, ReceiverVariable receiver
  |
    runnableBody(runCall, body) and
    helperCall.getEnclosingFunction() = body and
    helperCall.getCalleeExpr() = helperSelector and
    helperSelector.getSelector().refersTo(helper) and
    runCall.getArgument(1).stripParens() = wrapperSel and
    send.getChannel().stripParens() = sendSel and
    wrapperSel.refersTo(field) and
    sendSel.refersTo(field) and
    sameValue(wrapperSel.getBase(), helperSelector.getBase()) and
    receiver.isReceiverOf(helper.getFuncDecl()) and
    sendSel.getBase().(Ident).refersTo(receiver) and
    send.getEnclosingFunction() = helper.getFuncDecl()
  )
}

predicate isGoLaunchedCall(CallExpr call) {
  exists(GoStmt go | call = go.getCall() or call.getParent*() = go)
}

predicate oneHopHelperSend(CallExpr runCall, SendStmt send) {
  exists(CallExpr helperCall |
    not isGoLaunchedCall(helperCall) and
    helperParameterSend(runCall, helperCall, send)
    or
    not isGoLaunchedCall(helperCall) and
    helperReceiverSend(runCall, helperCall, send)
  )
}

from CallExpr runCall, SendStmt send
where isProductionNodeGoFile(send.getFile()) and (directRunnableSend(runCall, send) or oneHopHelperSend(runCall, send))
select send,
  "Return this runnable error instead of sending directly to the RunWithScissors error channel; RunWithScissors forwards returned errors without blocking."
