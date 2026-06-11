package channelcheck

/*
	Strategy: Identify proper SendStmt's early. If we don't see it within a Select, then it's not being used correctly.
	 Or, we simply MISS it, making it a false positive which is fine. We'd rather fail open and get the user to
	 use nolints than miss a potential bug altogether.

	Steps for blocking channel send:
	- Find a SelectStmt
	- Check if Comm Clause:
		- See if it has a 'ast.SendStmt'.
		- If it does, then check 'default' or 'ticker' types there too.
	- If we find SelectStmt otherwise, it must be non-blocking and we want to flag it.
*/
import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/constant"
	"go/printer"
	"go/token"
	"go/types"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

type ChannelCheckPlugin struct {
	settings Settings
}

// Settings holds the configuration for the channelcheck linter.
type Settings struct {
	CheckUnbufferedChannels bool     // Enable/disable checking for unbuffered channel creation.
	CheckBufferAmount       uint64   // The amount that can be in a buffer. 0 means don't do this check.
	CheckBlockingSends      bool     // Enable/disable checking for blocking sends without default/timeout.
	IgnoreChannelsByName    []string // Channel/field names whose direct sends are exempt from the blocking-send check.

	// ignoreChannelNames is the lookup form of IgnoreChannelsByName, built
	// from the slice during configuration.
	ignoreChannelNames map[string]bool
}

// EscapeKind classifies a non-send CommClause in a select, describing what
// kind of escape valve (if any) it provides for a sibling send.
type EscapeKind int

const (
	EscapeNone            EscapeKind = iota
	EscapeTimer                      // receive on <-chan time.Time
	EscapeDefaultWithCode            // default: with non-empty body
	EscapeEmptyDefault               // default: with empty body
	EscapeContextDone                // receive on a call to (context.Context).Done()
	EscapeOther                      // any other clause in the statement
)

// SelectAnalysis is the result of inspecting a select statement.
type SelectAnalysis struct {
	Sends           []*ast.SendStmt // SendStmts that are direct CommClause heads
	Escapes         []EscapeKind    // one entry per non-send clause (excludes EscapeNone)
	EmptyDefaultPos token.Pos       // position of an empty default clause, or NoPos if none
	ContextDonePos  token.Pos       // position of a <-ctx.Done() clause, or NoPos if none
}

var Analyzer = &analysis.Analyzer{
	Name:  "channelcheck",
	Doc:   "reports channel blocking issues",
	Run:   run,
	Flags: flagSet,
}

// Flags for the analyzer
var flagSet flag.FlagSet

// Global structure to store the variables in
var settings Settings

func New(settings_new any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Settings](settings_new)
	if err != nil {
		return nil, err
	}
	settings.CheckBlockingSends = s.CheckBlockingSends
	settings.CheckBufferAmount = s.CheckBufferAmount
	settings.CheckUnbufferedChannels = s.CheckUnbufferedChannels
	settings.IgnoreChannelsByName = s.IgnoreChannelsByName
	settings.ignoreChannelNames = buildIgnoreSet(s.IgnoreChannelsByName)

	return &ChannelCheckPlugin{settings: s}, nil
}

func buildIgnoreSet(names []string) map[string]bool {
	if len(names) == 0 {
		return nil
	}
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}

func (f *ChannelCheckPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		{
			Name:  "channelcheck",
			Doc:   "reports channel blocking issues",
			Run:   run,
			Flags: flagSet,
		},
	}, nil
}

// Initialize the flags from the golangci-lint
func init() {

	flagSet.BoolVar(&settings.CheckUnbufferedChannels, "unbuffered", false, "Check for unbuffered channel creation")
	flagSet.BoolVar(&settings.CheckBlockingSends, "blocking", true, "Check for blocking sends without default/timeout")
	flagSet.Uint64Var(&settings.CheckBufferAmount, "bufferMax", 0, "Check for maximum length of channel buffer being exceeded")
	Analyzer.Flags = flagSet
	register.Plugin("channelcheck", New)
}

