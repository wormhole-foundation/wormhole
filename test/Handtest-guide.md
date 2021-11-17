Hand-testing guide
------------------

Compile PyTEAL sources,

```
C:\src\PriceCasterService>python teal\wormhole\pyteal\vaa-processor.py vaap.teal vaap-clear.teal
VAA Processor Program, (c) 2021-22 Randlabs Inc.
Compiling approval program...
Written to vaap.teal
Compiling clear state program...
Written to vaap-clear.teal

```

Copy compiled teal files to Sandbox (optional):

```
./sandbox copyTo /mnt/c/src/PriceCasterService/vaap.teal

Now copying /mnt/c/src/PriceCasterService/vaap.teal to Algod container in /opt/data//mnt/c/src/PriceCasterService/vaap.teal
./sandbox copyTo /mnt/c/src/PriceCasterService/vaap-clear.teal

Now copying /mnt/c/src/PriceCasterService/vaap-clear.teal to Algod container in /opt/data//mnt/c/src/PriceCasterService/vaap-clear.teal
```




Three sample guardian keys:
0x52A26Ce40F8CAa8D36155d37ef0D5D783fc614d2
K: 563d8d2fd4e701901d3846dee7ae7a92c18f1975195264d676f8407ac5976757

0x389A74E8FFa224aeAD0778c786163a7A2150768C
K: 8d97f25916a755df1d9ef74eb4dbebc5f868cb07830527731e94478cdc2b9d5f

0xB4459EA6482D4aE574305B239B4f2264239e7599
K: 9bd728ad7617c05c31382053b57658d4a8125684c0098f740a054d87ddc0e93b


Set environment variables for convenience.

```

export OWNER_ADDR=OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU
export OWNER_MNEMO='assault approve result rare float sugar power float soul kind galaxy edit unusual pretty tone tilt net range pelican avoid unhappy amused recycle abstract master'
export GUARDIAN_KEYS='52A26Ce40F8CAa8D36155d37ef0D5D783fc614d2389A74E8FFa224aeAD0778c786163a7A2150768CB4459EA6482D4aE574305B239B4f2264239e7599'
export GKEYSBASE64=`node -e "console.log(Buffer.from('$GUARDIAN_KEYS',  'hex').toString('base64'))"`

```

* Make sure to clear previously created VAA Processor Apps, example:

```
goal account  info -a OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU
Created Assets:
        <none>
Held Assets:
        ID 14704676, Wrapped Algo Testnet, balance 0.000000 wALGO Ts
Created Apps:
        ID 45231352, global state used 2/2 uints, 20/20 byte slices
Opted In Apps:
        ID 14713804, local state used 1/3 uints, 1/2 byte slices
		
		
goal app delete --app-id 45231352 --from OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU -o delete.txn
algokey -t delete.txn -o delete.stxn sign  -m $OWNER_MNEMO
goal clerk rawsend -f delete.sxtn
```
	

* Deploy and bootstrap the VAA Processor contract.

```
 goal app create --creator "$OWNER_ADDR" --global-ints 4 --global-byteslices 20 --local-byteslices 0 --local-ints 0 --approval-prog vaap.teal --clear-prog vaap-clear.teal  --app-arg "b64:$GKEYSBASE64" --app-arg 'int:0' --app-arg 'int:0' -o create.txn
 algokey -t create.txn -o create.stxn sign  -m "$OWNER_MNEMO" && goal clerk rawsend -f create.stxn

```


Check the deployed application ID according to the final TX ID.

* Compile the stateless logic with the VAA Processor ID, for example: 


```
python teal\wormhole\pyteal\vaa-verify.py 45504480 vaaverify.teal

goal clerk compile vaaverify.teal
vaaverify.teal: 4H2VD6GY4L7HEOVTZBKGTO6EYFASYFOWN34ONR5HIFEE2JIJ2M5GK26SXI
```

* Set the stateless program hash with the "setvphash" appcall:

```
goal app call --app-id 45504480 --from "$OWNER_ADDR" --app-arg "str:setvphash" --app-arg "addr:4H2VD6GY4L7HEOVTZBKGTO6EYFASYFOWN34ONR5HIFEE2JIJ2M5GK26SXI" -o setvphash.txn 
algokey -t setvphash.txn -o setvphash.stxn sign  -m "$OWNER_MNEMO" && goal clerk rawsend -f setvphash.stxn
```

* To verify a sample VAA, you can use the testing library using Nodejs as follows:

```
Welcome to Node.js v14.17.6.
Type ".help" for more information.
> TestLib = require('./test/testlib')
{ TestLib: [class TestLib] }
> t = new TestLib.TestLib
TestLib {}
> sigkeys = ['563d8d2fd4e701901d3846dee7ae7a92c18f1975195264d676f8407ac5976757', '8d97f25916a755df1d9ef74eb4dbebc5f868cb07830527731e94478cdc2b9d5f', '9bd728ad7617c05c31382053b57658d4a8125684c0098f740a054d87ddc0e93b']
[
  '563d8d2fd4e701901d3846dee7ae7a92c18f1975195264d676f8407ac5976757',
  '8d97f25916a755df1d9ef74eb4dbebc5f868cb07830527731e94478cdc2b9d5f',
  '9bd728ad7617c05c31382053b57658d4a8125684c0098f740a054d87ddc0e93b'
]
> t.createSignedVAA(0, sigkeys, 1, 1, 1, '0x000000000000000000000000000000000000000000000000000000000000ffff', 0, 0, '0x12345678')
'0100000000030025a2cec435380f6413e8b5d5531cd8789322a1d8bc488309bb868c33a26cc9492947b48895460d8d2261d669bcef210987cc5eeb9fa21504c3f5a9b5a0ff32df0001ee4d1a5e589b5aa0d4787eaf57ba4b41e6a54e35e8ca60a028a0f1e35db3a8ed5d901a9831272fc117f472fcd0115d31365efd575a19a28eaf5ad9be7cf5f0d90102a671b7c2af66aa6bff3337adaa7e5f196630508f85491e650cfdaccd2f67d2a605faf1c267eddaa50ac6de8d35894afce7974f14982f6173b0d020e0567f2a4a0100000001000000010001000000000000000000000000000000000000000000000000000000000000ffff00000000000000000012345678'
```


The resulting decomposed in fields VAA is:

```

01			Version
00000000    Guardian-set-index
03			Signature count

sig indexes + signatures:
00			
25a2cec435380f6413e8b5d5531cd8789322a1d8bc488309bb868c33a26cc9492947b48895460d8d2261d669bcef210987cc5eeb9fa21504c3f5a9b5a0ff32df00
01
ee4d1a5e589b5aa0d4787eaf57ba4b41e6a54e35e8ca60a028a0f1e35db3a8ed5d901a9831272fc117f472fcd0115d31365efd575a19a28eaf5ad9be7cf5f0d901
02
a671b7c2af66aa6bff3337adaa7e5f196630508f85491e650cfdaccd2f67d2a605faf1c267eddaa50ac6de8d35894afce7974f14982f6173b0d020e0567f2a4a01

00000001    timestamp 
00000001    nonce 
0001		chain-id
000000000000000000000000000000000000000000000000000000000000ffff		emitterAddress
0000000000000000	sequence
00					consistency-level

payload: 
12345678

```

