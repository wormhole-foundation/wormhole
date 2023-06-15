package common

type Environment string

const (
	MainNet        Environment = "prod"
	UnsafeDevNet   Environment = "dev"  // local devnet; Keys are deterministic and many security controls are disabled
	TestNet        Environment = "test" // public testnet (needs to be reliable, but run with less Guardians and faster finality)
	GoTest         Environment = "unit-test"
	AccountantMock Environment = "accountant-mock" // Used for mocking accountant with a Wormchain connection
)