func (f *ChannelCheckPlugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

func run(pass *analysis.Pass) (interface{}, error) {

	for _, file := range pass.Files {
		var seenPositions = make(map[token.Pos]bool)

		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			// Fails open by design. Will
			case *ast.SelectStmt: // Select statement for channel matching

				if settings.CheckBlockingSends == false {
					break
				}
				selectAnalysis := processSelect(pass, n)

				/*
					A send is considered safe (i.e., the SendStmt-level diagnostic is
					suppressed) if its enclosing select has at least one of:
					  - Timer, DefaultWithCode, or EmptyDefault — real backpressure relief
					  - ctx.Done() — shutdown-safe; we suppress the bare blocking-send
					    diagnostic and emit a more specific ctx.Done() finding instead
					    (only when ctx.Done() is the lone escape — if a Timer/Default is
					    also present, that's already doing the real work).

					EscapeOther alone does NOT make the send safe.
				*/
				sendsAreSafe := false
				foundContext := false
				for _, escapeKind := range selectAnalysis.Escapes {
					switch escapeKind {
					case EscapeTimer, EscapeDefaultWithCode, EscapeEmptyDefault:
						sendsAreSafe = true
					case EscapeContextDone:
						foundContext = true
					}
				}

				if sendsAreSafe || foundContext {
					for _, send := range selectAnalysis.Sends {
						seenPositions[send.Pos()] = true
					}
				}

				// Collect the sends whose channels are NOT in the user's ignore list.
				// These are the anchor points for the empty-default and ctx.Done()
				// diagnostics so that a //nolint:channelcheck on the consciously-
				// blocking send line suppresses them.
				var trackedSends []*ast.SendStmt
				for _, send := range selectAnalysis.Sends {
					if name, named := sendChanName(send); named && settings.ignoreChannelNames[name] {
						continue
					}
					trackedSends = append(trackedSends, send)
				}

				// Empty default case. Only meaningful when there's a tracked send
				// in the select (a receive-only select with an empty default has no
				// backpressure concern). Anchor at each tracked send so users can
				// suppress with a //nolint next to the blocking send.
				if selectAnalysis.EmptyDefaultPos != token.NoPos {
					for _, send := range trackedSends {
						pass.Reportf(send.Pos(),
							"empty default case in channel select. Please add logging to it on failure")
					}
				}

				// ctx.Done() was the only thing found alongside the send. Flag it.
				// Only fire when there's a send in the select — a receive-only select
				// with ctx.Done() is the canonical "wait or shutdown" idiom and has
				// no backpressure concern. Anchor the diagnostic at each tracked send so that
				// a //nolint:channelcheck next to the consciously-blocking send works.
				if !sendsAreSafe && foundContext && selectAnalysis.ContextDonePos != token.NoPos {
					for _, send := range trackedSends {
						pass.Reportf(send.Pos(),
							"ctx.Done() in channel select not backfill safe. Consider adding a timer, or default statement.")
					}
				}

			// Most of the work is done in the previous case statement.
			case *ast.SendStmt:

				if settings.CheckBlockingSends == false {
					break
				}

				// If the SendStmt was NOT found within a Select clause, then add a linter error.
				tokenId := n.Pos()
				if _, ok := seenPositions[tokenId]; !ok {
					if name, named := sendChanName(n); named && settings.ignoreChannelNames[name] {
						return true
					}
					pass.Reportf(tokenId, "Blocking send. Add timer, ticker, or default case: %q", render(pass.Fset, n))
				}
				return true
			case *ast.CallExpr:
				// Channel creation that's unbuffered
				didCreateChannelWithoutBuffering, bufferAmount := checkChannelCreation(pass, n)
				if didCreateChannelWithoutBuffering && settings.CheckUnbufferedChannels {
					pass.Reportf(n.Pos(), "unbuffered channel creation detected - consider specifying buffer size %q", render(pass.Fset, n))
				}

				if settings.CheckBufferAmount > 0 && bufferAmount > 0 && uint64(bufferAmount) > settings.CheckBufferAmount {
					pass.Reportf(n.Pos(), "channel buffer size exceeds the specified limit %q", render(pass.Fset, n))
				}
				return true

			default:
				return true // Continue traversing for other node types
			}

			return true
		})
	}

	return nil, nil
}

