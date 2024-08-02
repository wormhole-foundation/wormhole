.PHONY test
test: 
# Use this command on amd64 systems
	go test -v -ldflags '-extldflags "-Wl,--allow-multiple-definition" ' ./...

.PHONY test-arm64
test-arm64:
# Use this command on arm64, other you will encounter linker errors.
# It's not perfect: it will fail due to 'undefined symbols' errors
# for packges using cgo. Still, it will get you farther than running
# the default command.
# To test a single package, use these -ldflags with e.g. ./pkg/governor
	go test -ldflags '-extldflags "-Wl,-ld_classic " ' ./...
