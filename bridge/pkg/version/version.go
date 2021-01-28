package version

// Wormhole release version injected by the compiler.
var version = "development"

func Version() string {
	return version
}
