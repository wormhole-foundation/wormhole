/**
 * @name XRPL generated publication uses non-domain-separated emitter
 * @description XRPL watcher-generated XTCF, XACK, and NTT publications must use their family-specific domain-separated emitter derivation.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/xrpl-derived-generated-emitter
 * @tags security
 *       external/cwe/cwe-20
 */

import go
import semmle.go.concepts.GeneratedFile

predicate isProductionXrplWatcherFile(File f) {
  f.getRelativePath().matches("node/pkg/watchers/xrpl/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate fieldInit(KeyValueExpr field, string name, Expr value) {
  field.getKey().(Ident).getName() = name and
  value = field.getValue()
}

predicate isMessagePublicationLiteral(CompositeLit lit) {
  isProductionXrplWatcherFile(lit.getFile()) and
  lit.getType().getName() = "MessagePublication"
}

predicate hasXrplEmitterChain(CompositeLit lit) {
  exists(int i, KeyValueExpr field, SelectorExpr chain |
    field = lit.getElement(i) and
    fieldInit(field, "EmitterChain", chain) and
    chain.getSelector().getName() = "ChainIDXRPL"
  )
}

predicate hasEmitterAddressField(CompositeLit lit, KeyValueExpr field, Expr emitter) {
  exists(int i |
    field = lit.getElement(i) and
    fieldInit(field, "EmitterAddress", emitter)
  )
}

predicate subtreeMentionsIdent(Expr e, string name) {
  exists(Ident id |
    id = e.getAChild*() and
    id.getName() = name
  )
}

predicate functionMentionsIdent(FuncDecl f, string name) {
  exists(Ident id |
    id.getEnclosingFunction() = f and
    id.getName() = name
  )
}

predicate functionCalls(FuncDecl f, string calleeName) {
  exists(CallExpr call |
    call.getEnclosingFunction() = f and
    call.getTarget().getFuncDecl().getName() = calleeName
  )
}

predicate constructsGeneratedFamily(FuncDecl f, string family) {
  isProductionXrplWatcherFile(f.getFile()) and
  (
    family = "XTCF" and
    functionMentionsIdent(f, "xtcfPrefix")
    or
    family = "XACK" and
    functionMentionsIdent(f, "xackPrefix")
    or
    family = "NTT" and
    f.getName() = "parseNttTransaction" and
    functionCalls(f, "buildNTTPayload")
  )
}

predicate isXrplGeneratedPublication(CompositeLit lit, string family) {
  isMessagePublicationLiteral(lit) and
  hasXrplEmitterChain(lit) and
  constructsGeneratedFamily(lit.getEnclosingFunction(), family)
}

predicate sameLocal(Expr a, Expr b) {
  exists(Entity target |
    a.stripParens().(Ident).refersTo(target) and
    b.stripParens().(Ident).refersTo(target)
  )
}

predicate choosesEmitterFor(Expr chosen, Expr emitter) {
  chosen = emitter.stripParens()
  or
  exists(Assignment assign |
    sameLocal(assign.getLhs(0), emitter) and
    chosen = assign.getRhs(0).stripParens() and
    assign.getEnclosingFunction() = emitter.getEnclosingFunction() and
    not exists(Assignment other |
      other != assign and
      other.getEnclosingFunction() = emitter.getEnclosingFunction() and
      sameLocal(other.getLhs(0), emitter)
    )
  )
}

predicate reportLocationFor(Expr emitter, AstNode report) {
  exists(Assignment assign |
    sameLocal(assign.getLhs(0), emitter) and
    assign.getEnclosingFunction() = emitter.getEnclosingFunction() and
    not exists(Assignment other |
      other != assign and
      other.getEnclosingFunction() = emitter.getEnclosingFunction() and
      sameLocal(other.getLhs(0), emitter)
    ) and
    report = assign
  )
  or
  not exists(Assignment assign |
    sameLocal(assign.getLhs(0), emitter) and
    assign.getEnclosingFunction() = emitter.getEnclosingFunction() and
    not exists(Assignment other |
      other != assign and
      other.getEnclosingFunction() = emitter.getEnclosingFunction() and
      sameLocal(other.getLhs(0), emitter)
    )
  ) and
  report = emitter
}

predicate callToHelper(Expr e, string helperName, FuncDecl helper) {
  exists(CallExpr call |
    call = e.stripParens() and
    call.getTarget().getFuncDecl() = helper and
    helper.getName() = helperName
  )
}

predicate isCopyCall(CallExpr call) { call.getTarget().getName() = "copy" }

predicate isKeccak256Call(CallExpr call) { call.getTarget().getName() = "Keccak256" }

predicate isSliceOfLocal(Expr e, Entity target, int low, int high) {
  exists(SliceExpr slice, Ident base |
    slice = e.stripParens() and
    base = slice.getBase().stripParens() and
    base.refersTo(target) and
    (
      low = 0 and not exists(slice.getLow())
      or
      slice.getLow().getIntValue() = low
    ) and
    slice.getHigh().getIntValue() = high
  )
}

predicate isSliceOfLocalFromStart(Expr e, Entity target, FuncDecl helper) {
  exists(SliceExpr slice, Ident base |
    slice.getEnclosingFunction() = helper and
    slice = e.stripParens() and
    base = slice.getBase().stripParens() and
    base.refersTo(target) and
    (
      not exists(slice.getLow())
      or
      slice.getLow().getIntValue() = 0
    )
  )
}

predicate isNttManagerSliceOfLocal(Expr e, Entity target, FuncDecl helper) {
  exists(SliceExpr slice, Ident base |
    slice.getEnclosingFunction() = helper and
    slice = e.stripParens() and
    base = slice.getBase().stripParens() and
    base.refersTo(target) and
    (
      slice.getLow().getIntValue() = 3
      or
      subtreeMentionsIdent(slice.getLow(), "nttEmitterDomainLen")
    ) and
    (
      slice.getHigh().getIntValue() = 35
      or
      subtreeMentionsIdent(slice.getHigh(), "nttEmitterDomainLen") and
      subtreeMentionsIdent(slice.getHigh(), "addrLen")
    )
  )
}

predicate isNttTokenSliceOfLocal(Expr e, Entity target, FuncDecl helper) {
  exists(SliceExpr slice, Ident base |
    slice.getEnclosingFunction() = helper and
    slice = e.stripParens() and
    base = slice.getBase().stripParens() and
    base.refersTo(target) and
    (
      slice.getLow().getIntValue() = 35
      or
      subtreeMentionsIdent(slice.getLow(), "nttEmitterDomainLen") and
      subtreeMentionsIdent(slice.getLow(), "addrLen")
    ) and
    (
      not exists(slice.getHigh())
      or
      slice.getHigh().getIntValue() = 67
    )
  )
}

predicate noIndexedWritesToLocal(FuncDecl helper, Entity target) {
  not exists(Assignment assign, Expr lhs, Ident written |
    assign.getEnclosingFunction() = helper and
    lhs = assign.getLhs(_) and
    written = lhs.getAChild*() and
    written.refersTo(target) and
    not lhs.stripParens() instanceof Ident
  )
}

predicate noOtherCopiesToLocal(FuncDecl helper, Entity target, CallExpr approvedCopy) {
  not exists(CallExpr otherCopy, SliceExpr destination, Ident base |
    otherCopy != approvedCopy and
    otherCopy.getEnclosingFunction() = helper and
    isCopyCall(otherCopy) and
    destination = otherCopy.getArgument(0).stripParens() and
    base = destination.getBase().stripParens() and
    base.refersTo(target)
  )
}

predicate generatedEmitterHelperHasRequiredOverlay(FuncDecl helper) {
  helper.getName() = "calculateGeneratedEmitterAddress" and
  exists(ReturnStmt ret, Ident returned, Entity target, CallExpr seedCall, CallExpr copyCall |
    ret.getEnclosingFunction() = helper and
    returned = ret.getExpr(0).stripParens() and
    returned.refersTo(target) and
    seedCall.getEnclosingFunction() = helper and
    callToHelper(seedCall, "addressToEmitter", _) and
    choosesEmitterFor(seedCall, returned) and
    copyCall.getEnclosingFunction() = helper and
    isCopyCall(copyCall) and
    isSliceOfLocal(copyCall.getArgument(0), target, 0, 4) and
    subtreeMentionsIdent(copyCall.getArgument(1), "generatedEmitterPrefix") and
    noIndexedWritesToLocal(helper, target) and
    noOtherCopiesToLocal(helper, target, copyCall)
  )
}

predicate hashCallFeedsReturnedEmitter(FuncDecl helper, CallExpr hashCall) {
  exists(ReturnStmt ret |
    ret.getEnclosingFunction() = helper and
    DataFlow::localFlow(
      DataFlow::exprNode(hashCall), DataFlow::exprNode(ret.getExpr(0).stripParens())
    )
  )
  or
  exists(ReturnStmt ret, Ident returned, Entity target, CallExpr resultCopy |
    ret.getEnclosingFunction() = helper and
    returned = ret.getExpr(0).stripParens() and
    returned.refersTo(target) and
    resultCopy.getEnclosingFunction() = helper and
    isCopyCall(resultCopy) and
    isSliceOfLocalFromStart(resultCopy.getArgument(0), target, helper) and
    DataFlow::localFlow(
      DataFlow::exprNode(hashCall), DataFlow::exprNode(resultCopy.getArgument(1))
    )
  )
}

predicate nttEmitterHelperHasRequiredDomain(FuncDecl helper) {
  helper.getName() = "calculateEmitterAddress" and
  exists(CallExpr hashCall, Ident buf, Entity bufTarget, CallExpr prefixCopy, CallExpr managerCopy, CallExpr tokenCopy |
    hashCall.getEnclosingFunction() = helper and
    isKeccak256Call(hashCall) and
    hashCallFeedsReturnedEmitter(helper, hashCall) and
    buf = hashCall.getArgument(0).stripParens() and
    buf.refersTo(bufTarget) and
    prefixCopy.getEnclosingFunction() = helper and
    isCopyCall(prefixCopy) and
    isSliceOfLocalFromStart(prefixCopy.getArgument(0), bufTarget, helper) and
    prefixCopy.getArgument(1).getStringValue() = "ntt" and
    managerCopy.getEnclosingFunction() = helper and
    isCopyCall(managerCopy) and
    isNttManagerSliceOfLocal(managerCopy.getArgument(0), bufTarget, helper) and
    subtreeMentionsIdent(managerCopy.getArgument(1), "sourceNTTManager") and
    tokenCopy.getEnclosingFunction() = helper and
    isCopyCall(tokenCopy) and
    isNttTokenSliceOfLocal(tokenCopy.getArgument(0), bufTarget, helper) and
    subtreeMentionsIdent(tokenCopy.getArgument(1), "sourceToken")
  )
}

predicate approvedEmitterDerivation(string family, Expr emitter) {
  exists(Expr chosen, FuncDecl helper |
    choosesEmitterFor(chosen, emitter) and
    (
      family in ["XTCF", "XACK"] and
      callToHelper(chosen, "calculateGeneratedEmitterAddress", helper) and
      generatedEmitterHelperHasRequiredOverlay(helper)
      or
      family = "NTT" and
      callToHelper(chosen, "calculateEmitterAddress", helper) and
      nttEmitterHelperHasRequiredDomain(helper)
    )
  )
}

string requiredDerivation(string family) {
  family in ["XTCF", "XACK"] and
  result = "the \"XRPL\" generated account emitter"
  or
  family = "NTT" and
  result = "keccak256(\"ntt\" + source manager + source token)"
}

string observedDerivation(Expr emitter) {
  exists(Expr chosen, FuncDecl helper |
    choosesEmitterFor(chosen, emitter) and
    callToHelper(chosen, "addressToEmitter", helper) and
    result = "a raw account emitter"
  )
  or
  not exists(Expr chosen, FuncDecl helper |
    choosesEmitterFor(chosen, emitter) and
    callToHelper(chosen, "addressToEmitter", helper)
  ) and
  result = "a non-approved emitter derivation"
}

from CompositeLit lit, KeyValueExpr field, Expr emitter, AstNode report, string family
where
  isXrplGeneratedPublication(lit, family) and
  hasEmitterAddressField(lit, field, emitter) and
  reportLocationFor(emitter, report) and
  not approvedEmitterDerivation(family, emitter)
select report,
  "XRPL generated " + family + " publication must use the domain-separated emitter (" +
    requiredDerivation(family) + "); this expression appears to use " + observedDerivation(emitter) + "."
