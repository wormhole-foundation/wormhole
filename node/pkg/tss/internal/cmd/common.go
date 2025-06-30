package cmd

import (
	engine "github.com/certusone/wormhole/node/pkg/tss"
)

type Identifier struct {
	Hostname string
	TlsX509  engine.PEM // PEM Encoded (see certs.go). Note, you must have the private key of this cert later.
}

type SetupConfigs struct {
	NumParticipants int
	WantedThreshold int // should be non inclusive. That is, if you have n=19,f=6, then threshold=12 (13 guardians needed to sign).

	Self       Identifier
	SelfSecret engine.PEM // PEM Encoded (see certs.go). Note, you must have the private key of this cert later.

	Peers        []Identifier
	Secrets      []engine.PEM
	SaveLocation []string
}
