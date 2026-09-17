/**
 * @name AlreadyLocked helper called without holding the required receiver mutex
 * @description Documented AlreadyLocked helper methods must be called only while the exact receiver's mutex write lock is held.
 * @kind problem
 * @problem.severity warning
 * @precision medium
 * @id wormhole/go/already-locked-receiver-mutex
 * @tags security
 */

import go
import semmle.go.concepts.GeneratedFile
import semmle.go.controlflow.ControlFlowGraph
import semmle.go.dataflow.GlobalValueNumbering

predicate isRelevantProductionFile(File f) {
  (
    f.getRelativePath().matches("node/pkg/accountant/%.go") or
    f.getRelativePath().matches("node/pkg/governor/%.go") or
    f.getRelativePath().matches("node/cmd/ccq/%.go")
  ) and
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

predicate isDeferCall(CallExpr call) { exists(DeferStmt defer | defer.getCall() = call) }

predicate isAsynchronousCall(CallExpr call) {
  isDeferCall(call) or exists(GoStmt go | call = go.getCall() or call.getParent*() = go)
}

predicate methodOnReceiver(CallExpr call, Expr receiver, string packageName, string typeName, string methodName) {
  isRelevantProductionFile(call.getFile()) and
  call.getCalleeName() = methodName and
  call.getTarget().getPackage().getName() = packageName and
  call.getTarget() instanceof Method and
  call.getTarget().(Method).getReceiverBaseType().getName() = typeName and
  exists(SelectorExpr callee |
    typeName in ["Accountant", "ChainGovernor", "PendingResponses"] and
    callee = call.getCalleeExpr().(SelectorExpr) and
    receiver = callee.getBase()
  )
}

predicate isAlreadyLockedMethod(Method method, string mutexField) {
  method.getPackage().getName() = "accountant" and
  method.getReceiverBaseType().getName() = "Accountant" and
  method.getName() in ["publishTransferAlreadyLocked", "addPendingTransferAlreadyLocked", "deletePendingTransferAlreadyLocked"] and
  mutexField = "pendingTransfersLock"
  or
  method.getPackage().getName() = "governor" and
  method.getReceiverBaseType().getName() = "ChainGovernor" and
  method.getName() in ["parseMsgAlreadyLocked", "loadFromDBAlreadyLocked"] and
  mutexField = "mutex"
  or
  method.getPackage().getName() = "ccq" and
  method.getReceiverBaseType().getName() = "PendingResponses" and
  method.getName() = "updateMetricsAlreadyLocked" and
  mutexField = "mu"
}

predicate isAlreadyLockedCall(CallExpr call, Expr receiver, string mutexField) {
  methodOnReceiver(call, receiver, "accountant", "Accountant", "publishTransferAlreadyLocked") and
  mutexField = "pendingTransfersLock"
  or
  methodOnReceiver(call, receiver, "accountant", "Accountant", "addPendingTransferAlreadyLocked") and
  mutexField = "pendingTransfersLock"
  or
  methodOnReceiver(call, receiver, "accountant", "Accountant", "deletePendingTransferAlreadyLocked") and
  mutexField = "pendingTransfersLock"
  or
  methodOnReceiver(call, receiver, "governor", "ChainGovernor", "parseMsgAlreadyLocked") and
  mutexField = "mutex"
  or
  methodOnReceiver(call, receiver, "governor", "ChainGovernor", "loadFromDBAlreadyLocked") and
  mutexField = "mutex"
  or
  methodOnReceiver(call, receiver, "ccq", "PendingResponses", "updateMetricsAlreadyLocked") and
  mutexField = "mu"
}

predicate mutexMethodCallReceiver(
  CallExpr call, Expr actualReceiver, string mutexField, string methodName
) {
  call.getCalleeName() = methodName and
  exists(SelectorExpr callee, SelectorExpr mutexSelector, Field field |
    callee = call.getCalleeExpr().(SelectorExpr) and
    mutexSelector = callee.getBase().(SelectorExpr) and
    mutexSelector.refersTo(field) and
    field.getName() = mutexField and
    actualReceiver = mutexSelector.getBase()
  )
}

predicate isMutexMethodCall(CallExpr call, Expr receiver, string mutexField, string methodName) {
  exists(Expr actualReceiver |
    mutexMethodCallReceiver(call, actualReceiver, mutexField, methodName) and
    sameValue(actualReceiver, receiver)
  )
}

predicate lockDominatesCall(CallExpr lockCall, CallExpr lockedCall) {
  exists(ControlFlow::Node lockNode, ControlFlow::Node lockedNode |
    lockNode.isFirstNodeOf(lockCall) and
    lockedNode.isFirstNodeOf(lockedCall) and
    lockNode.getBasicBlock().dominates(lockedNode.getBasicBlock())
  )
}

predicate unlockDominatesCallBetween(CallExpr lockCall, CallExpr lockedCall, Expr receiver, string mutexField) {
  exists(CallExpr unlockCall |
    unlockCall.getEnclosingFunction() = lockedCall.getEnclosingFunction() and
    isMutexMethodCall(unlockCall, receiver, mutexField, "Unlock") and
    not isDeferCall(unlockCall) and
    before(lockCall, unlockCall) and
    before(unlockCall, lockedCall) and
    lockDominatesCall(unlockCall, lockedCall)
  )
}

predicate unlockCanReachCall(CallExpr unlockCall, CallExpr lockedCall) {
  exists(ControlFlow::Node unlockNode, ControlFlow::Node lockedNode |
    unlockNode.isFirstNodeOf(unlockCall) and
    lockedNode.isFirstNodeOf(lockedCall) and
    lockedNode = unlockNode.getASuccessor*()
  )
}

predicate directlyContainedInBlock(AstNode node, BlockStmt block) {
  block = node.getParent*() and
  not exists(BlockStmt inner |
    inner != block and
    inner = node.getParent*() and
    block = inner.getParent*()
  )
}

predicate branchExitCanBypassRelock(CallExpr unlockCall, CallExpr relockCall, BlockStmt branch) {
  exists(Stmt exit |
    exit instanceof BreakStmt or exit instanceof ContinueStmt or exit instanceof GotoStmt
  |
    branch = exit.getParent*() and
    before(unlockCall, exit) and
    before(exit, relockCall)
  )
}

predicate hasSameBranchProtectingRelock(
  CallExpr unlockCall, CallExpr lockedCall, Expr receiver, string mutexField
) {
  exists(CallExpr relockCall, BlockStmt branch |
    relockCall.getEnclosingFunction() = lockedCall.getEnclosingFunction() and
    isMutexMethodCall(relockCall, receiver, mutexField, "Lock") and
    not isAsynchronousCall(relockCall) and
    before(unlockCall, relockCall) and
    before(relockCall, lockedCall) and
    directlyContainedInBlock(unlockCall, branch) and
    directlyContainedInBlock(relockCall, branch) and
    not branchExitCanBypassRelock(unlockCall, relockCall, branch)
  )
}

predicate unlockMayExposeCallBetween(CallExpr lockCall, CallExpr lockedCall, Expr receiver, string mutexField) {
  exists(CallExpr unlockCall |
    unlockCall.getEnclosingFunction() = lockedCall.getEnclosingFunction() and
    isMutexMethodCall(unlockCall, receiver, mutexField, "Unlock") and
    not isDeferCall(unlockCall) and
    before(lockCall, unlockCall) and
    unlockCanReachCall(unlockCall, lockedCall) and
    not hasSameBranchProtectingRelock(unlockCall, lockedCall, receiver, mutexField)
  )
}

predicate receiverReassignedBetween(
  Expr receiver, Expr lockedReceiver, AstNode earlier, CallExpr later
) {
  exists(Ident receiverRead, Variable receiverVariable, Assignment assign, Ident lhs, Expr rhs |
    receiver.stripParens() = receiverRead and
    receiverRead.refersTo(receiverVariable) and
    assign.getEnclosingFunction() = later.getEnclosingFunction() and
    assign.getAnLhs().stripParens() = lhs and
    lhs.refersTo(receiverVariable) and
    rhs = assign.getAnRhs() and
    not sameValue(rhs, lockedReceiver) and
    before(earlier, assign) and
    before(assign, later)
  )
}

predicate hasRequiredDominatingLock(CallExpr lockedCall, Expr receiver, string mutexField) {
  exists(CallExpr lockCall, Expr lockedReceiver |
    lockCall.getEnclosingFunction() = lockedCall.getEnclosingFunction() and
    mutexMethodCallReceiver(lockCall, lockedReceiver, mutexField, "Lock") and
    sameValue(lockedReceiver, receiver) and
    not isAsynchronousCall(lockCall) and
    before(lockCall, lockedCall) and
    lockDominatesCall(lockCall, lockedCall) and
    not receiverReassignedBetween(receiver, lockedReceiver, lockCall, lockedCall) and
    not unlockDominatesCallBetween(lockCall, lockedCall, receiver, mutexField) and
    not unlockMayExposeCallBetween(lockCall, lockedCall, receiver, mutexField)
  )
}

predicate inheritsRequiredLockPrecondition(CallExpr lockedCall, Expr receiver, string mutexField) {
  exists(Method enclosing, ReceiverVariable enclosingReceiver |
    lockedCall.getEnclosingFunction() = enclosing.getFuncDecl() and
    isAlreadyLockedMethod(enclosing, mutexField) and
    enclosingReceiver.isReceiverOf(enclosing.getFuncDecl()) and
    hasUnmodifiedReceiverProvenance(receiver, enclosingReceiver, lockedCall)
  )
}

predicate assignmentToVariableBeforeCall(Assignment assignment, Variable variable, CallExpr call) {
  exists(Ident lhs |
    assignment.getEnclosingFunction() = call.getEnclosingFunction() and
    assignment.getAnLhs().stripParens() = lhs and
    lhs.refersTo(variable) and
    before(assignment, call)
  )
}

predicate assignmentToVariableBetween(Assignment assignment, Variable variable, AstNode earlier, CallExpr later) {
  exists(Ident lhs |
    assignment.getEnclosingFunction() = later.getEnclosingFunction() and
    assignment.getAnLhs().stripParens() = lhs and
    lhs.refersTo(variable) and
    before(earlier, assignment) and
    before(assignment, later)
  )
}

predicate hasUnmodifiedReceiverProvenance(Expr receiver, ReceiverVariable enclosingReceiver, CallExpr lockedCall) {
  // Direct calls on the enclosing method receiver inherit the helper precondition only while the
  // receiver parameter has not been rebound. A rebinding may point at a different mutex.
  exists(Ident receiverRead |
    receiver.stripParens() = receiverRead and
    receiverRead.refersTo(enclosingReceiver) and
    not assignmentToVariableBeforeCall(_, enclosingReceiver, lockedCall)
  )
  or
  // One-hop aliases inherit the precondition only when they were assigned from the still-unmodified
  // receiver and were not overwritten before the nested AlreadyLocked call.
  exists(Ident aliasRead, Ident aliasLhs, Variable aliasVariable, Assignment aliasAssignment, Ident receiverRead |
    receiver.stripParens() = aliasRead and
    aliasRead.refersTo(aliasVariable) and
    not aliasVariable = enclosingReceiver and
    aliasAssignment.getEnclosingFunction() = lockedCall.getEnclosingFunction() and
    aliasAssignment.assigns(aliasLhs, receiverRead) and
    aliasLhs.refersTo(aliasVariable) and
    receiverRead.refersTo(enclosingReceiver) and
    before(aliasAssignment, lockedCall) and
    not assignmentToVariableBeforeCall(_, enclosingReceiver, lockedCall) and
    not assignmentToVariableBetween(_, aliasVariable, aliasAssignment, lockedCall)
  )
}

from CallExpr call, Expr receiver, string mutexField
where
  isAlreadyLockedCall(call, receiver, mutexField) and
  (
    isAsynchronousCall(call)
    or
    not hasRequiredDominatingLock(call, receiver, mutexField) and
    not inheritsRequiredLockPrecondition(call, receiver, mutexField)
  )
select call,
  "AlreadyLocked helper must be called only while holding the required write lock on the same receiver."
