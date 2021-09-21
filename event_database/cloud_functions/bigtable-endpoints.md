
# API design
There are two endpoints designed to be flexible enough to answer most questions; "recent" and "totals".

- "recent" returns rows, is a gap-list query
- "totals" returns counts of how many rows were found in the period

---
## QueryParams
These endpoints can be used to query across all chains and addresses, and you can also drill-down into a chain or address.

### groupBy
- `groupBy=chain` results will be grouped by (keyed by) `emitterChain`.
- `groupBy=address` results will be be grouped by (keyed by) `emitterChain:emitterAddress`.

### filter
- `forChain=2` only returns results for the specified chain.
- `forChain=2&forAddress=c69a...cb4f` only returns results for the specified chain + address.

### endpoint specific
- `/totals?numDays=6` specify the query interval.
- `/recent?numRows=6` specify the number of results.

---
## `Totals` function

Get the number of messages in the last 7 days. The `*` key designates all results.

	https://us-east4-wormhole-315720.cloudfunctions.net/devnet/totals?numDays=7

```json
{
	"LastDayCount": { "*": 14},
	"PeriodCount": { "*": 69},
	"DailyTotals": {
		"2021-09-21": {"*": 55},
		"2021-09-22": {"*": 0},
		"2021-09-23": {"*": 0},
		"2021-09-24": {"*": 0},
		"2021-09-25": {"*": 0},
		"2021-09-26": {"*": 0},
		"2021-09-27": {"*": 14},
		"2021-09-28": {"*": 0},
	}
}
```


Get message counts grouped by chain, for the last 7 days:

	https://us-east4-wormhole-315720.cloudfunctions.net/devnet/totals?groupBy=chain&numDays=7

```json
{
    "LastDayCount": {
        "1": 8,
        "2": 3,
        "4": 3,
        "*": 14
    },
    "LastMonthCount": {
        "1": 21,
        "2": 24,
        "4": 24,
        "*": 69
    },
    "DailyTotals": {
        "2021-09-21": {
            "1": 13,
            "2": 21,
            "4": 21,
            "*": 55
        },
        "2021-09-22": {
            "1": 0,
            "2": 0,
            "4": 0,
            "*": 0
        },
        "2021-09-23": {
            "1": 0,
            "2": 0,
            "4": 0,
            "*": 0
        },
        "2021-09-24": {
            "1": 0,
            "2": 0,
            "4": 0,
            "*": 0
        },
        "2021-09-25": {
            "1": 0,
            "2": 0,
            "4": 0,
            "*": 0
        },
        "2021-09-26": {
            "1": 0,
            "2": 0,
            "4": 0,
            "*": 0
        },
        "2021-09-27": {
            "1": 8,
            "2": 3,
            "4": 3,
            "*": 14
        },
        "2021-09-28": {
            "1": 0,
            "2": 0,
            "4": 0,
            "*": 0
        }
    }
}
```


Get message counts grouped by EmitterAddress, for the previous 3 days (includes the current day):

	https://us-east4-wormhole-315720.cloudfunctions.net/devnet/totals?groupBy=address&numDays=3

```json
{
    "LastDayCount": {
        "*": 14,
        "1:96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab": 1,
        "1:c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f": 7,
        "2:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 3,
        "4:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 3
    },
    "TotalCount": {
        "*": 14,
        "1:96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab": 1,
        "1:c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f": 7,
        "2:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 3,
        "4:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 3
    },
    "DailyTotals": {
        "2021-09-25": {
            "*": 0,
            "1:96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab": 0,
            "1:c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f": 0,
            "2:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 0,
            "4:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 0
        },
        "2021-09-26": {
            "*": 0,
            "1:96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab": 0,
            "1:c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f": 0,
            "2:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 0,
            "4:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 0
        },
        "2021-09-27": {
            "*": 14,
            "1:96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab": 1,
            "1:c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f": 7,
            "2:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 3,
            "4:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 3
        },
        "2021-09-28": {
            "*": 0,
            "1:96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab": 0,
            "1:c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f": 0,
            "2:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 0,
            "4:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": 0
        }
    }
}
```

---
## `Recent` function

