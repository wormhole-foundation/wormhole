/**
 * @name Manual VAA ID parsing bypasses canonical parser
 * @description Serialized Wormhole VAA IDs should be parsed with the canonical parser before constructing storage identities.
 * @kind problem
 * @problem.severity warning
 * @precision high
 * @id wormhole/go/canonical-vaa-id-parsing
 * @tags security
 *       external/cwe/cwe-20
 */

import go
import semmle.go.concepts.GeneratedFile

predicate isProductionNodeFile(File f) {
  f.getRelativePath().matches("node/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f.getRelativePath().matches("%.pb.go") and
  not f instanceof GeneratedFile
}

predicate isCanonicalVaaIdParser(FuncDecl f) {
  f.getName() in ["VaaIDFromString", "VAAIDFromString"] and
  (
    f.getFile().getRelativePath().matches("node/pkg/db/%.go")
    or
    f.getFile().getRelativePath().matches("pkg/db/%.go")
    or
    f.getFile().getRelativePath().matches("sdk/vaa/%.go")
  )
}

predicate isInsideCanonicalVaaIdParser(Expr e) {
  isCanonicalVaaIdParser(e.getEnclosingFunction())
}

predicate isVaaIdLiteral(CompositeLit lit) {
  isProductionNodeFile(lit.getFile()) and
  lit.getType().getName() = "VAAID" and
  not isCanonicalVaaIdParser(lit.getEnclosingFunction())
}

predicate fieldInit(KeyValueExpr field, string name, Expr value) {
  field.getKey().(Ident).getName() = name and
  value = field.getValue()
}

predicate hasEmitterAddressField(CompositeLit lit, KeyValueExpr field, Expr value) {
  exists(int i |
    field = lit.getElement(i) and
    fieldInit(field, "EmitterAddress", value)
  )
}

predicate isSlashStringSplit(CallExpr call) {
  call.getCalleeName() = "Split" and
  call.getNumArgument() = 2 and
  call.getArgument(1).getStringValue() = "/"
}

predicate splitResultFlowsToBase(CallExpr split, Expr base) {
  DataFlow::localFlow(DataFlow::exprNode(split), DataFlow::exprNode(base.stripParens()))
}

predicate isEmitterAddressSplitComponent(IndexExpr idx, CallExpr split) {
  isSlashStringSplit(split) and
  idx.getIndex().getIntValue() = 1 and
  idx.getEnclosingFunction() = split.getEnclosingFunction() and
  splitResultFlowsToBase(split, idx.getBase())
}

predicate emitterAddressFieldUsesManualSplit(Expr emitterAddress, IndexExpr splitIndex, CallExpr split) {
  isEmitterAddressSplitComponent(splitIndex, split) and
  not isInsideCanonicalVaaIdParser(splitIndex) and
  DataFlow::localFlow(DataFlow::exprNode(splitIndex), DataFlow::exprNode(emitterAddress))
}

predicate isEmitterAddressComponentParser(CallExpr call) {
  call.getCalleeName() = "StringToAddress"
}

DataFlow::Node emitterAddressComponentParserValue(CallExpr call) {
  isEmitterAddressComponentParser(call) and
  (
    result = DataFlow::exprNode(call) or
    result = DataFlow::extractTupleElement(DataFlow::exprNode(call), 0)
  )
}

predicate emitterAddressFieldUsesManualComponentParser(
  Expr emitterAddress, IndexExpr splitIndex, CallExpr split
) {
  exists(CallExpr parserCall |
    isEmitterAddressSplitComponent(splitIndex, split) and
    isEmitterAddressComponentParser(parserCall) and
    parserCall.getEnclosingFunction() = split.getEnclosingFunction() and
    not isInsideCanonicalVaaIdParser(splitIndex) and
    DataFlow::localFlow(
      DataFlow::exprNode(splitIndex), DataFlow::exprNode(parserCall.getArgument(0))
    ) and
    DataFlow::localFlow(
      emitterAddressComponentParserValue(parserCall), DataFlow::exprNode(emitterAddress)
    )
  )
}

from CompositeLit lit, KeyValueExpr field, Expr emitterAddress, IndexExpr splitIndex, CallExpr split
where
  isVaaIdLiteral(lit) and
  hasEmitterAddressField(lit, field, emitterAddress) and
  (
    emitterAddressFieldUsesManualSplit(emitterAddress, splitIndex, split) or
    emitterAddressFieldUsesManualComponentParser(emitterAddress, splitIndex, split)
  )
select lit,
  "Parse serialized VAA IDs with the canonical VAA ID parser before constructing storage identities; manual split/reconstruction can misdecode emitter addresses and miss existing signed VAAs."
