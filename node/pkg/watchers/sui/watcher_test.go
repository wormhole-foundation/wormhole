package sui

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func Test_fixSuiWsURL(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		expected   string
		logMessage string
		errMessage string
	}{
		{
			name:     "valid",
			value:    "ws://1.2.3.4:5678",
			expected: "ws://1.2.3.4:5678",
		},
		{
			name:       "tilt",
			value:      "sui:9000",
			expected:   "ws://sui:9000",
			logMessage: "DEPRECATED: Prefix --suiWS address with the url scheme e.g.: ws://sui:9000 or wss://sui:9000",
		},
		{
			name:     "valid-no-port",
			value:    "ws://1.2.3.4",
			expected: "ws://1.2.3.4",
		},
		{
			name:       "no-scheme",
			value:      "1.2.3.4:5678",
			expected:   "ws://1.2.3.4:5678",
			logMessage: "DEPRECATED: Prefix --suiWS address with the url scheme e.g.: ws://1.2.3.4:5678 or wss://1.2.3.4:5678",
		},
		{
			name:       "no-scheme-no-port",
			value:      "1.2.3.4",
			expected:   "ws://1.2.3.4",
			logMessage: "DEPRECATED: Prefix --suiWS address with the url scheme e.g.: ws://1.2.3.4 or wss://1.2.3.4",
		},
		{
			name:       "wrong-scheme",
			value:      "http://1.2.3.4",
			errMessage: "invalid url scheme specified for --suiWS, try ws:// or wss://: http://1.2.3.4",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testCore, logs := observer.New(zap.InfoLevel)
			testLogger := zap.New(testCore)

			suiWatcher := Watcher{
				suiWS: testCase.value,
			}
			err := suiWatcher.fixSuiWsURL(testLogger)
			if testCase.errMessage != "" {
				require.EqualError(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
				// Only verify the value if no error was returned
				assert.Equal(t, testCase.expected, suiWatcher.suiWS)
			}

			if len(testCase.logMessage) != 0 {
				// If the testcase expects a log, then there should only be 1 log
				require.Equal(t, 1, logs.Len())

				// Ensure the log message is correct
				actualLogMessage := logs.All()[0].Message
				require.Equal(t, testCase.logMessage, actualLogMessage)
			} else {
				// If the testcase does not expect a log, none should be emitted
				require.Equal(t, 0, logs.Len())
			}
		})
	}
}