Get the 2 most recent messages:

	https://us-east4-wormhole-315720.cloudfunctions.net/devnet/recent?numRows=2


```json
{
	"*": [
		{
			"EmitterChain": "solana",
			"EmitterAddress": "c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
			"Sequence": "17",
			"InitiatingTxID": "0xd418d81b7b2f298a37b28b97e240237b6210f00b702d2101d5e423ab5fa6366b",
			"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAgAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"GuardiansThatSigned": [
				"0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
			],
			"SignedVAABytes": "AQAAAAABADjricLUCKqwbuHYEgG8dMetrH5acGibV/l4z6mNzYmyXlE0sPK4lVngQ5c+vwWU0XYVlrh1KoCsEhZF132ouo8BYUk6ywAA1PUAAcaaGxpl3TNr8d9qd6+1Afwl23/Ak4ywhZWp70cyZctPAAAAAAAAABEgAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAgAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"QuorumTime": "2021-09-21 01:52:26.038 +0000 UTC"
		},
		{
			"EmitterChain": "solana",
			"EmitterAddress": "c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
			"Sequence": "16",
			"InitiatingTxID": "0xd2bcadceb8c1beb7cd531e2c621733b96df96a397ea88abb948cc28c1546e139",
			"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAgAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"GuardiansThatSigned": [
				"0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
			],
			"SignedVAABytes": "AQAAAAABACISbeEGlIf5z32yTEQDw2zNgS4GUj36YSTlSCqTj4lgaH663yeir/4Gi9iM6OWWc4Vct2UiE5jfv4PW8MTrdr0BYUk6sAAABBMAAcaaGxpl3TNr8d9qd6+1Afwl23/Ak4ywhZWp70cyZctPAAAAAAAAABAgAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAgAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"QuorumTime": "2021-09-21 01:51:59.138 +0000 UTC"
		}
	]
}
```

Get the 2 most recent messages for each chain:

	https://us-east4-wormhole-315720.cloudfunctions.net/devnet/recent?numRows=2&groupBy=chain

