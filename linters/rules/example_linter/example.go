// Package example is a no-op reference linter. It demonstrates the minimum
// structure needed to plug a new analyzer into both the wormhole-lint
// aggregator (cmd/wormhole-lint) and the custom golangci-lint binary
// (.custom-gcl.yml). Copy this package as a starting point when adding a
// real linter under rules/<your_linter>/.
package example

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

// Settings is the per-invocation configuration block passed in via
// golangci-lint's `linters.settings.custom.example.settings:` map. Fields
// here become user-facing tunables; an empty struct is fine for a no-op.
type Settings struct{}

// Analyzer is the standalone entrypoint consumed by cmd/wormhole-lint.
// Keep Name unique across all rules/<linter>/ analyzers — golangci-lint
// addresses the plugin by this name in .golangci.yml.
var Analyzer = &analysis.Analyzer{
	Name: "example",
	Doc:  "no-op reference linter; never reports diagnostics",
	Run:  run,
}

func run(_ *analysis.Pass) (any, error) {
	// Intentionally empty: this linter never flags anything. A real linter
	// walks pass.Files and calls pass.Reportf / pass.Report on offenders.
	return nil, nil
}

// Plugin is the type returned to golangci-lint's module-plugin loader. One
// plugin can expose multiple analyzers via BuildAnalyzers.
type Plugin struct {
	settings Settings
}

// New is the constructor registered with golangci-lint. It receives the
// raw settings map from .golangci.yml and decodes it into Settings.
func New(raw any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Settings](raw)
	if err != nil {
		return nil, err
	}
	return &Plugin{settings: s}, nil
}

// BuildAnalyzers returns the analyzers this plugin contributes. Add more
// entries here if your linter ships multiple checks.
func (p *Plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

// GetLoadMode tells golangci-lint how much type information to load.
// LoadModeSyntax is enough for AST-only checks; LoadModeTypesInfo is
// required when you need full type resolution (recommended default).
func (p *Plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

func init() {
	// The string passed here is the name used in .golangci.yml under
	// linters.enable and linters.settings.custom.
	register.Plugin("example", New)
}
