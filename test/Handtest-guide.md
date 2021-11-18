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




The nineteen sample guardian keys:
0x52A26Ce40F8CAa8D36155d37ef0D5D783fc614d2 K: 563d8d2fd4e701901d3846dee7ae7a92c18f1975195264d676f8407ac5976757
0x389A74E8FFa224aeAD0778c786163a7A2150768C K: 8d97f25916a755df1d9ef74eb4dbebc5f868cb07830527731e94478cdc2b9d5f
0xB4459EA6482D4aE574305B239B4f2264239e7599 K: 9bd728ad7617c05c31382053b57658d4a8125684c0098f740a054d87ddc0e93b

0x072491bd66F63356090C11Aae8114F5372aBf12B K: 5a02c4cd110d20a83a7ce8d1a2b2ae5df252b4e5f6781c7855db5cc28ed2d1b4
0x51280eA1fd2B0A1c76Ae29a7d54dda68860A2bfF 93d4e3b443bf11f99a00901222c032bd5f63cf73fc1bcfa40829824d121be9b2
0xfa9Aa60CfF05e20E2CcAA784eE89A0A16C2057CB ea40e40c63c6ff155230da64a2c44fcd1f1c9e50cacb752c230f77771ce1d856

0xe42d59F8FCd86a1c5c4bA351bD251A5c5B05DF6A 87eaabe9c27a82198e618bca20f48f9679c0f239948dbd094005e262da33fe6a
0x4B07fF9D5cE1A6ed58b6e9e7d6974d1baBEc087e 61ffed2bff38648a6d36d6ed560b741b1ca53d45391441124f27e1e48ca04770
0xc8306B84235D7b0478c61783C50F990bfC44cFc0 bd12a242c6da318fef8f98002efb98efbf434218a78730a197d981bebaee826e

0xC8C1035110a13fe788259A4148F871b52bAbcb1B 20d3597bb16525b6d09e5fb56feb91b053d961ab156f4807e37d980f50e71aff
0x58A2508A20A7198E131503ce26bBE119aA8c62b2 344b313ffbc0199ff6ca08cacdaf5dc1d85221e2f2dc156a84245bd49b981673
0x8390820f04ddA22AFe03be1c3bb10f4ba6CF94A0 848b93264edd3f1a521274ca4da4632989eb5303fd15b14e5ec6bcaa91172b05

0x1FD6e97387C34a1F36DE0f8341E9D409E06ec45b c6f2046c1e6c172497fc23bd362104e2f4460d0f61984938fa16ef43f27d93f6
0x255a41fC2792209CB998A8287204D40996df9E54 693b256b1ee6b6fb353ba23274280e7166ab3be8c23c203cc76d716ba4bc32bf
0xbA663B12DD23fbF4FbAC618Be140727986B3BBd0 13c41508c0da03018d61427910b9922345ced25e2bbce50652e939ee6e5ea56d

0x79040E577aC50486d0F6930e160A5C75FD1203C6 460ee0ee403be7a4f1eb1c63dd1edaa815fbaa6cf0cf2344dcba4a8acf9aca74
0x3580D2F00309A9A85efFAf02564Fc183C0183A96 b25148579b99b18c8994b0b86e4dd586975a78fa6e7ad6ec89478d7fbafd2683
0x3869795913D3B6dBF3B24a1C7654672c69A23c35 90d7ac6a82166c908b8cf1b352f3c9340a8d1f2907d7146fb7cd6354a5436cca

0x1c0Cc52D7673c52DE99785741344662F5b2308a0 b71d23908e4cf5d6cd973394f3a4b6b164eb1065785feee612efdfd8d30005ed



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
> t.createSignedVAA(0, sigkeys, 1, 1, 1, '0x71f8dcb863d176e2c420ad6610cf687359612b6fb392e0642b0ca6b1f186aa3b', 0, 0, '0x12345678')
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
71f8dcb863d176e2c420ad6610cf687359612b6fb392e0642b0ca6b1f186aa3b		emitterAddress
0000000000000000	sequence
00					consistency-level

payload: 
12345678

```

VAA Body:
00 00 00 01 00 00 00 01 00 01 71f8dcb863d176e2c420ad6610cf687359612b6fb392e0642b0ca6b1f186aa3b00000000000000010012345678

Since there are three signers, VAA can be verified in a group transaction with just one step.

Stateless logic accepts signature subset as argument (encoded in base64).



```
export SIGNATURES64=`node -e "console.log(Buffer.from('25a2cec435380f6413e8b5d5531cd8789322a1d8bc488309bb868c33a26cc9492947b48895460d8d2261d669bcef210987cc5eeb9fa21504c3f5a9b5a0ff32df00ee4d1a5e589b5aa0d4787eaf57ba4b41e6a54e35e8ca60a028a0f1e35db3a8ed5d901a9831272fc117f472fcd0115d31365efd575a19a28eaf5ad9be7cf5f0d901a671b7c2af66aa6bff3337adaa7e5f196630508f85491e650cfdaccd2f67d2a605faf1c267eddaa50ac6de8d35894afce7974f14982f6173b0d020e0567f2a4a01','hex').toString('base64'))"`
export VAABODY=`0000000100000001000171f8dcb863d176e2c420ad6610cf687359612b6fb392e0642b0ca6b1f186aa3b00000000000000010012345678`
export VAABODY64=`node -e "console.log(Buffer.from('$VAABODY64',  'hex').toString('base64'))"`
export STATELESS_ADDR=4H2VD6GY4L7HEOVTZBKGTO6EYFASYFOWN34ONR5HIFEE2JIJ2M5GK26SXI
export GUARDIAN_KEYS='52A26Ce40F8CAa8D36155d37ef0D5D783fc614d2389A74E8FFa224aeAD0778c786163a7A2150768CB4459EA6482D4aE574305B239B4f2264239e7599'
export GKEYSBASE64=`node -e "console.log(Buffer.from('$GUARDIAN_KEYS',  'hex').toString('base64'))"`

goal app call --app-id 45504480 --from "$STATELESS_ADDR" --app-arg "str:verify" --app-arg "b64:$GKEYSBASE64" --app-arg "int:3" --noteb64 "$VAABODY" -o verify.txn 
 
```

