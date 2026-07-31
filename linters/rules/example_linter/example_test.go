package example

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestExample(t *testing.T) {
	// The fixture contains no `// want "..."` comments — the linter is a
	// no-op, so analysistest should see zero diagnostics.
	analysistest.Run(t, analysistest.TestData(), Analyzer, "./...")
}
