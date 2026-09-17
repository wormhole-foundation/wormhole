/**
 * @name XRPL parser scans beyond the first memo
 * @description XRPL Wormhole Core and NTT parsing must inspect only memo index zero to avoid reinterpreting later or future memo formats as canonical Wormhole messages.
 * @kind problem
 * @problem.severity error
 * @precision high
 * @id wormhole/go/xrpl-first-memo-only
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

predicate isCoreOrNttMemoFormatSource(Expr e) {
  exists(Ident id |
    id = e.getAChild*() and
    id.getName() in ["coreMemoFormat", "nttMemoFormat"]
  )
}

predicate isCoreOrNttMemoFormatExpr(Expr e) {
  isCoreOrNttMemoFormatSource(e)
  or
  exists(FuncDecl helper, Parameter parameter, Ident parameterRead, CallExpr call, int i |
    parameter = helper.getParameter(i) and
    parameterRead.refersTo(parameter) and
    parameterRead.getEnclosingFunction() = helper and
    call.getTarget().getFuncDecl() = helper and
    isCoreOrNttMemoFormatExpr(call.getArgument(i)) and
    DataFlow::localFlow(
      DataFlow::exprNode(parameterRead), DataFlow::exprNode(e.stripParens())
    )
  )
}

predicate recognizesCoreOrNttMemoFormat(FuncDecl f) {
  isProductionXrplWatcherFile(f.getFile()) and
  exists(Expr recognition |
    recognition.getEnclosingFunction() = f and
    isCoreOrNttMemoFormatSource(recognition) and
    (
      recognition instanceof EqualityTestExpr
      or
      exists(CallExpr call, int i |
        recognition = call.getArgument(i)
      )
    )
  )
}

predicate calls(FuncDecl caller, FuncDecl callee) {
  exists(CallExpr call |
    call.getEnclosingFunction() = caller and
    call.getTarget().getFuncDecl() = callee
  )
}

predicate isReachableFromCoreOrNttRecognizer(FuncDecl f) {
  exists(FuncDecl root |
    recognizesCoreOrNttMemoFormat(root) and
    calls*(root, f)
  )
}

predicate isMemoCollectionSource(Expr e) {
  exists(SelectorExpr memos, Field field |
    e.stripParens() = memos and
    memos.refersTo(field) and
    field.getName() = "Memos" and
    memos.getBase().getType().getName() = "Transaction"
  )
  or
  exists(IndexExpr transactionMemos, FuncDecl f, Parameter parameter, Ident parameterRead |
    e.stripParens() = transactionMemos and
    transactionMemos.getIndex().getStringValue() = "Memos" and
    parameterRead.refersTo(parameter) and
    parameterRead.getEnclosingFunction() = f and
    isReachableFromCoreOrNttRecognizer(f) and
    parameter.getType().getName() = "FlatTransaction" and
    DataFlow::localFlow(
      DataFlow::exprNode(parameterRead), DataFlow::exprNode(transactionMemos.getBase().stripParens())
    )
  )
}

predicate isMemoCollection(Expr e) {
  exists(Expr source |
    isMemoCollectionSource(source) and
    DataFlow::localFlow(DataFlow::exprNode(source), DataFlow::exprNode(e.stripParens()))
  )
  or
  exists(FuncDecl helper, Parameter parameter, Ident parameterRead, CallExpr call, int i |
    parameter = helper.getParameter(i) and
    parameterRead.refersTo(parameter) and
    parameterRead.getEnclosingFunction() = helper and
    call.getTarget().getFuncDecl() = helper and
    isMemoCollection(call.getArgument(i)) and
    DataFlow::localFlow(
      DataFlow::exprNode(parameterRead), DataFlow::exprNode(e.stripParens())
    )
  )
}

predicate isFirstMemoIndex(IndexExpr idx) { idx.getIndex().getIntValue() = 0 }

predicate scansMemoCollection(FuncDecl f, RangeStmt loop) {
  loop.getEnclosingFunction() = f and
  isMemoCollection(loop.getDomain())
}

predicate usesNonFirstMemoIndex(FuncDecl f, IndexExpr idx) {
  idx.getEnclosingFunction() = f and
  isMemoCollection(idx.getBase()) and
  not isFirstMemoIndex(idx)
}

predicate isScanningHelper(FuncDecl f) {
  exists(RangeStmt loop | scansMemoCollection(f, loop))
  or
  exists(IndexExpr idx | usesNonFirstMemoIndex(f, idx))
  or
  exists(CallExpr call, FuncDecl helper |
    call.getEnclosingFunction() = f and
    call.getTarget().getFuncDecl() = helper and
    helper != f and
    isScanningHelper(helper) and
    exists(int memoArg, int formatArg |
      isMemoCollection(call.getArgument(memoArg)) and
      isCoreOrNttMemoFormatExpr(call.getArgument(formatArg))
    )
  )
}

predicate isCoreOrNttScanningHelperCall(CallExpr call, FuncDecl helper) {
  call.getTarget().getFuncDecl() = helper and
  helper != call.getEnclosingFunction() and
  isScanningHelper(helper) and
  exists(int memoArg, int formatArg |
    isMemoCollection(call.getArgument(memoArg)) and
    isCoreOrNttMemoFormatExpr(call.getArgument(formatArg))
  )
}

predicate hasCoreOrNttScanningHelperCall(FuncDecl helper) {
  exists(CallExpr call |
    isCoreOrNttScanningHelperCall(call, helper)
  )
}

from AstNode report
where
  exists(FuncDecl f, RangeStmt loop |
    isReachableFromCoreOrNttRecognizer(f) and
    (f = loop.getEnclosingFunction() and (recognizesCoreOrNttMemoFormat(f) or hasCoreOrNttScanningHelperCall(f))) and
    scansMemoCollection(f, loop) and
    report = loop
  )
  or
  exists(FuncDecl f, IndexExpr idx |
    isReachableFromCoreOrNttRecognizer(f) and
    (f = idx.getEnclosingFunction() and (recognizesCoreOrNttMemoFormat(f) or hasCoreOrNttScanningHelperCall(f))) and
    usesNonFirstMemoIndex(f, idx) and
    report = idx
  )
  or
  exists(FuncDecl f, CallExpr call, FuncDecl helper |
    recognizesCoreOrNttMemoFormat(f) and
    call.getEnclosingFunction() = f and
    isCoreOrNttScanningHelperCall(call, helper) and
    report = call
  )
select report, "XRPL Wormhole Core/NTT parsing must inspect only Memos[0]."
