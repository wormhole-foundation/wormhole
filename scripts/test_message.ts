import algosdk from 'algosdk'
import { Buffer } from 'buffer'
import { writeFileSync } from 'fs'

// OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU
const mnemo = 'assault approve result rare float sugar power float soul kind galaxy edit unusual pretty tone tilt net range pelican avoid unhappy amused recycle abstract master'

const appId = BigInt(0x123456789)
const nonce = BigInt(0x1000)
const symbol = 'BTC/USD         '
const price = 45278.65
const sd = 8.00000000004

// Create message
const buf = Buffer.alloc(131)
buf.write('PRICEDATA', 0)
// v
buf.writeInt8(1, 9)
// dest
buf.writeBigUInt64BE(appId, 10)
// nonce
buf.writeBigUInt64BE(nonce, 18)
// symbol
buf.write(symbol, 26)
// price
buf.writeDoubleBE(price, 42)
// sd
buf.writeDoubleBE(sd, 50)
// ts
buf.writeBigUInt64BE(BigInt(Date.now()), 58)

const signature = Buffer.from(algosdk.signBytes(buf, algosdk.mnemonicToSecretKey(mnemo).sk))
signature.copy(buf, 66)

// v-component (ignored in Algorand it seems)
buf.writeInt8(1, 130)

writeFileSync('msg.bin', buf)
writeFileSync('msg.b64', buf.toString('base64'))
