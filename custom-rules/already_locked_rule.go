package main

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/mgechev/revive/lint"
)

// AlreadyLockedRule checks for proper mutex usage with alreadyLocked functions
type AlreadyLockedRule struct{}

// Name returns the rule name
func (r *AlreadyLockedRule) Name() string {
	return "already-locked-checker"
}

// Apply applies the rule to the given file
func (r *AlreadyLockedRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	var failures []lint.Failure

	// Find all function declarations and check them
	for _, decl := range file.AST.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			failures = append(failures, r.checkFunction(funcDecl, file)...)
		}
	}

	return failures
}

// checkFunction checks a single function for alreadyLocked calls
func (r *AlreadyLockedRule) checkFunction(funcDecl *ast.FuncDecl, file *lint.File) []lint.Failure {
	var failures []lint.Failure

	if funcDecl.Body == nil {
		return failures
	}

	// Check if this function itself is an "alreadyLocked" function
	isAlreadyLockedFunction := r.isAlreadyLockedFunction(funcDecl)

	// Check if this function has mutex locking
	hasMutexLocking := r.hasMutexLocking(funcDecl)

	// If this function is itself an "alreadyLocked" function, then calls to other
	// "alreadyLocked" functions are valid (the lock is held by the original caller)
	if isAlreadyLockedFunction {
		return failures // No need to check - this function assumes lock is already held
	}

	// Find all alreadyLocked function calls
	ast.Inspect(funcDecl, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if failure := r.checkCallExpr(call, hasMutexLocking, file); failure != nil {
				failures = append(failures, *failure)
			}
		}
		return true
	})

	return failures
}

// checkCallExpr checks if a function call to an "alreadyLocked" function has proper mutex usage
func (r *AlreadyLockedRule) checkCallExpr(call *ast.CallExpr, hasMutexLocking bool, file *lint.File) *lint.Failure {
	// Check if this is a function call with "alreadyLocked" in the name
	var funcName string
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		funcName = fun.Name
	case *ast.SelectorExpr:
		funcName = fun.Sel.Name
	default:
		return nil
	}

	// Check if function name contains "alreadyLocked" (case insensitive)
	if !strings.Contains(strings.ToLower(funcName), "alreadylocked") {
		return nil
	}

	// If no mutex locking found, report an issue
	if !hasMutexLocking {
		return &lint.Failure{
			Confidence: 1,
			Node:       call,
			Category:   "logic",
			Failure:    fmt.Sprintf("function call to '%s' should be within a mutex lock/unlock block", funcName),
		}
	}

	return nil
}

// hasMutexLocking checks if a function has mutex locking patterns
func (r *AlreadyLockedRule) hasMutexLocking(funcDecl *ast.FuncDecl) bool {
	hasLock := false
	hasUnlock := false
	hasDefer := false

	ast.Inspect(funcDecl, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if r.isLockCall(node) {
				hasLock = true
			}
			if r.isUnlockCall(node) {
				hasUnlock = true
			}
		case *ast.DeferStmt:
			if r.isUnlockCall(node.Call) {
				hasDefer = true
			}
		}
		return true
	})

	// Accept either explicit lock/unlock or defer unlock pattern
	return hasLock && (hasUnlock || hasDefer)
}

// isLockCall checks if a call expression is a mutex Lock() call
func (r *AlreadyLockedRule) isLockCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name == "Lock"
	}
	return false
}

// isUnlockCall checks if a call expression is a mutex Unlock() call
func (r *AlreadyLockedRule) isUnlockCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name == "Unlock"
	}
	return false
}

// isAlreadyLockedFunction checks if a function has "alreadyLocked" in its name
func (r *AlreadyLockedRule) isAlreadyLockedFunction(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Name == nil {
		return false
	}
	return strings.Contains(strings.ToLower(funcDecl.Name.Name), "alreadylocked")
}
