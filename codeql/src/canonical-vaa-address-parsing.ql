/**
 * @name Non-canonical Wormhole address parsing
 * @description External Wormhole address data must be normalized with vaa.StringToAddress or vaa.BytesToAddress before use as a Wormhole identity.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/canonical-vaa-address-parsing
 * @tags security
 *       external/cwe/cwe-20
 */

import go
import semmle.go.concepts.GeneratedFile

predicate isProductionNodeFile(File f) {
  (
    f.getRelativePath().matches("node/%.go") or
    f.getRelativePath().matches("pkg/%.go")
  ) and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate isVaaAddressType(Type t) { t.getName() = "Address" and t.getPackage().getName() = "vaa" }

predicate isVaaAddressExpr(Expr e) { isVaaAddressType(e.getType()) }

predicate isByteLikeExpr(Expr e) {
  e.getType() instanceof SliceType or
  e.getType() instanceof ArrayType or
  e.getType().getName() = "string"
}

predicate isCanonicalAddressCall(CallExpr call) {
  call.getTarget().getName() in ["StringToAddress", "BytesToAddress"] and
  call.getTarget().getPackage().getName() = "vaa"
}

predicate isCanonicalBytesAddressCall(CallExpr call) {
  call.getTarget().getName() = "BytesToAddress" and call.getTarget().getPackage().getName() = "vaa"
}

predicate isCanonicalStringAddressCall(CallExpr call) {
  call.getTarget().getName() = "StringToAddress" and call.getTarget().getPackage().getName() = "vaa"
}

predicate isCanonicalAddressResult(Expr e) {
  exists(CallExpr call |
    isCanonicalAddressCall(call) and
    DataFlow::localFlow(DataFlow::extractTupleElement(DataFlow::exprNode(call), 0), DataFlow::exprNode(e))
  )
}

predicate isTypedInternalAddressRead(Expr e) {
  exists(SelectorExpr sel, Field field |
    e.stripParens() = sel and
    sel.refersTo(field) and
    field.getName() = "EmitterAddress" and
    sel.getBase().getType().getName() in ["MessagePublication", "VAA", "VAAID"]
  )
}

predicate isEvmPadAddressCopy(CallExpr call) {
  exists(FuncDecl f |
    call.getEnclosingFunction() = f and
    f.getName() = "PadAddress" and
    f.getFile().getRelativePath().matches("%/watchers/evm/utils.go") and
    call.getTarget().getName() = "copy" and
    exists(Parameter p | p = f.getParameter(0) and p.getType().getName() = "Address")
  )
}

predicate isAptosUint64SenderCopy(CallExpr call) {
  exists(FuncDecl f |
    call.getEnclosingFunction() = f and
    f.getFile().getRelativePath().matches("%/watchers/aptos/watcher.go") and
    f.getName() = "observeData" and
    call.getTarget().getName() = "copy"
  )
}

predicate isNearExact32EmitterDigestCopy(CallExpr call) {
  exists(FuncDecl f |
    call.getEnclosingFunction() = f and
    f.getFile().getRelativePath().matches("%/watchers/near/tx_processing.go") and
    f.getName() = "processWormholeLog" and
    call.getTarget().getName() = "copy" and
    hasExact32LengthGuard(call.getArgument(1), call)
  )
}

predicate isSuiTypedArrayConversion(ConversionExpr conv) {
  exists(FuncDecl f |
    conv.getEnclosingFunction() = f and
    f.getFile().getRelativePath().matches("%/watchers/sui/watcher.go") and
    conv.getOperand().getType() instanceof ArrayType
  )
}

predicate isXrplAccountCopy(CallExpr call) {
  exists(FuncDecl f |
    call.getEnclosingFunction() = f and
    f.getFile().getRelativePath().matches("%/watchers/xrpl/%.go") and
    f.getName() in ["CoreEmitterAccount", "addressToEmitter", "calculateEmitterAddress", "calculateGeneratedEmitterAddress"] and
    call.getTarget().getName() = "copy"
  )
}

predicate isCosmwasmCoreStringToAddressCopy(CallExpr call) {
  exists(FuncDecl f |
    call.getEnclosingFunction() = f and
    f.getFile().getRelativePath().matches("%/watchers/cosmwasm/watcher.go") and
    f.getName() = "StringToAddress" and
    call.getTarget().getName() = "copy" and
    cosmwasmDecodedEmitterSource(call.getArgument(1), f) and
    hasExact32LengthGuard(call.getArgument(1), call)
  )
}

predicate isNotaryKnownEmitterConversion(ConversionExpr conv) {
  exists(FuncDecl f, SelectorExpr sel |
    conv.getEnclosingFunction() = f and
    f.getFile().getRelativePath().matches("%/notary/admincommands.go") and
    f.getName() = "createTestMessagePublication" and
    DataFlow::localFlow(DataFlow::exprNode(sel), DataFlow::exprNode(conv.getOperand())) and
    sel.getSelector().getName() = "KnownTokenbridgeEmitters"
  )
  or
  exists(FuncDecl f, IndexExpr idx, SelectorExpr sel |
    conv.getEnclosingFunction() = f and
    f.getFile().getRelativePath().matches("%/notary/admincommands.go") and
    f.getName() = "createTestMessagePublication" and
    DataFlow::localFlow(DataFlow::exprNode(idx), DataFlow::exprNode(conv.getOperand())) and
    idx.getBase().stripParens() = sel and
    sel.getSelector().getName() = "KnownTokenbridgeEmitters"
  )
}

predicate isLenCallFor(CallExpr lenCall, Expr e) {
  lenCall.getTarget().getName() = "len" and
  (
    lenCall.getArgument(0) = e
    or
    DataFlow::localFlow(DataFlow::exprNode(lenCall.getArgument(0)), DataFlow::exprNode(e))
    or
    DataFlow::localFlow(DataFlow::exprNode(e), DataFlow::exprNode(lenCall.getArgument(0)))
  )
}

predicate exact32LengthConditionFor(Expr condition, Expr e, boolean truth) {
  truth = true and
  exists(EqExpr eq, CallExpr lenCall |
    condition = eq and
    (
      isLenCallFor(lenCall, e) and lenCall = eq.getLeftOperand() and eq.getRightOperand().getIntValue() = 32
      or
      isLenCallFor(lenCall, e) and lenCall = eq.getRightOperand() and eq.getLeftOperand().getIntValue() = 32
    )
  )
  or
  truth = false and
  exists(NeqExpr neq, CallExpr lenCall |
    condition = neq and
    (
      isLenCallFor(lenCall, e) and lenCall = neq.getLeftOperand() and neq.getRightOperand().getIntValue() = 32
      or
      isLenCallFor(lenCall, e) and lenCall = neq.getRightOperand() and neq.getLeftOperand().getIntValue() = 32
    )
  )
}

predicate hasExact32LengthGuard(Expr e, AstNode use) {
  exists(ControlFlow::ConditionGuardNode guard, Expr condition, boolean truth, ControlFlow::Node useNode |
    exact32LengthConditionFor(condition, e, truth) and
    guard.ensures(DataFlow::exprNode(condition), truth) and
    (
      use instanceof Expr and
      guard.dominates(DataFlow::exprNode(use.(Expr)).getBasicBlock())
      or
      useNode.isFirstNodeOf(use) and
      guard.dominates(useNode.getBasicBlock())
    )
  )
}

predicate cosmwasmDecodedEmitterSource(Expr decoded, FuncDecl f) {
  exists(CallExpr decode, Parameter p, Ident pRead |
    p = f.getParameter(0) and
    p.getType().getName() = "string" and
    pRead.refersTo(p) and
    pRead.getEnclosingFunction() = f and
    decode.getTarget().getName() = "DecodeString" and
    decode.getTarget().getPackage().getName() = "hex" and
    DataFlow::localFlow(DataFlow::exprNode(pRead), DataFlow::exprNode(decode.getArgument(0))) and
    DataFlow::localFlow(DataFlow::extractTupleElement(DataFlow::exprNode(decode), 0), DataFlow::exprNode(decoded))
  )
}

predicate isPublicRpcMessageIdEmitterAddressLookup(Expr e) {
  exists(SelectorExpr emitterAddress, SelectorExpr messageId |
    e.stripParens() = emitterAddress and
    emitterAddress.getSelector().getName() = "EmitterAddress" and
    emitterAddress.getBase().stripParens() = messageId and
    messageId.getSelector().getName() = "MessageId"
  )
}

predicate publicRpcMessageIdEmitterAddressDecodeFlowsTo(Expr copied, FuncDecl f) {
  exists(CallExpr decode |
    decode.getEnclosingFunction() = f and
    decode.getTarget().getName() = "DecodeString" and
    decode.getTarget().getPackage().getName() = "hex" and
    isPublicRpcMessageIdEmitterAddressLookup(decode.getArgument(0)) and
    DataFlow::localFlow(DataFlow::extractTupleElement(DataFlow::exprNode(decode), 0), DataFlow::exprNode(copied))
  )
}

predicate isPublicRpcExact32MessageIdCopy(CallExpr call) {
  exists(FuncDecl f |
    call.getEnclosingFunction() = f and
    f.getFile().getRelativePath().matches("%/publicrpc/publicrpcserver.go") and
    f.getName() = "GetSignedVAA" and
    call.getTarget().getName() = "copy" and
    publicRpcMessageIdEmitterAddressDecodeFlowsTo(call.getArgument(1), f) and
    hasExact32LengthGuard(call.getArgument(1), call)
  )
}

predicate stringParameterFlowsToBytesToAddressArg(FuncDecl f, Expr bytesArg) {
  exists(Parameter p, Ident pRead |
    p = f.getParameter(_) and
    p.getType().getName() = "string" and
    pRead.refersTo(p) and
    pRead.getEnclosingFunction() = f and
    DataFlow::localFlow(DataFlow::exprNode(pRead), DataFlow::exprNode(bytesArg))
  )
  or
  exists(Parameter p, Ident pRead, CallExpr decode |
    p = f.getParameter(_) and
    p.getType().getName() = "string" and
    pRead.refersTo(p) and
    pRead.getEnclosingFunction() = f and
    decode.getTarget().getName() = "DecodeString" and
    decode.getTarget().getPackage().getName() = "hex" and
    DataFlow::localFlow(DataFlow::exprNode(pRead), DataFlow::exprNode(decode.getArgument(0))) and
    DataFlow::localFlow(DataFlow::extractTupleElement(DataFlow::exprNode(decode), 0), DataFlow::exprNode(bytesArg))
  )
}

predicate isExplicitTypedChainAdapter(Expr e) {
  exists(CallExpr call | e = call and (isEvmPadAddressCopy(call) or isAptosUint64SenderCopy(call) or isNearExact32EmitterDigestCopy(call) or isXrplAccountCopy(call) or isCosmwasmCoreStringToAddressCopy(call) or isPublicRpcExact32MessageIdCopy(call)))
  or
  exists(ConversionExpr conv | e = conv and (isSuiTypedArrayConversion(conv) or isNotaryKnownEmitterConversion(conv)))
}

predicate isFullVaaIdAddressComponent(Expr e) {
  exists(IndexExpr component, CallExpr split |
    component.getIndex().getIntValue() = 1 and
    split.getCalleeName() = "Split" and
    split.getNumArgument() = 2 and
    split.getArgument(1).getStringValue() = "/" and
    (
      exists(Ident idRead |
        DataFlow::localFlow(DataFlow::exprNode(idRead), DataFlow::exprNode(split.getArgument(0))) and
        idRead.getName().regexpMatch(".*(vaa|Vaa|VAA|id|ID|key|Key).*")
      )
      or
      exists(SelectorExpr sel |
        DataFlow::localFlow(DataFlow::exprNode(sel), DataFlow::exprNode(split.getArgument(0))) and
        sel.getSelector().getName().regexpMatch(".*(vaa|Vaa|VAA|id|ID|key|Key).*")
      )
    ) and
    DataFlow::localFlow(DataFlow::exprNode(split), DataFlow::exprNode(component.getBase())) and
    DataFlow::localFlow(DataFlow::exprNode(component), DataFlow::exprNode(e))
  )
}

predicate isIdentityStructType(Type t) {
  t.getName() in ["MessagePublication", "VAA", "VAAID", "tokenBridgeKey"]
}

predicate isIdentityFieldSink(Expr sink, string sinkName) {
  exists(CompositeLit lit, int i, KeyValueExpr kv |
    kv = lit.getElement(i) and
    sink = kv.getValue() and
    kv.getKey().(Ident).getName() in ["EmitterAddress", "emitterAddr", "targetAddress"] and
    isVaaAddressExpr(sink) and
    isIdentityStructType(lit.getType()) and
    sinkName = kv.getKey().(Ident).getName()
  )
}

predicate isIdentityReturnSink(Expr sink, string sinkName) {
  exists(ReturnStmt ret |
    ret.getAnExpr() = sink and
    isVaaAddressExpr(sink) and
    sinkName = "vaa.Address return"
  )
}

predicate isIdentitySink(Expr sink, string sinkName) {
  isIdentityFieldSink(sink, sinkName)
  or
  isIdentityReturnSink(sink, sinkName)
}

predicate flowsToIdentitySink(Expr e, string sinkName) {
  exists(Expr sink |
    isIdentitySink(sink, sinkName) and
    DataFlow::localFlow(DataFlow::exprNode(e), DataFlow::exprNode(sink))
  )
}

predicate helperCallResultFlowsToIdentitySink(CallExpr call, string sinkName) {
  exists(Expr sink |
    isIdentitySink(sink, sinkName) and
    DataFlow::localFlow(DataFlow::extractTupleElement(DataFlow::exprNode(call), 0), DataFlow::exprNode(sink))
  )
}

predicate isDirectVaaAddressConversion(ConversionExpr conv) {
  isProductionNodeFile(conv.getFile()) and
    isVaaAddressExpr(conv) and
    isByteLikeExpr(conv.getOperand()) and
    not isCanonicalAddressResult(conv) and
    not isFullVaaIdAddressComponent(conv.getOperand()) and
    not isTypedInternalAddressRead(conv.getOperand()) and
  not isExplicitTypedChainAdapter(conv)
}

predicate copyDestBase(CallExpr call, Expr base) {
  call.getTarget().getName() = "copy" and
  (
    base = call.getArgument(0).stripParens().(SliceExpr).getBase().stripParens()
    or
    base = call.getArgument(0).stripParens().(IndexExpr).getBase().stripParens()
  )
}

predicate isManualVaaAddressCopy(CallExpr call, Expr base) {
  isProductionNodeFile(call.getFile()) and
  copyDestBase(call, base) and
  isVaaAddressExpr(base) and
  not isCanonicalAddressResult(base) and
  not isTypedInternalAddressRead(base) and
  not isExplicitTypedChainAdapter(call)
}

predicate returnsExpr(FuncDecl f, Expr e) {
  exists(ReturnStmt ret | ret.getEnclosingFunction() = f and ret.getExpr(0) = e)
}

predicate isCanonicalAddressWrapper(FuncDecl f) {
  isProductionNodeFile(f.getFile()) and
  exists(ReturnStmt ret, CallExpr call |
    ret.getEnclosingFunction() = f and
    ret.getExpr(0) = call and
    isCanonicalAddressCall(call) and
    (
      isCanonicalStringAddressCall(call)
      or
      not exists(Parameter p | p = f.getParameter(_) and p.getType().getName() = "string")
    )
  ) and
  not exists(ReturnStmt ret |
    ret.getEnclosingFunction() = f and
    not exists(CallExpr call |
      ret.getExpr(0) = call and
      isCanonicalAddressCall(call) and
      (
        isCanonicalStringAddressCall(call)
        or
        not exists(Parameter p | p = f.getParameter(_) and p.getType().getName() = "string")
      )
    )
  )
}

predicate isStringToBytesAddressWrapper(FuncDecl f) {
  isProductionNodeFile(f.getFile()) and
  exists(ReturnStmt ret, CallExpr call |
    ret.getEnclosingFunction() = f and
    ret.getExpr(0) = call and
    isCanonicalBytesAddressCall(call) and
    stringParameterFlowsToBytesToAddressArg(f, call.getArgument(0))
  )
}

predicate isUnsafeAddressHelper(FuncDecl f) {
  isProductionNodeFile(f.getFile()) and
  not isCanonicalAddressWrapper(f) and
  (
    exists(ConversionExpr conv | isDirectVaaAddressConversion(conv) and returnsExpr(f, conv))
    or
    exists(CallExpr copyCall, Expr base |
      isManualVaaAddressCopy(copyCall, base) and
      returnsExpr(f, base)
    )
    or
    isStringToBytesAddressWrapper(f)
  )
}

predicate isUnsafeAddressHelperCall(CallExpr call, string sinkName) {
  isProductionNodeFile(call.getFile()) and
  call.getTarget().getFuncDecl() = any(FuncDecl f | isUnsafeAddressHelper(f)) and
  helperCallResultFlowsToIdentitySink(call, sinkName) and
  not isExplicitTypedChainAdapter(call)
}

from AstNode report, string sinkName
where
  exists(ConversionExpr conv |
    isDirectVaaAddressConversion(conv) and
    flowsToIdentitySink(conv, sinkName) and
    report = conv
  )
  or
  exists(CallExpr call, Expr base |
    isManualVaaAddressCopy(call, base) and
    flowsToIdentitySink(base, sinkName) and
    report = call
  )
  or
  exists(CallExpr call |
    isUnsafeAddressHelperCall(call, sinkName) and
    report = call
  )
select report, "External Wormhole address data must be normalized with vaa.StringToAddress or vaa.BytesToAddress before use as " + sinkName + "; this conversion bypasses canonical left-padding, 0x handling, and overlength rejection."
