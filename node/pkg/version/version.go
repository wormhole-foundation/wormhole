package version

// Wormhole release version injected by the compiler.
var version = "development"

func Version() string {
	if version == "" {
		panic("binary compiled with empty version")
	}
	return version
}
