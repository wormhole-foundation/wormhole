// Command wormhole-lint runs every custom linter under rules/ as a single
// standalone binary. Each rules/<linter>/ exports an Analyzer; add new
// linters by appending one import + one Analyzer below.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/certusone/wormhole/linters/rules/channelcheck"
)

func main() {
	multichecker.Main(
		channelcheck.Analyzer,
	)
}
