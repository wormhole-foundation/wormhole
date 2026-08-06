/**
 * @name Solana message published without commitment match proof
 * @description Solana watcher paths must check the decoded message commitment against the watcher's configured commitment before scheduling or publishing a MessagePublication.
 * @kind problem
 * @problem.severity error
 * @precision high
 * @id wormhole/go/solana-commitment-match-before-publication
 * @tags security
 *       external/cwe/cwe-345
 */

import go
import semmle.go.concepts.GeneratedFile

predicate isProductionSolanaWatcherFile(File f) {
  f.getRelativePath().matches("node/pkg/watchers/solana/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate before(AstNode earlier, AstNode later) {
  earlier.getLocation().getStartLine() < later.getLocation().getStartLine()
  or
  earlier.getLocation().getStartLine() = later.getLocation().getStartLine() and
  earlier.getLocation().getStartColumn() < later.getLocation().getStartColumn()
}

predicate syntacticallyNestedIn(AstNode inner, AstNode outer) {
  inner.getLocation().getStartLine() >= outer.getLocation().getStartLine() and
  inner.getLocation().getEndLine() <= outer.getLocation().getEndLine()
}

predicate guardedOnlyOnBypassedBranch(IfStmt ifStmt, AstNode sink) {
  exists(IfStmt outer |
    outer.getEnclosingFunction() = sink.getEnclosingFunction() and
    outer != ifStmt and
    syntacticallyNestedIn(ifStmt, outer) and
    outer.getLocation().getEndLine() < sink.getLocation().getStartLine()
  )
}

predicate branchReturnsDirectly(Stmt branch) {
  exists(ReturnStmt ret | ret = branch.getAChild())
}

predicate sameLocalVariable(Expr a, Expr b) {
  exists(Ident ai, Ident bi, Entity v |
    a.stripParens() = ai and
    b.stripParens() = bi and
    ai.refersTo(v) and
    bi.refersTo(v)
  )
}

predicate sameValue(Expr a, Expr b) {
  a.stripParens() = b.stripParens()
  or
  sameLocalVariable(a, b)
}

predicate assignmentToSameLocal(Assignment assign, Expr local) {
  exists(Ident lhs, Ident use, Entity v |
    assign.getLhs(_) = lhs and
    local.stripParens() = use and
    lhs.refersTo(v) and
    use.refersTo(v)
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

predicate isConsistencyLevelRead(Expr e) {
  exists(SelectorExpr sel, Field field |
    e.stripParens() = sel and
    sel.refersTo(field) and
    field.getName() = "ConsistencyLevel"
  )
}

predicate isCommitmentConversionCall(CallExpr call) {
  isProductionSolanaWatcherFile(call.getFile()) and
  (
    call.getCalleeName() = "accountConsistencyLevelToCommitment" and
    call.getNumArgument() = 1 and
    isConsistencyLevelRead(call.getArgument(0))
    or
    call.getCalleeName() = "Commitment" and
    exists(SelectorExpr callee, SelectorExpr levelRead |
      call.getCalleeExpr().stripParens() = callee and
      callee.getBase().stripParens() = levelRead and
      isConsistencyLevelRead(levelRead)
    )
  )
}

predicate assignmentReceivesTupleElement(Assignment assign, CallExpr call, int index, Expr lhs) {
  assign.getRhs(0) = call and
  lhs = assign.getLhs(index)
}

predicate convertedCommitment(CallExpr conversion, Expr commitment) {
  isCommitmentConversionCall(conversion) and
  exists(Assignment assign | assignmentReceivesTupleElement(assign, conversion, 0, commitment))
}

predicate convertedInstructionCommitment(CallExpr conversion, Expr commitment) {
  convertedCommitment(conversion, commitment) and
  exists(SelectorExpr callee, SelectorExpr levelRead |
    conversion.getCalleeExpr().stripParens() = callee and
    callee.getBase().stripParens() = levelRead and
    isConsistencyLevelRead(levelRead) and
    levelRead.getBase().getType().getName() = "PostMessageData"
  )
}

predicate pairedConversionErrorExpr(CallExpr conversion, Expr err) {
  exists(Assignment assign |
    assign.getEnclosingFunction() = conversion.getEnclosingFunction() and
    assignmentReceivesTupleElement(assign, conversion, 1, err)
  )
}

predicate isCheckCommitmentCall(CallExpr call, Expr commitment) {
  isProductionSolanaWatcherFile(call.getFile()) and
  call.getEnclosingFunction() = commitment.getEnclosingFunction() and
  call.getCalleeName() = "checkCommitment" and
  call.getTarget() instanceof Method and
  call.getTarget().(Method).getReceiverBaseType().getName() = "SolanaWatcher" and
  call.getNumArgument() = 2 and
  sameValue(call.getArgument(0), commitment)
}

predicate checkCommitmentReceiver(CallExpr call, Expr receiver) {
  exists(SelectorExpr callee |
    callee = call.getCalleeExpr().(SelectorExpr) and
    receiver = callee.getBase()
  )
}

predicate msgCSendReceiver(SendStmt send, Expr receiver) {
  exists(SelectorExpr channel, Field field |
    send.getChannel().stripParens() = channel and
    channel.refersTo(field) and
    field.getName() = "msgC" and
    receiver = channel.getBase()
  )
}

predicate retryFetchReceiver(CallExpr call, Expr receiver) {
  call.getCalleeName() = "retryFetchMessageAccount" and
  call.getTarget() instanceof Method and
  call.getTarget().(Method).getReceiverBaseType().getName() = "SolanaWatcher" and
  exists(SelectorExpr callee |
    callee = call.getCalleeExpr().(SelectorExpr) and
    receiver = callee.getBase()
  )
}

predicate sinkWatcherReceiver(AstNode sink, Expr receiver) {
  exists(SendStmt send |
    sink = send and
    isSolanaWatcherMsgCSend(send) and
    msgCSendReceiver(send, receiver)
  )
  or
  exists(CallExpr call |
    sink = call and
    isInstructionAccountFetchSink(call) and
    retryFetchReceiver(call, receiver)
  )
  or
  exists(CallExpr call, FuncDef body, CallExpr retry |
    sink = call and
    runnableBody(call, body) and
    retry.getEnclosingFunction() = body and
    isInstructionAccountFetchSink(retry) and
    retryFetchReceiver(retry, receiver)
  )
}

predicate sameWatcherReceiver(CallExpr check, AstNode sink) {
  exists(Expr checkReceiver, Expr sinkReceiver |
    checkCommitmentReceiver(check, checkReceiver) and
    sinkWatcherReceiver(sink, sinkReceiver) and
    sameValue(checkReceiver, sinkReceiver)
  )
}

predicate condIsCheckTrue(Expr cond, CallExpr check) { cond.stripParens() = check }

predicate condIsCheckFalse(Expr cond, CallExpr check) {
  exists(NotExpr neg |
    cond.stripParens() = neg and
    neg.getOperand().stripParens() = check
  )
}

predicate guardProvesCallTrueBefore(CallExpr check, AstNode sink) {
  exists(IfStmt ifStmt |
    ifStmt.getEnclosingFunction() = sink.getEnclosingFunction() and
    before(ifStmt, sink) and
    condIsCheckFalse(ifStmt.getCond(), check) and
    branchReturnsDirectly(ifStmt.getThen()) and
    not guardedOnlyOnBypassedBranch(ifStmt, sink)
  )
  or
  exists(IfStmt ifStmt |
    ifStmt.getEnclosingFunction() = sink.getEnclosingFunction() and
    before(ifStmt, sink) and
    condIsCheckTrue(ifStmt.getCond(), check) and
    syntacticallyNestedIn(sink, ifStmt.getThen())
  )
  or
  exists(IfStmt ifStmt |
    ifStmt.getEnclosingFunction() = sink.getEnclosingFunction() and
    before(ifStmt, sink) and
    condIsCheckFalse(ifStmt.getCond(), check) and
    syntacticallyNestedIn(sink, ifStmt.getElse())
  )
}

predicate guardProvesErrorNilBefore(Expr err, AstNode proof, AstNode sink) {
  exists(IfStmt ifStmt, NeqExpr neq, Expr errRead, Expr nil |
    exprRefersToNil(nil) and
    errRead.getEnclosingFunction() = sink.getEnclosingFunction() and
    ifStmt.getEnclosingFunction() = sink.getEnclosingFunction() and
    before(proof, ifStmt) and
    before(ifStmt, sink) and
    syntacticallyNestedIn(neq, ifStmt.getCond()) and
    (
      neq.getLeftOperand() = errRead and neq.getRightOperand() = nil
      or
      neq.getRightOperand() = errRead and neq.getLeftOperand() = nil
    ) and
    sameValue(errRead, err) and
    branchReturnsDirectly(ifStmt.getThen()) and
    not guardedOnlyOnBypassedBranch(ifStmt, sink)
  )
}

predicate hasCommitmentProofBefore(Expr commitment, AstNode sink) {
  exists(CallExpr check |
    isCheckCommitmentCall(check, commitment) and
    before(check, sink) and
    sameWatcherReceiver(check, sink) and
    guardProvesCallTrueBefore(check, sink) and
    not localReassignedBetween(commitment, check, sink)
  )
}

predicate hasConversionErrorProofBefore(CallExpr conversion, AstNode sink) {
  exists(Expr err |
    pairedConversionErrorExpr(conversion, err) and
    guardProvesErrorNilBefore(err, conversion, sink) and
    not localReassignedBetween(err, conversion, sink)
  )
}

predicate decodedCommitmentLacksProofBefore(CallExpr conversion, Expr commitment, AstNode sink) {
  convertedCommitment(conversion, commitment) and
  conversion.getEnclosingFunction() = sink.getEnclosingFunction() and
  before(conversion, sink) and
  (
    not hasConversionErrorProofBefore(conversion, sink)
    or
    not hasCommitmentProofBefore(commitment, sink)
  )
}

predicate isSolanaWatcherMsgCSend(SendStmt send) {
  isProductionSolanaWatcherFile(send.getFile()) and
  exists(SelectorExpr channel, Field field |
    send.getChannel().stripParens() = channel and
    channel.refersTo(field) and
    field.getName() = "msgC"
  )
}

predicate isInstructionAccountFetchSink(CallExpr call) {
  isProductionSolanaWatcherFile(call.getFile()) and
  call.getCalleeName() = "retryFetchMessageAccount"
}

predicate runnableBody(CallExpr call, FuncDef body) {
  call.getCalleeName() = "RunWithScissors" and
  call.getNumArgument() = 4 and
  body = call.getArgument(3).(FuncLit)
}

predicate runnableCallsRetryFetchMessageAccount(CallExpr call) {
  exists(FuncDef body, CallExpr retry |
    runnableBody(call, body) and
    retry.getEnclosingFunction() = body and
    isInstructionAccountFetchSink(retry)
  )
}

predicate isInstructionAccountFetchSchedule(CallExpr call) {
  isInstructionAccountFetchSink(call)
  or
  (
    isProductionSolanaWatcherFile(call.getFile()) and
    call.getCalleeName() = "RunWithScissors" and
    call.getNumArgument() = 4 and
    call.getArgument(2).getStringValue() = "retryFetchMessageAccount" and
    runnableCallsRetryFetchMessageAccount(call)
  )
}

predicate sinkLacksMatchingProof(AstNode sink, CallExpr conversion, Expr commitment) {
  exists(SendStmt send |
    sink = send and
    isSolanaWatcherMsgCSend(send) and
    conversion.getFile() = send.getFile() and
    not convertedInstructionCommitment(conversion, commitment) and
    decodedCommitmentLacksProofBefore(conversion, commitment, sink)
  )
  or
  exists(CallExpr call |
    sink = call and
    isInstructionAccountFetchSchedule(call) and
    conversion.getFile() = call.getFile() and
    convertedInstructionCommitment(conversion, commitment) and
    conversion.getEnclosingFunction() = sink.getEnclosingFunction() and
    before(conversion, sink) and
    (
      not hasConversionErrorProofBefore(conversion, sink)
      or
      not hasCommitmentProofBefore(commitment, sink)
    )
  )
}

class PublicationSink extends AstNode {
  PublicationSink() {
    exists(SendStmt send | this = send and isSolanaWatcherMsgCSend(send))
    or
    exists(CallExpr call | this = call and isInstructionAccountFetchSchedule(call))
  }
}

from PublicationSink sink
where
  exists(CallExpr conversion, Expr commitment |
    sinkLacksMatchingProof(sink, conversion, commitment)
  )
select sink,
  "Solana watcher must prove the decoded message commitment matches the watcher commitment before scheduling or publishing the observation."
