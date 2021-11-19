const TestLib = require('../test/testlib.js')
const t = new TestLib.TestLib()
const sigkeys = ['563d8d2fd4e701901d3846dee7ae7a92c18f1975195264d676f8407ac5976757', '8d97f25916a755df1d9ef74eb4dbebc5f868cb07830527731e94478cdc2b9d5f', '9bd728ad7617c05c31382053b57658d4a8125684c0098f740a054d87ddc0e93b']
const vaa = t.createSignedVAA(0, sigkeys, 1, 1, 1, '0x71f8dcb863d176e2c420ad6610cf687359612b6fb392e0642b0ca6b1f186aa3b', 0, 0, '0x12345678')
if (process.argv[2] === '--sig') {
  console.log(vaa.substr(12, sigkeys.length * 132))
} else if (process.argv[2] === '--body') {
  console.log(vaa.substr(12 + sigkeys.length * 132))
} else {
  console.log(vaa)
}
