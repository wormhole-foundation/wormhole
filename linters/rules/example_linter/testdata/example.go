package testdata

// Fixture for the example linter. It contains no offenders because the
// example linter never flags anything; this file exists only to give
// `go test` something to load when running `analysistest`.

func Hello() string {
	return "hello"
}
