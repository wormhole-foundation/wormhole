## Google Cloud function for reading BigTable

This is a reference implementaion for getting data out of BigTable.

### invocation

both methods read the same data. just two different ways of querying:

GET

```bash
curl "https://region-project-id.cloudfunctions.net/your-function-name?emitterChain=2&emitterAddress=000000000000000000000000e982e462b094850f12af94d21d470e21be9d0e9c&sequence=6"
```

POST

```bash
curl -X POST  https://region-project-id.cloudfunctions.net/your-function-name \
-H "Content-Type:application/json" \
-d \
'{"emitterChain":"2", "emitterAddress":"000000000000000000000000e982e462b094850f12af94d21d470e21be9d0e9c", "sequence":"6"}' | jq '.'

{
  "Message": {
    "InitiatingTxID": "0x47727f32a3c6033044fd9f11778c6b5691262533607a654fd020c068e5d12fba",
    "Payload": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAAApD7FnIIr0VbsTd4AWO3t6mhDBYAAjsehQZiDTcv6/TspR/9xdoL+60kUe5xBKmz74vCab+YAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
  },
  "GuardianAddresses": [
    "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
  ],
  "SignedVAA": "AQAAAAABAIOSpeda6nEXWxJoS/d59cniULw0+DDSOVBxxOZPltunSM0BHgoJh6Srbg8Fa4eqLlifpCibLJx9MbJSwbXerZkAAAE4yuBzAQAAAgAAAAAAAAAAAAAAAOmC5GKwlIUPEq+U0h1HDiG+nQ6cAAAAAAAAAAYPAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF9eEAAAAAAAAAAAAAAAAAApD7FnIIr0VbsTd4AWO3t6mhDBYAAjsehQZiDTcv6/TspR/9xdoL+60kUe5xBKmz74vCab+YAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "QuorumTime": "2021-08-11 00:16:11.757 +0000 UTC"
}

```
