/**
 * @name Boundary Wormhole chain ID parsed without SDK helper
 * @description Production node code should convert boundary-derived Wormhole chain IDs through the SDK chain-ID helpers instead of local casts or parsing.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/canonical-chain-id-parsing
 * @tags security
 *       external/cwe/cwe-20
 */

import go
import semmle.go.concepts.GeneratedFile

predicate isProductionNodeFile(File f) {
  (
    f.getRelativePath().matches("node/%.go")
    or
    f.getRelativePath().matches("pkg/%.go")
    or
    f.getRelativePath().matches("cmd/%.go")
  ) and
  not f.getRelativePath().matches("%_test.go") and
  not f.getRelativePath().matches("%.pb.go") and
  not f.getRelativePath().matches("%_grpc.pb.go") and
  not f.getRelativePath().matches("node/pkg/ethereum/abi/%.go") and
  not f.getRelativePath().matches("pkg/ethereum/abi/%.go") and
  not f instanceof GeneratedFile
}

predicate isWormholeChainIdType(Type t) {
  t.hasQualifiedName("github.com/wormhole-foundation/wormhole/sdk/vaa", "ChainID")
  or
  t.hasQualifiedName("github.com/certusone/wormhole/sdk/vaa", "ChainID")
}

predicate isSdkChainIdHelperCall(CallExpr call, string semantics) {
  exists(string pkg |
    pkg = call.getTarget().getPackage().getPath() and
    pkg in [
      "github.com/wormhole-foundation/wormhole/sdk/vaa",
      "github.com/certusone/wormhole/sdk/vaa"
    ]
  ) and
  (
    call.getTarget().getName() = "ChainIDFromNumber" and semantics = "wire-valid"
    or
    call.getTarget().getName() = "KnownChainIDFromNumber" and semantics = "registered-numeric"
    or
    call.getTarget().getName() = "StringToKnownChainID" and semantics = "registered-string"
  )
}

predicate isGeneratedProtoPackage(Package pkg) {
  pkg.getPath().matches("github.com/wormhole-foundation/wormhole/node/pkg/proto/%")
  or
  pkg.getPath().matches("github.com/certusone/wormhole/node/pkg/proto/%")
}

predicate isBoundarySchemaFieldRead(Expr e) {
  exists(SelectorExpr sel, Field field |
    e = sel and
    sel.refersTo(field) and
    isGeneratedProtoPackage(field.getPackage())
  )
}

predicate isBoundarySchemaGetterCall(Expr e) {
  exists(CallExpr call |
    e = call and
    isGeneratedProtoPackage(call.getTarget().getPackage()) and
    call.getTarget().getName().matches("Get%")
  )
}

predicate boundarySourceBase(Expr source, Expr base) {
  exists(SelectorExpr sel |
    source = sel and
    base = sel.getBase()
  )
  or
  exists(CallExpr call, SelectorExpr callee |
    source = call and
    callee = call.getCalleeExpr().stripParens() and
    base = callee.getBase()
  )
}

predicate selectorRoot(Expr e, Expr root) {
  root = e
  or
  exists(SelectorExpr sel |
    e = sel and
    selectorRoot(sel.getBase(), root)
  )
}

predicate hasBoundaryOrigin(Expr base) {
  exists(Expr root, Parameter p, Ident pRead |
    selectorRoot(base, root) and
    pRead.refersTo(p) and
    DataFlow::localFlow(DataFlow::exprNode(pRead), DataFlow::exprNode(root))
  )
  or
  exists(Expr root, Assignment assign, RecvExpr recv, Expr lhs |
    selectorRoot(base, root) and
    assign.getRhs(0) = recv and
    lhs = assign.getLhs(_) and
    DataFlow::localFlow(DataFlow::exprNode(lhs), DataFlow::exprNode(root))
  )
  or
  exists(Expr root, RecvExpr recv |
    selectorRoot(base, root) and
    DataFlow::localFlow(DataFlow::exprNode(recv), DataFlow::exprNode(root))
  )
}

predicate isIbcMessageChainIdAttributeRead(Expr e) {
  exists(CallExpr call, SelectorExpr callee |
    e = call and
    callee = call.getCalleeExpr().stripParens() and
    (
      e.getFile().getRelativePath().matches("node/pkg/watchers/ibc/%.go") or
      e.getFile().getRelativePath().matches("pkg/watchers/ibc/%.go")
    ) and
    call.getTarget() instanceof Method and
    call.getTarget().getName() = "GetAsUint" and
    call.getTarget().(Method).getReceiverBaseType().getName() = "WasmAttributes" and
    call.getArgument(0).getStringValue() = "message.chain_id" and
    call.getArgument(1).getIntValue() = 16
  )
}

predicate isIbcChannelChainsSelector(Expr e) {
  exists(SelectorExpr sel, Field field |
    e = sel and
    sel.refersTo(field) and
    field.getName() = "ChannelChains"
  )
}