```json
{
	"1": [
		{
			"EmitterChain": "solana",
			"EmitterAddress": "c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
			"Sequence": "19",
			"InitiatingTxID": "0xd7a34663ce6ee1d1c42f24513f6f37221e81e16a5153d542d2c951af1401e49d",
			"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAgAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"GuardiansThatSigned": [
				"0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
			],
			"SignedVAABytes": "AQAAAAABAOcc6ah0v1QFBl8SOkzKzAme6I2Us/kGwM1QCumJNqOnGmsH82w0k+1kgxu6yHA1XKRNUbJFgz/RfHrgfXUXKeEBYUk7PwAAph4AAcaaGxpl3TNr8d9qd6+1Afwl23/Ak4ywhZWp70cyZctPAAAAAAAAABMgAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAgAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"QuorumTime": "2021-09-21 01:54:22.107 +0000 UTC"
		},
		{
			"EmitterChain": "solana",
			"EmitterAddress": "c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
			"Sequence": "18",
			"InitiatingTxID": "0x32e8a87d4cd8a717e4d785bb317398c4cc8e36fbe45c53b75e4e85dc1181c92b",
			"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAgAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"GuardiansThatSigned": [
				"0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
			],
			"SignedVAABytes": "AQAAAAABAMCe6wEJplDwtyr7ELM15nrSSMSr6xYcuDC3qA0Mx1WKdy7WRXE13tP9SyMJ/sYESqpJtgvYnNEB3wnUeEbW2scAYUk6+AAAGp4AAcaaGxpl3TNr8d9qd6+1Afwl23/Ak4ywhZWp70cyZctPAAAAAAAAABIgAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAgAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"QuorumTime": "2021-09-21 01:53:11.139 +0000 UTC"
		}
	],
	"2": [
		{
			"EmitterChain": "ethereum",
			"EmitterAddress": "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
			"Sequence": "23",
			"InitiatingTxID": "0x0515a7375f101e79a1d5e0f5159cce98fe8fe861bd2ab548e22f43375b04defb",
			"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"GuardiansThatSigned": [
				"0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
			],
			"SignedVAABytes": "AQAAAAABAGclDJrZDoZ2BxHBCxpPHZFwRhwesOgV9gkcGCeqBQaTZj/PjYM/25a5owDllBvS2pAg0nkRWYJskJf+Z3vIqLcAAAAW9pRWAAAAAgAAAAAAAAAAAAAAAAKQ+xZyCK9FW7E3eAFjt7epoQwWAAAAAAAAABcPAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"QuorumTime": "2021-09-21 01:48:27.025 +0000 UTC"
		},
		{
			"EmitterChain": "ethereum",
			"EmitterAddress": "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
			"Sequence": "22",
			"InitiatingTxID": "0x9f2dbf04c8088009b8c0ae1313baee546ac604ad5f608dcf5291bee4aa19b57b",
			"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"GuardiansThatSigned": [
				"0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
			],
			"SignedVAABytes": "AQAAAAABAAPsvYSDgik3jFPBiH97URck6lQxeXKixD/U3YplSwx4EZPeVWLzqgzjCb5nhBhAafYY5MmVSf8YF1cnPW4qXO0BAAAW0sNgAQAAAgAAAAAAAAAAAAAAAAKQ+xZyCK9FW7E3eAFjt7epoQwWAAAAAAAAABYPAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"QuorumTime": "2021-09-21 01:47:51.506 +0000 UTC"
		}
	],
	"4": [
		{
			"EmitterChain": "bsc",
			"EmitterAddress": "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
			"Sequence": "23",
			"InitiatingTxID": "0x0515a7375f101e79a1d5e0f5159cce98fe8fe861bd2ab548e22f43375b04defb",
			"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"GuardiansThatSigned": [
				"0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
			],
			"SignedVAABytes": "AQAAAAABAEc9grHDBKGhicCbWPFFuEKxfEuWc+PS0C3smLeIrBkVCdm9Tg8q76MK47OeuTF+ieTAxG+d/z2B9OeMWd87oMsAAAAW9pRWAAAABAAAAAAAAAAAAAAAAAKQ+xZyCK9FW7E3eAFjt7epoQwWAAAAAAAAABcPAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"QuorumTime": "2021-09-21 01:48:26.983 +0000 UTC"
		},
		{
			"EmitterChain": "bsc",
			"EmitterAddress": "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
			"Sequence": "22",
			"InitiatingTxID": "0x9f2dbf04c8088009b8c0ae1313baee546ac604ad5f608dcf5291bee4aa19b57b",
			"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"GuardiansThatSigned": [
				"0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
			],
			"SignedVAABytes": "AQAAAAABABSFvsV41QWUwqKJC+Q62PtxHWmludvu4AKQDxorezX4BzYhX0rkj9BDxPtEc+utn6Y5q/ryft+PdWX8WIDhxSMAAAAW0sNgAQAABAAAAAAAAAAAAAAAAAKQ+xZyCK9FW7E3eAFjt7epoQwWAAAAAAAAABYPAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"QuorumTime": "2021-09-21 01:47:51.419 +0000 UTC"
		}
	]
}
```

Get the 2 most recent messages for a specific address:

	https://us-east4-wormhole-315720.cloudfunctions.net/devnet/recent?numRows=2&forChain=2&forAddress=0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16

```json
{
	"2:0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": [
		{
			"EmitterChain": "ethereum",
			"EmitterAddress": "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
			"Sequence": "23",
			"InitiatingTxID": "0x0515a7375f101e79a1d5e0f5159cce98fe8fe861bd2ab548e22f43375b04defb",
			"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"GuardiansThatSigned": [
				"0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
			],
			"SignedVAABytes": "AQAAAAABAGclDJrZDoZ2BxHBCxpPHZFwRhwesOgV9gkcGCeqBQaTZj/PjYM/25a5owDllBvS2pAg0nkRWYJskJf+Z3vIqLcAAAAW9pRWAAAAAgAAAAAAAAAAAAAAAAKQ+xZyCK9FW7E3eAFjt7epoQwWAAAAAAAAABcPAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"QuorumTime": "2021-09-21 01:48:27.025 +0000 UTC"
		},
		{
			"EmitterChain": "ethereum",
			"EmitterAddress": "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
			"Sequence": "22",
			"InitiatingTxID": "0x9f2dbf04c8088009b8c0ae1313baee546ac604ad5f608dcf5291bee4aa19b57b",
			"Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"GuardiansThatSigned": [
				"0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
			],
			"SignedVAABytes": "AQAAAAABAAPsvYSDgik3jFPBiH97URck6lQxeXKixD/U3YplSwx4EZPeVWLzqgzjCb5nhBhAafYY5MmVSf8YF1cnPW4qXO0BAAAW0sNgAQAAAgAAAAAAAAAAAAAAAAKQ+xZyCK9FW7E3eAFjt7epoQwWAAAAAAAAABYPAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			"QuorumTime": "2021-09-21 01:47:51.506 +0000 UTC"
		}
	]
}
```
---
## `Transaction` function

