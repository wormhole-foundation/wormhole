/**
 * @name Delegated guardian config reaches governance serialization without required validation
 * @description Admin-supplied delegated guardian configs must strictly parse guardian addresses, reject duplicate canonical keys, and enforce non-empty threshold bounds/quorum before governance serialization.
 * @kind problem
 * @problem.severity error
 * @precision high
 * @id wormhole/go/delegated-guardian-config-validation
 * @tags security
 *       external/cwe/cwe-20
 */

import go
import semmle.go.concepts.GeneratedFile

predicate isProductionNodeFile(File f) {
  f.getRelativePath().matches("node/%.go") and
  not f.getRelativePath().matches("%_test.go") and
  not f instanceof GeneratedFile
}

predicate isDelegatedGuardianSetConfigSerialize(CallExpr call) {
  isProductionNodeFile(call.getFile()) and
  call.getCalleeName() = "Serialize" and
  call.getTarget() instanceof Method and
  call.getTarget().(Method).getReceiverBaseType().getName() = "BodyDelegatedGuardiansSetConfig" and
  call.getTarget().getPackage().getPath() = "github.com/wormhole-foundation/wormhole/sdk/vaa"
}

predicate isGoEthereumCommonCall(CallExpr call, string name) {
  call.getCalleeName() = name and
  call.getTarget().getPackage().getPath() = "github.com/ethereum/go-ethereum/common"
}

predicate hasAddressCanonicalization(FuncDecl f) {
  exists(CallExpr call |
    call.getEnclosingFunction() = f and
    isGoEthereumCommonCall(call, "HexToAddress")
  )
}

predicate hasStrictAddressParse(FuncDecl f) {
  exists(CallExpr call |
    call.getEnclosingFunction() = f and
    isGoEthereumCommonCall(call, "IsHexAddress")
  )
}

predicate hasRuntimeDelegatedGuardianValidator(FuncDecl f) {
  exists(CallExpr call |
    call.getEnclosingFunction() = f and
    call.getCalleeName() = "NewDelegatedGuardianChainConfig" and
    call.getTarget().getPackage().getPath() in [
      "github.com/wormhole-foundation/wormhole/node/pkg/processor",
      "github.com/certusone/wormhole/node/pkg/processor"
    ]
  )
}

predicate hasQuorumFloorCheck(FuncDecl f) {
  exists(CallExpr call |
    call.getEnclosingFunction() = f and
    call.getCalleeName() = "CalculateQuorum" and
    call.getTarget().getPackage().getPath() = "github.com/wormhole-foundation/wormhole/sdk/vaa"
  )
}

from CallExpr sink, FuncDecl f
where
  isDelegatedGuardianSetConfigSerialize(sink) and
  sink.getEnclosingFunction() = f and
  hasAddressCanonicalization(f) and
  (
    not hasStrictAddressParse(f) or
    not hasRuntimeDelegatedGuardianValidator(f) or
    not hasQuorumFloorCheck(f)
  )
select sink,
  "Delegated guardian config from admin input reaches governance serialization without required validation: strictly parse EVM guardian addresses before HexToAddress, reject post-canonical duplicate keys, and enforce non-empty threshold bounds/quorum before creating the VAA."