predicate isJsonDecodedNumberRead(Expr e) {
  exists(TypeAssertExpr assertion, IndexExpr idx, RangeStmt loop, Ident rangeValue, Ident valueRead, Variable v |
    e = assertion and
    (
      e.getFile().getRelativePath().matches("node/pkg/watchers/ibc/%.go") or
      e.getFile().getRelativePath().matches("pkg/watchers/ibc/%.go")
    ) and
    assertion.getExpr() = idx and
    idx.getIndex().getIntValue() = 1 and
    rangeValue = loop.getValue() and
    rangeValue.refersTo(v) and
    idx.getBase() = valueRead and
    valueRead.refersTo(v) and
    isIbcChannelChainsSelector(loop.getDomain()) and
    assertion.getTypeExpr().toString() = "float64"
  )
}

predicate isRangeValueFromParameter(Expr e) {
  exists(RangeStmt loop, Parameter p, Ident pRead, Ident rangeValue, Ident valueRead, Variable v |
    rangeValue = loop.getValue() and
    rangeValue.refersTo(v) and
    e = valueRead and
    (
      e.getFile().getRelativePath().matches("node/pkg/txverifier/%.go") or
      e.getFile().getRelativePath().matches("pkg/txverifier/%.go")
    ) and
    valueRead.refersTo(v) and
    pRead.refersTo(p) and
    DataFlow::localFlow(DataFlow::exprNode(pRead), DataFlow::exprNode(loop.getDomain()))
  )
}

predicate isBoundaryChainIdSource(Expr e) {
  isProductionNodeFile(e.getFile()) and
  (
    ((isBoundarySchemaFieldRead(e) or isBoundarySchemaGetterCall(e)) and
      exists(Expr base | boundarySourceBase(e, base) and hasBoundaryOrigin(base)))
    or
    isIbcMessageChainIdAttributeRead(e)
    or
    isJsonDecodedNumberRead(e)
    or
    isRangeValueFromParameter(e)
  )
}

predicate boundarySourceFlowsTo(Expr source, Expr sink) {
  isBoundaryChainIdSource(source) and
  (
    DataFlow::localFlow(DataFlow::exprNode(source), DataFlow::exprNode(sink))
    or
    (isIbcMessageChainIdAttributeRead(source) or isJsonDecodedNumberRead(source)) and
    DataFlow::localFlow(
      DataFlow::extractTupleElement(DataFlow::exprNode(source), 0), DataFlow::exprNode(sink)
    )
  )
}

predicate isDirectChainIdConversion(ConversionExpr conv) {
  isProductionNodeFile(conv.getFile()) and
  isWormholeChainIdType(conv.getType())
}

predicate isStructFieldContext(ConversionExpr conv) {
  exists(KeyValueExpr fieldInit |
    conv = fieldInit.getValue()
    or
    conv = fieldInit.getValue().getAChild*()
  )
}

predicate isCallArgumentContext(ConversionExpr conv) {
  exists(CallExpr call, int i |
    conv = call.getArgument(i)
    or
    conv = call.getArgument(i).getAChild*()
  )
}

predicate isIndexContext(ConversionExpr conv) {
  exists(IndexExpr idx |
    conv = idx.getIndex()
    or
    conv = idx.getIndex().getAChild*()
  )
}

predicate isComparisonContext(ConversionExpr conv) {
  exists(ComparisonExpr cmp |
    conv = cmp.getLeftOperand()
    or
    conv = cmp.getLeftOperand().getAChild*()
    or
    conv = cmp.getRightOperand()
    or
    conv = cmp.getRightOperand().getAChild*()
  )
}

predicate isAssignmentContext(ConversionExpr conv) {
  exists(Assignment assign |
    conv = assign.getRhs(_)
    or
    conv = assign.getRhs(_).getAChild*()
  )
}

predicate hasChainIdUseContext(ConversionExpr conv) {
  isStructFieldContext(conv)
  or
  isCallArgumentContext(conv)
  or
  isIndexContext(conv)
  or
  isComparisonContext(conv)
  or
  isAssignmentContext(conv)
}

predicate bypassesSdkChainIdHelper(ConversionExpr conv, Expr source) {
  isDirectChainIdConversion(conv) and
  hasChainIdUseContext(conv) and
  boundarySourceFlowsTo(source, conv.getOperand()) and
  not exists(CallExpr helper, string semantics |
    isSdkChainIdHelperCall(helper, semantics) and
    DataFlow::localFlow(DataFlow::exprNode(source), DataFlow::exprNode(helper.getAnArgument())) and
    DataFlow::localFlow(DataFlow::exprNode(helper), DataFlow::exprNode(conv.getOperand()))
  )
}

from ConversionExpr conv
where exists(Expr source | bypassesSdkChainIdHelper(conv, source))
select conv,
  "Convert boundary-derived Wormhole chain ID through `vaa.ChainIDFromNumber`, `vaa.KnownChainIDFromNumber`, or `vaa.StringToKnownChainID` according to whether this context needs wire-valid or registered-chain semantics."