Lookup a message by the native transaction identifier from the user's interaction:

	https://us-east4-wormhole-315720.cloudfunctions.net/devnet/transaction?id=0x0515a7375f101e79a1d5e0f5159cce98fe8fe861bd2ab548e22f43375b04defb

```json
{
    "EmitterChain": "bsc",
    "EmitterAddress": "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
    "Sequence": "23",
    "InitiatingTxID": "0x0515a7375f101e79a1d5e0f5159cce98fe8fe861bd2ab548e22f43375b04defb",
    "Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
    "GuardiansThatSigned": [
        "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
    ],
    "SignedVAABytes": "AQAAAAABAEc9grHDBKGhicCbWPFFuEKxfEuWc+PS0C3smLeIrBkVCdm9Tg8q76MK47OeuTF+ieTAxG+d/z2B9OeMWd87oMsAAAAW9pRWAAAABAAAAAAAAAAAAAAAAAKQ+xZyCK9FW7E3eAFjt7epoQwWAAAAAAAAABcPAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAlraZ6SC3I261q1BLAdbD9zRURvzAgIW7YAEZEXawNBFAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
    "QuorumTime": "2021-09-21 01:48:26.983 +0000 UTC"
}
```
---
## `ReadRow` function

Lookup a message by the MessageID values:

	https://us-east4-wormhole-315720.cloudfunctions.net/devnet/readrow?emitterChain=1&emitterAddress=96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab&sequence=0

```json
{
    "EmitterChain": "solana",
    "EmitterAddress": "96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab",
    "Sequence": "0",
    "InitiatingTxID": "0xcc3aedef591ff7725b9a1873a006b1431a6cc6e3ae69f03f7692a6053de06b3e",
    "Payload": "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAAFQVU5L8J+OuAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE5vdCBhIFBVTkvwn464AAAAAAAAAAAAAAAAAAAAAAAAnABsSMjL8zhJywej+TYVnMUj+VkcsZmavUWJDsX+6bczaHR0cHM6Ly93cmFwcGVkcHVua3MuY29tOjMwMDAvYXBpL3B1bmtzL21ldGFkYXRhLzM5AAAAAAAAAAAAAAAAkPi/akefMg6tB0QRpLDnlE6oycEAAg==",
    "GuardiansThatSigned": [
        "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
    ],
    "SignedVAABytes": "AQAAAAABAP9HdhYz1TU+XRH7fVlYU9FJH8WVxknCJwDoPHvCM/2FMkRS8vuEIo/yvoW8TLkNJq7ydXhhZNzc/elwsBEEqZkBYVJaqAABTIMAAZbumCKTJRtIcpgEyOiyS1U+triHhnAklI0iNv03pXerAAAAAAAAAAABAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAAFQVU5L8J+OuAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE5vdCBhIFBVTkvwn464AAAAAAAAAAAAAAAAAAAAAAAAnABsSMjL8zhJywej+TYVnMUj+VkcsZmavUWJDsX+6bczaHR0cHM6Ly93cmFwcGVkcHVua3MuY29tOjMwMDAvYXBpL3B1bmtzL21ldGFkYXRhLzM5AAAAAAAAAAAAAAAAkPi/akefMg6tB0QRpLDnlE6oycEAAg==",
    "QuorumTime": "2021-09-27 23:58:33.874 +0000 UTC"
}
```