func Test_JSONParseOneWHMSg(t *testing.T) {
	// JSON with only the first result (contains all of the fields in `FieldsData` - parses successfully)
	msg := []byte("{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":{\"txDigest\":\"2Z4A1ND5JL8c5ma9WMzFXUvpVqnwoQdYuaX4RwnLyMXU\",\"eventSeq\":\"0\"},\"packageId\":\"0x826915f8ca6d11597dfe6599b8aa02a4c08bd8d39674855254a06ee83fe7220e\",\"transactionModule\":\"lending_portal_v2\",\"sender\":\"0xccce7bbffaf1b9e9e8ca88a68a08fec11f568a697023f475f99efb7bcee951cf\",\"type\":\"0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage\",\"parsedJson\":{\"consistency_level\":0,\"nonce\":0,\"payload\":[0,1,0,34,0,0,204,206,123,191,250,241,185,233,232,202,136,166,138,8,254,193,31,86,138,105,112,35,244,117,249,158,251,123,206,233,81,207,2,0,133,0,0,0,0,0,0,0,0,10,202,0,0,0,10,122,53,130,0,0,76,0,0,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,50,58,58,115,117,105,58,58,83,85,73,0,34,0,0,204,206,123,191,250,241,185,233,232,202,136,166,138,8,254,193,31,86,138,105,112,35,244,117,249,158,251,123,206,233,81,207,2],\"sender\":\"0xdd1ca0bd0b9e449ff55259e5bcf7e0fc1b8b7ab49aabad218681ccce7b202bd6\",\"sequence\":\"2768\",\"timestamp\":\"1693091880\"},\"bcs\":\"J8cfJrtMWT2kg6uBgWQmd8T9k9cSibQg65ufpgxugVM2ghgC8bb1vvqoXmETiMvfb9DJLEDKy2pnvAYivyWJfz8zKSn5u7EfDbMntpszG7D4gsNNu9cU2rMUi4aF7DXnv6QAp5hoaHvJymehRwXkncHfjZ7zKsZ8cUtSKJh6S6YjHMRZ67s1PPwGEVwUdQt5S3WhQdag3tuySe8FDrUWgJfbBawyUKLdbNcR1aXFtBiPu6jQ51BF7sv13x9hp2nbs5EUMYjnN1ykK4YQaKx55eY7TQcxVCRzPrEARSkMjB8VgqefLNpwiCRdq\"}],\"id\":1}")
	expectedPayload := []byte{0, 1, 0, 34, 0, 0, 204, 206, 123, 191, 250, 241, 185, 233, 232, 202, 136, 166, 138, 8, 254, 193, 31, 86, 138, 105, 112, 35, 244, 117, 249, 158, 251, 123, 206, 233, 81, 207, 2, 0, 133, 0, 0, 0, 0, 0, 0, 0, 0, 10, 202, 0, 0, 0, 10, 122, 53, 130, 0, 0, 76, 0, 0, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 50, 58, 58, 115, 117, 105, 58, 58, 83, 85, 73, 0, 34, 0, 0, 204, 206, 123, 191, 250, 241, 185, 233, 232, 202, 136, 166, 138, 8, 254, 193, 31, 86, 138, 105, 112, 35, 244, 117, 249, 158, 251, 123, 206, 233, 81, 207, 2}

	var res SuiTxnQuery
	err := json.Unmarshal(msg, &res)
	require.NoError(t, err)
	for _, chunk := range res.Result {
		// chunk is a SuiResult
		fmt.Println("body.Type", *chunk.Type)
		assert.Equal(t, "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage", *chunk.Type)
		var fields FieldsData
		err := json.Unmarshal(*chunk.Fields, &fields)
		require.NoError(t, err)

		assert.Equal(t, uint8(0), *fields.ConsistencyLevel)
		assert.Equal(t, uint64(0), *fields.Nonce)
		assert.Equal(t, expectedPayload, fields.Payload)
		assert.Equal(t, "0xdd1ca0bd0b9e449ff55259e5bcf7e0fc1b8b7ab49aabad218681ccce7b202bd6", *fields.Sender)
		assert.Equal(t, "2768", *fields.Sequence)
		assert.Equal(t, "1693091880", *fields.Timestamp)
	}
}

func Test_JSONParseMultipleMsgs(t *testing.T) {
	// Original JSON (fails to parse)
	msg := []byte("{\"jsonrpc\":\"2.0\",\"result\":[{\"id\":{\"txDigest\":\"2Z4A1ND5JL8c5ma9WMzFXUvpVqnwoQdYuaX4RwnLyMXU\",\"eventSeq\":\"0\"},\"packageId\":\"0x826915f8ca6d11597dfe6599b8aa02a4c08bd8d39674855254a06ee83fe7220e\",\"transactionModule\":\"lending_portal_v2\",\"sender\":\"0xccce7bbffaf1b9e9e8ca88a68a08fec11f568a697023f475f99efb7bcee951cf\",\"type\":\"0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage\",\"parsedJson\":{\"consistency_level\":0,\"nonce\":0,\"payload\":[0,1,0,34,0,0,204,206,123,191,250,241,185,233,232,202,136,166,138,8,254,193,31,86,138,105,112,35,244,117,249,158,251,123,206,233,81,207,2,0,133,0,0,0,0,0,0,0,0,10,202,0,0,0,10,122,53,130,0,0,76,0,0,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,50,58,58,115,117,105,58,58,83,85,73,0,34,0,0,204,206,123,191,250,241,185,233,232,202,136,166,138,8,254,193,31,86,138,105,112,35,244,117,249,158,251,123,206,233,81,207,2],\"sender\":\"0xdd1ca0bd0b9e449ff55259e5bcf7e0fc1b8b7ab49aabad218681ccce7b202bd6\",\"sequence\":\"2768\",\"timestamp\":\"1693091880\"},\"bcs\":\"J8cfJrtMWT2kg6uBgWQmd8T9k9cSibQg65ufpgxugVM2ghgC8bb1vvqoXmETiMvfb9DJLEDKy2pnvAYivyWJfz8zKSn5u7EfDbMntpszG7D4gsNNu9cU2rMUi4aF7DXnv6QAp5hoaHvJymehRwXkncHfjZ7zKsZ8cUtSKJh6S6YjHMRZ67s1PPwGEVwUdQt5S3WhQdag3tuySe8FDrUWgJfbBawyUKLdbNcR1aXFtBiPu6jQ51BF7sv13x9hp2nbs5EUMYjnN1ykK4YQaKx55eY7TQcxVCRzPrEARSkMjB8VgqefLNpwiCRdq\"},{\"id\":{\"txDigest\":\"2Z4A1ND5JL8c5ma9WMzFXUvpVqnwoQdYuaX4RwnLyMXU\",\"eventSeq\":\"1\"},\"packageId\":\"0x826915f8ca6d11597dfe6599b8aa02a4c08bd8d39674855254a06ee83fe7220e\",\"transactionModule\":\"lending_portal_v2\",\"sender\":\"0xccce7bbffaf1b9e9e8ca88a68a08fec11f568a697023f475f99efb7bcee951cf\",\"type\":\"0x826915f8ca6d11597dfe6599b8aa02a4c08bd8d39674855254a06ee83fe7220e::wormhole_adapter_pool::RelayEvent\",\"parsedJson\":{\"app_id\":1,\"call_type\":2,\"fee_amount\":\"57821069\",\"nonce\":\"2762\",\"sequence\":\"2768\"},\"bcs\":\"V7pAXEvqBvtV5ps2fetket9wYhsoQT1Y8X6bT\"},{\"id\":{\"txDigest\":\"2Z4A1ND5JL8c5ma9WMzFXUvpVqnwoQdYuaX4RwnLyMXU\",\"eventSeq\":\"2\"},\"packageId\":\"0x826915f8ca6d11597dfe6599b8aa02a4c08bd8d39674855254a06ee83fe7220e\",\"transactionModule\":\"lending_portal_v2\",\"sender\":\"0xccce7bbffaf1b9e9e8ca88a68a08fec11f568a697023f475f99efb7bcee951cf\",\"type\":\"0x826915f8ca6d11597dfe6599b8aa02a4c08bd8d39674855254a06ee83fe7220e::lending_portal_v2::LendingPortalEvent\",\"parsedJson\":{\"amount\":\"45000000000\",\"call_type\":2,\"dola_pool_address\":[48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,48,50,58,58,115,117,105,58,58,83,85,73],\"dst_chain_id\":0,\"nonce\":\"2762\",\"receiver\":[204,206,123,191,250,241,185,233,232,202,136,166,138,8,254,193,31,86,138,105,112,35,244,117,249,158,251,123,206,233,81,207],\"sender\":\"0xccce7bbffaf1b9e9e8ca88a68a08fec11f568a697023f475f99efb7bcee951cf\",\"source_chain_id\":0},\"bcs\":\"U7Gg8eey15TjPBGfZcKPCnYHJ84s3pL2BYUaw4r3NjK8AcWfzgKqsgW9F27yhPBtQdytETbAVqfx6b2Xsw7Ypprnbym5UEzyLHzuS79PMaAbGrXtVmdDYeHnoQ3DjfCSVZ5fZEaENLmmhe5m4iEYdkrjjaujoVQtuoFqjzaXYbMj89oksCE3E19PWsKzP7DVDcC99JjphepJgtGjQhCvdtzLd8kR\"}],\"id\":1}")
	expectedPayload := []byte{0, 1, 0, 34, 0, 0, 204, 206, 123, 191, 250, 241, 185, 233, 232, 202, 136, 166, 138, 8, 254, 193, 31, 86, 138, 105, 112, 35, 244, 117, 249, 158, 251, 123, 206, 233, 81, 207, 2, 0, 133, 0, 0, 0, 0, 0, 0, 0, 0, 10, 202, 0, 0, 0, 10, 122, 53, 130, 0, 0, 76, 0, 0, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 50, 58, 58, 115, 117, 105, 58, 58, 83, 85, 73, 0, 34, 0, 0, 204, 206, 123, 191, 250, 241, 185, 233, 232, 202, 136, 166, 138, 8, 254, 193, 31, 86, 138, 105, 112, 35, 244, 117, 249, 158, 251, 123, 206, 233, 81, 207, 2}

	var res SuiTxnQuery
	err := json.Unmarshal(msg, &res)
	require.NoError(t, err)

	for _, chunk := range res.Result {
		// chunk is a SuiResult
		if "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage" != *chunk.Type {
			continue
		}
		var fields FieldsData
		err := json.Unmarshal(*chunk.Fields, &fields)
		require.NoError(t, err)

		assert.Equal(t, uint8(0), *fields.ConsistencyLevel)
		assert.Equal(t, uint64(0), *fields.Nonce)
		assert.Equal(t, expectedPayload, fields.Payload)
		assert.Equal(t, "0xdd1ca0bd0b9e449ff55259e5bcf7e0fc1b8b7ab49aabad218681ccce7b202bd6", *fields.Sender)
		assert.Equal(t, "2768", *fields.Sequence)
		assert.Equal(t, "1693091880", *fields.Timestamp)
	}
}