/*
Walks a select statement and classifies each CommClause.

Sends that are the direct Comm of a CommClause are recorded in SendPositions.
Every other clause is classified via classifyClause and appended to Escapes.
An empty default body is additionally recorded by position so the caller can
emit a separate finding for it.

NOTE: A send is always a candidate for a finding unless its enclosing select has a
recognized escape (Timer, DefaultWithCode, or EmptyDefault). Fails open by design.
*/
func processSelect(pass *analysis.Pass, selectStmt *ast.SelectStmt) SelectAnalysis {
	var analysis SelectAnalysis
	for _, clause := range selectStmt.Body.List {
		commClause, ok := clause.(*ast.CommClause)
		if !ok {
			continue // Skip if not a CommClause (e.g., a declaration inside the select)
		}
		if sendNode, isSend := commClause.Comm.(*ast.SendStmt); isSend {
			analysis.Sends = append(analysis.Sends, sendNode)
			continue
		}
		clauseKind := classifyClause(pass, commClause)
		if clauseKind == EscapeNone {
			continue
		}
		analysis.Escapes = append(analysis.Escapes, clauseKind)
		if clauseKind == EscapeEmptyDefault {
			analysis.EmptyDefaultPos = commClause.Pos()
		}
		if clauseKind == EscapeContextDone {
			analysis.ContextDonePos = commClause.Pos()
		}
	}
	return analysis
}

// classifyClause returns the EscapeKind for a non-send CommClause.
// Returns EscapeNone for SendStmt clauses (they aren't escapes themselves).
func classifyClause(pass *analysis.Pass, commClause *ast.CommClause) EscapeKind {
	if commClause.Comm == nil {
		if len(commClause.Body) == 0 {
			return EscapeEmptyDefault
		}
		return EscapeDefaultWithCode
	}
	if _, isSend := commClause.Comm.(*ast.SendStmt); isSend {
		return EscapeNone
	}
	channelExpr := extractRecvChannel(commClause.Comm)
	if channelExpr == nil {
		return EscapeOther
	}
	if isContextDoneCall(pass, channelExpr) {
		return EscapeContextDone
	}
	elementType := recvElementType(pass, channelExpr)
	if elementType != nil && isNamedType(elementType, "time", "Time") {
		return EscapeTimer
	}
	return EscapeOther
}

// isContextDoneCall reports whether expr is a call to (context.Context).Done().
func isContextDoneCall(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if selector.Sel.Name != "Done" {
		return false
	}
	if pass.TypesInfo == nil {
		return false
	}
	methodObj, ok := pass.TypesInfo.Uses[selector.Sel].(*types.Func)
	if !ok {
		return false
	}
	methodPkg := methodObj.Pkg()
	return methodPkg != nil && methodPkg.Path() == "context"
}

// extractRecvChannel returns the channel expression of a receive embedded in
// any of the three CommClause shapes:
//
//	case <-ch:           (*ast.ExprStmt wrapping *ast.UnaryExpr)
//	case v := <-ch:      (*ast.AssignStmt, one RHS)
//	case v, ok := <-ch:  (*ast.AssignStmt, one RHS, two LHS)
//
// Returns nil if the statement is not a receive.
func extractRecvChannel(commStmt ast.Stmt) ast.Expr {
	var receiveExpr ast.Expr
	switch typedStmt := commStmt.(type) {
	case *ast.ExprStmt:
		receiveExpr = typedStmt.X
	case *ast.AssignStmt:
		if len(typedStmt.Rhs) != 1 {
			return nil
		}
		receiveExpr = typedStmt.Rhs[0]
	default:
		return nil
	}
	unaryExpr, ok := receiveExpr.(*ast.UnaryExpr)
	if !ok || unaryExpr.Op != token.ARROW {
		return nil
	}
	return unaryExpr.X
}

// recvElementType returns the element type of the channel being received from,
// or nil if type info is unavailable or the expression is not a channel.
func recvElementType(pass *analysis.Pass, channelExpr ast.Expr) types.Type {
	if pass.TypesInfo == nil {
		return nil
	}
	channelExprType := pass.TypesInfo.TypeOf(channelExpr)
	if channelExprType == nil {
		return nil
	}
	channelType, ok := channelExprType.Underlying().(*types.Chan)
	if !ok {
		return nil
	}
	return channelType.Elem()
}

