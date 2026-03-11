package main

import (
	"github.com/mgechev/revive/cli"
	"github.com/mgechev/revive/lint"
	"github.com/mgechev/revive/revivelib"
)

func main() {
	// Create our custom rule
	alreadyLockedRule := &AlreadyLockedRule{}

	// Run revive with our custom rule added
	cli.RunRevive(revivelib.NewExtraRule(alreadyLockedRule, lint.RuleConfig{}))
}
