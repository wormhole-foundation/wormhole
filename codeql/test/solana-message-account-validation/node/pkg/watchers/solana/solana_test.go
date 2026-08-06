package solana

func testUnsafe(raw []byte) {
	ParseMessagePublicationAccount(MessageAccountData{Data: raw})
}