// isNamedType reports whether t is the named type pkgPath.name (e.g. "time", "Time").
func isNamedType(candidateType types.Type, pkgPath, typeName string) bool {
	namedType, ok := candidateType.(*types.Named)
	if !ok {
		return false
	}
	typeObj := namedType.Obj()
	if typeObj == nil || typeObj.Pkg() == nil {
		return false
	}
	return typeObj.Pkg().Path() == pkgPath && typeObj.Name() == typeName
}

func checkChannelCreation(pass *analysis.Pass, node *ast.CallExpr) (bool, int64) {
	fun, ok := node.Fun.(*ast.Ident)
	if !ok || fun == nil || fun.Name != "make" {
		return false, 0
	}

	if len(node.Args) > 0 {
		if _, ok := node.Args[0].(*ast.ChanType); ok { // It's a channel
			if len(node.Args) == 1 {
				return true, 0 // Unbuffered channel
			}

			if len(node.Args) == 2 {
				// Evaluate the buffer size expression
				bufferSize, err := evalBufferSize(pass, node.Args[1])
				if err != nil {
					// Has a buffer arg but the size is not statically determinable
					// (e.g. a runtime variable). Don't flag it as unbuffered and
					// don't check against the max — the size is just unknown.
					return false, 0
				}

				// make(chan T, 0) is semantically identical to make(chan T) —
				// both produce an unbuffered channel with synchronous rendezvous.
				if bufferSize == 0 {
					return true, 0
				}
				return false, int64(bufferSize)
			}
		}
	}

	return false, 0
}

/*
Evaluates the buffer size expression in a make(chan T, N) call. Returns the
constant value of N if it is statically determinable — which covers integer
literals (`make(chan T, 100)`), named constants (`const N = 100; make(chan T, N)`),
and constant arithmetic (`make(chan T, 2*N+1)`).

Returns an error for runtime values (`var n = 100; make(chan T, n)`) which
cannot be evaluated at lint time.
*/
func evalBufferSize(pass *analysis.Pass, expr ast.Expr) (uint64, error) {
	if pass.TypesInfo == nil {
		return 0, fmt.Errorf("type info unavailable")
	}
	typeAndValue, ok := pass.TypesInfo.Types[expr]
	if !ok || typeAndValue.Value == nil {
		return 0, fmt.Errorf("buffer size is not a constant expression")
	}
	intValue := constant.ToInt(typeAndValue.Value)
	if intValue.Kind() != constant.Int {
		return 0, fmt.Errorf("buffer size is not an integer: %v", typeAndValue.Value.Kind())
	}
	bufferSize, exact := constant.Uint64Val(intValue)
	if !exact {
		return 0, fmt.Errorf("buffer size is too large or negative")
	}
	return bufferSize, nil
}

// render returns the pretty-print of the given node
func render(fset *token.FileSet, x interface{}) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, x); err != nil {
		panic(err)
	}
	return buf.String()
}

// sendChanName returns the receiving channel's variable name for a send
// statement (the channel on the left of `<-`). The second return value is
// false when the Chan expression has no meaningful single name — e.g.
// `channels[0] <- x`, `chFunc() <- x`, or a type assertion `iface.(chan T) <- x`.
// For those, callers should fall back to render(fset, sendStmt.Chan).
//
// Supported shapes:
//
//	ch <- x         → "ch"        (*ast.Ident)
//	s.eventCh <- x  → "eventCh"   (*ast.SelectorExpr; returns the field name)
//	(ch) <- x       → "ch"        (*ast.ParenExpr; unwrapped recursively)
func sendChanName(sendStmt *ast.SendStmt) (string, bool) {
	chanExpr := sendStmt.Chan
	for {
		switch typedExpr := chanExpr.(type) {
		case *ast.Ident:
			return typedExpr.Name, true
		case *ast.SelectorExpr:
			return typedExpr.Sel.Name, true
		case *ast.ParenExpr:
			chanExpr = typedExpr.X
			continue
		default:
			return "", false
		}
	}
}
