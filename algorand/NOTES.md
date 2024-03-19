npm run test -- algorand

https://developer.algorand.org/docs/rest-apis/algod/v2/#get-v2statuswait-for-block-afterround

current algorand machine size:

  https://howbigisalgorand.com/

custom indexes:

  https://github.com/algorand/indexer/blob/develop/docs/PostgresqlIndexes.md

Installing node:

  https://developer.algorand.org/docs/run-a-node/setup/install/


kubectl exec -it algorand-0 -c algorand-algod -- /bin/bash

docker exec -it algorand-tilt-indexer /bin/bash

to switch to sandbox, change devnet/node.yaml

-            - http://algorand:8980
+            - http://host.minikube.internal:8980

put into dev/node.yaml

            - --algorandAppID
            - "1004"

Install the algorand requirements

  python3 -m pip  install  -r requirements.txt

install docker-compile

./sandbox down; ./sandbox clean; ./sandbox up dev -v; python3 admin.py --devnet

bring up the dev sandbox

  ./sandbox down; ./sandbox clean


<!-- cspell:disable -->
[jsiegel@gusc1a-ossdev-jsl1 ~/.../algorand/_sandbox]{master} git diff
diff --git a/images/indexer/start.sh b/images/indexer/start.sh
index 9e224c2..f1714ea 100755
--- a/images/indexer/start.sh
+++ b/images/indexer/start.sh
@@ -28,6 +28,7 @@ start_with_algod() {
   /tmp/algorand-indexer daemon \
     --dev-mode \
+    --enable-all-parameters \
     --server ":$PORT" \
     -P "$CONNECTION_STRING" \
     --algod-net "${ALGOD_ADDR}" \

  ./sandbox up dev


docker_compose("./algorand/sandbox-algorand/tilt-compose.yml")

dc_resource('algo-algod', labels=["algorand"])
dc_resource('algo-indexer', labels=["algorand"])
dc_resource('algo-indexer-db', labels=["algorand"])

    // Solana
    "01000000000100c9f4230109e378f7efc0605fb40f0e1869f2d82fda5b1dfad8a5a2dafee85e033d155c18641165a77a2db6a7afbf2745b458616cb59347e89ae0c7aa3e7cc2d400000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e4272696467650100000001c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
    // Ethereum
    "01000000000100e2e1975d14734206e7a23d90db48a6b5b6696df72675443293c6057dcb936bf224b5df67d32967adeb220d4fe3cb28be515be5608c74aab6adb31099a478db5c01000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e42726964676501000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
    // BSC
        "01000000000100719b4ada436f614489dbf87593c38ba9aea35aa7b997387f8ae09f819806f5654c8d45b6b751faa0e809ccbc294794885efa205bd8a046669464c7cbfb03d183010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000002c8bb0600000000000000000000000000000000000000000000546f6b656e42726964676501000000040000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",

('0100000001010001ca2fbf60ac6227d47dda4fe2e7bccc087f27d22170a212b9800da5b4cbf0d64c52deb2f65ce58be2267bf5b366437c267b5c7b795cd6cea1ac2fee8a1db3ad006225f801000000010001000000000000000000000000000000000000000000000000000000000000000400000000000000012000000000000000000000000000000000000000000000000000000000436f72650200000000000001beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe',
 {'Meta': 'CoreGovernance',
  'NewGuardianSetIndex': 0,
  'action': 2,
  'chain': 1,
  'chainRaw': b'\x00\x01',
  'consistency': 32,
  'digest': b'b%\xf8\x01\x00\x00\x00\x01\x00\x01\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00'
            b'\x00\x01 \x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00C'
            b'ore\x02\x00\x00\x00\x00\x00\x00\x01\xbe\xfaB\x9dW\xcd\x18\xb7\xf8'
            b'\xa4\xd9\x1a-\xa9\xabJ\xf0]\x0f\xbe',
  'emitter': b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
             b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04',
  'index': 1,
  'module': '00000000000000000000000000000000000000000000000000000000436f7265',
  'nonce': 1,
  'sequence': 1,
  'siglen': 1,
  'signatures': b"\x00\x01\xca/\xbf`\xacb'\xd4}\xdaO\xe2\xe7\xbc\xcc\x08\x7f'"
                b'\xd2!p\xa2\x12\xb9\x80\r\xa5\xb4\xcb\xf0\xd6LR\xde'
                b'\xb2\xf6\\\xe5\x8b\xe2&{\xf5\xb3fC|&{\\{y\\\xd6\xce\xa1\xac/'
                b'\xee\x8a\x1d\xb3\xad\x00',
  'sigs': ['0001ca2fbf60ac6227d47dda4fe2e7bccc087f27d22170a212b9800da5b4cbf0d64c52deb2f65ce58be2267bf5b366437c267b5c7b795cd6cea1ac2fee8a1db3ad00'],
  'targetChain': 0,
  'timestamp': 1646655489,
  'version': 1})


Registering chain 1
('01000000020100c2f0b6e546e093630295e5007e8b077b1028d3aa9a72ab4c454b261306eb4f550179638597f25afd6f40a18580bc87fa315552e7294b407bd4616f0995d1cb55016225f5fd0000000300010000000000000000000000000000000000000000000000000000000000000004000000000000000320000000000000000000000000000000000000000000546f6b656e4272696467650100000001ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5',
 {'EmitterChainID': 1,
  'Meta': 'TokenBridge RegisterChain',
  'action': 1,
  'chain': 1,
  'chainRaw': b'\x00\x01',
  'consistency': 32,
  'digest': b'b%\xf5\xfd\x00\x00\x00\x03\x00\x01\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00'
            b'\x00\x03 \x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00TokenBridge\x01\x00\x00\x00\x01'
            b'\xecsr\x99]\\\xc8s#\x97\xfb\n\xd3\\\x01!\xe0\xea\xa9\r&\xf8(\xa5'
            b'4\xca\xb5C\x91\xb3\xa4\xf5',
  'emitter': b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
             b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04',
  'index': 2,
  'module': '000000000000000000000000000000000000000000546f6b656e427269646765',
  'nonce': 3,
  'sequence': 3,
  'siglen': 1,
  'signatures': b'\x00\xc2\xf0\xb6\xe5F\xe0\x93c\x02\x95\xe5\x00~\x8b\x07'
                b'{\x10(\xd3\xaa\x9ar\xabLEK&\x13\x06\xebOU\x01yc\x85\x97\xf2Z'
                b'\xfdo@\xa1\x85\x80\xbc\x87\xfa1UR\xe7)K@{\xd4ao\t\x95\xd1\xcb'
                b'U\x01',
  'sigs': ['00c2f0b6e546e093630295e5007e8b077b1028d3aa9a72ab4c454b261306eb4f550179638597f25afd6f40a18580bc87fa315552e7294b407bd4616f0995d1cb5501'],
  'targetChain': 0,
  'targetEmitter': 'ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5',
  'timestamp': 1646654973,
  'version': 1})
Sending 3000 algo to cover fees
[1000, 1000, 1000, 1000]
{0: 99997976000}
Registering chain 2
('010000000201008c7153db06d433e304dcb7dc029b6cb142093adf87eac7a14adff78060f9b80275479d0620612ae656f7281190ab7bbf85f31eb2ace579e77b2e7855af2a4504016225f5fe0000000400010000000000000000000000000000000000000000000000000000000000000004000000000000000420000000000000000000000000000000000000000000546f6b656e42726964676501000000020000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585',
 {'EmitterChainID': 2,
  'Meta': 'TokenBridge RegisterChain',
  'action': 1,
  'chain': 1,
  'chainRaw': b'\x00\x01',
  'consistency': 32,
  'digest': b'b%\xf5\xfe\x00\x00\x00\x04\x00\x01\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00'
            b'\x00\x04 \x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00TokenBridge\x01\x00\x00\x00\x02'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00>\xe1\x8b"'
            b'\x14\xaf\xf9p\x00\xd9t\xcfd~|4~\x8f\xa5\x85',
  'emitter': b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
             b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04',
  'index': 2,
  'module': '000000000000000000000000000000000000000000546f6b656e427269646765',
  'nonce': 4,
  'sequence': 4,
  'siglen': 1,
  'signatures': b'\x00\x8cqS\xdb\x06\xd43\xe3\x04\xdc\xb7\xdc\x02\x9bl\xb1B\t:'
                b'\xdf\x87\xea\xc7\xa1J\xdf\xf7\x80`\xf9\xb8\x02uG\x9d\x06 a*'
                b'\xe6V\xf7(\x11\x90\xab{\xbf\x85\xf3\x1e\xb2\xac\xe5y\xe7{.x'
                b'U\xaf*E\x04\x01',
  'sigs': ['008c7153db06d433e304dcb7dc029b6cb142093adf87eac7a14adff78060f9b80275479d0620612ae656f7281190ab7bbf85f31eb2ace579e77b2e7855af2a450401'],
  'targetChain': 0,
  'targetEmitter': '0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585',
  'timestamp': 1646654974,
  'version': 1})
Sending 3000 algo to cover fees
[1000, 1000, 1000, 1000]
{0: 99997967000}
Registering chain 3
('010000000201006896223475308eb13bc6d279b620b167f0e4884afc56942b2199faa81e1d50d83d74f7c0700254aa78a7e8966508608f0d827969df09745ad569575136551bce006225f5ff0000000500010000000000000000000000000000000000000000000000000000000000000004000000000000000520000000000000000000000000000000000000000000546f6b656e42726964676501000000030000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2',
 {'EmitterChainID': 3,
  'Meta': 'TokenBridge RegisterChain',
  'action': 1,
  'chain': 1,
  'chainRaw': b'\x00\x01',
  'consistency': 32,
  'digest': b'b%\xf5\xff\x00\x00\x00\x05\x00\x01\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00'
            b'\x00\x05 \x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00TokenBridge\x01\x00\x00\x00\x03'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00|\xf7\xb7d'
            b'\xe3\x8a\n^\x96yr\xc1\xdfw\xd42Q\x05d\xe2',
  'emitter': b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
             b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04',
  'index': 2,
  'module': '000000000000000000000000000000000000000000546f6b656e427269646765',
  'nonce': 5,
  'sequence': 5,
  'siglen': 1,
  'signatures': b'\x00h\x96"4u0\x8e\xb1;\xc6\xd2y\xb6 \xb1g\xf0\xe4\x88'
                b'J\xfcV\x94+!\x99\xfa\xa8\x1e\x1dP\xd8=t\xf7\xc0p\x02T'
                b'\xaax\xa7\xe8\x96e\x08`\x8f\r\x82yi\xdf\ttZ\xd5iWQ6U\x1b'
                b'\xce\x00',
  'sigs': ['006896223475308eb13bc6d279b620b167f0e4884afc56942b2199faa81e1d50d83d74f7c0700254aa78a7e8966508608f0d827969df09745ad569575136551bce00'],
  'targetChain': 0,
  'targetEmitter': '0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2',
  'timestamp': 1646654975,
  'version': 1})
Sending 3000 algo to cover fees
[1000, 1000, 1000, 1000]
{0: 99997958000}
Registering chain 4
('0100000002010023b80ca2402119348543c14134218cd0e1e54428e54ecdf21acb1a1d6c01be261fcc138023955a04bcd09230a5710340251b68db080a8bbf64d06ab744624d6a016225f5ff0000000600010000000000000000000000000000000000000000000000000000000000000004000000000000000620000000000000000000000000000000000000000000546f6b656e4272696467650100000004000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7',
 {'EmitterChainID': 4,
  'Meta': 'TokenBridge RegisterChain',
  'action': 1,
  'chain': 1,
  'chainRaw': b'\x00\x01',
  'consistency': 32,
  'digest': b'b%\xf5\xff\x00\x00\x00\x06\x00\x01\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00'
            b'\x00\x06 \x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00TokenBridge\x01\x00\x00\x00\x04'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xb6\xf6\xd8j'
            b'\x8f\x98y\xa9\xc8\x7fd7h\xd9\xef\xc3\x8c\x1d\xa6\xe7',
  'emitter': b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
             b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04',
  'index': 2,
  'module': '000000000000000000000000000000000000000000546f6b656e427269646765',
  'nonce': 6,
  'sequence': 6,
  'siglen': 1,
  'signatures': b'\x00#\xb8\x0c\xa2@!\x194\x85C\xc1A4!\x8c\xd0\xe1\xe5D'
                b'(\xe5N\xcd\xf2\x1a\xcb\x1a\x1dl\x01\xbe&\x1f\xcc\x13'
                b'\x80#\x95Z\x04\xbc\xd0\x920\xa5q\x03@%\x1bh\xdb\x08\n\x8b'
                b'\xbfd\xd0j\xb7DbMj\x01',
  'sigs': ['0023b80ca2402119348543c14134218cd0e1e54428e54ecdf21acb1a1d6c01be261fcc138023955a04bcd09230a5710340251b68db080a8bbf64d06ab744624d6a01'],
  'targetChain': 0,
  'targetEmitter': '000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7',
  'timestamp': 1646654975,
  'version': 1})
Sending 3000 algo to cover fees
[1000, 1000, 1000, 1000]
{0: 99997949000}
Registering chain 5
('010000000201003a168d6617cc74c3a5e254a6e65441d341cec315dcd5b588e72f781f8dd9c82977ad1234732d097151a54add996a33a6e4da3a2e80c41146de0bc834d8830661006225f6000000000700010000000000000000000000000000000000000000000000000000000000000004000000000000000720000000000000000000000000000000000000000000546f6b656e42726964676501000000050000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde',
 {'EmitterChainID': 5,
  'Meta': 'TokenBridge RegisterChain',
  'action': 1,
  'chain': 1,
  'chainRaw': b'\x00\x01',
  'consistency': 32,
  'digest': b'b%\xf6\x00\x00\x00\x00\x07\x00\x01\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00'
            b'\x00\x07 \x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
            b'\x00\x00\x00\x00\x00\x00\x00\x00TokenBridge\x01\x00\x00\x00\x05'
            b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00ZXPZ'
            b'\x96\xd1\xdb\xf8\xdf\x91\xcb!\xb5D\x19\xfc6\xe9?\xde',
  'emitter': b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
             b'\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04',
  'index': 2,
  'module': '000000000000000000000000000000000000000000546f6b656e427269646765',
  'nonce': 7,
  'sequence': 7,
  'siglen': 1,
  'signatures': b'\x00:\x16\x8df\x17\xcct\xc3\xa5\xe2T\xa6\xe6TA\xd3A\xce\xc3'
                b'\x15\xdc\xd5\xb5\x88\xe7/x\x1f\x8d\xd9\xc8)w\xad\x124s-\t'
                b'qQ\xa5J\xdd\x99j3\xa6\xe4\xda:.\x80\xc4\x11F\xde\x0b\xc8'
                b'4\xd8\x83\x06a\x00',
  'sigs': ['003a168d6617cc74c3a5e254a6e65441d341cec315dcd5b588e72f781f8dd9c82977ad1234732d097151a54add996a33a6e4da3a2e80c41146de0bc834d883066100'],
  'targetChain': 0,
  'targetEmitter': '0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde',
  'timestamp': 1646654976,
  'version': 1})

./sandbox down; ./sandbox clean; ./sandbox up dev -v; python3 admin.py --devnet


[jsiegel@gusc1a-ossdev-jsl1 ~/.../algorand/_sandbox]{master} git diff
diff --git a/images/indexer/start.sh b/images/indexer/start.sh
index 9e224c2..f1714ea 100755
--- a/images/indexer/start.sh
+++ b/images/indexer/start.sh
@@ -28,6 +28,7 @@ start_with_algod() {
   /tmp/algorand-indexer daemon \
     --dev-mode \
+    --enable-all-parameters \
     --server ":$PORT" \
     -P "$CONNECTION_STRING" \
     --algod-net "${ALGOD_ADDR}" \

--

#!/usr/bin/env bash

if [ ! -d _sandbox ]; then
  echo We need to create it...
  git clone https://github.com/algorand/sandbox.git _sandbox
fi

if [ "`grep enable-all-parameters _sandbox/images/indexer/start.sh | wc -l`" == "0" ]; then
  echo the indexer is incorrectly configured
  sed -i -e 's/dev-mode/dev-mode --enable-all-parameters/'  _sandbox/images/indexer/start.sh
  echo delete all the existing docker images
  ./sandbox clean
fi

./sandbox up dev

echo "run the tests"
cd test
python3 test.py

echo "bring the sandbox down"
cd ..
./sandbox down
<!-- cspell:enable -->
