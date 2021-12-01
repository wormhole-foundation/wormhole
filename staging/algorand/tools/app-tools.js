/*************************************************************************
 *  [2018] - [2020] Rand Labs Inc.
 *  All Rights Reserved.
 *
 * NOTICE:  All information contained herein is, and remains
 * the property of Rand Labs Inc.
 * The intellectual and technical concepts contained
 * herein are proprietary to Rand Labs Inc.
 */
const sha512 = require('js-sha512')
const hibase32 = require('hi-base32')

const ALGORAND_ADDRESS_SIZE = 58

function timeoutPromise (ms, promise) {
  return new Promise((resolve, reject) => {
    const timeoutId = setTimeout(() => {
      reject(new Error('promise timeout'))
    }, ms)
    promise.then(
      (res) => {
        clearTimeout(timeoutId)
        resolve(res)
      },
      (err) => {
        clearTimeout(timeoutId)
        reject(err)
      }
    )
  })
}

function getInt64Bytes (x, len) {
  if (!len) {
    len = 8
  }
  const bytes = new Uint8Array(len)
  do {
    len -= 1
    // eslint-disable-next-line no-bitwise
    bytes[len] = x & (255)
    // eslint-disable-next-line no-bitwise
    x >>= 8
  } while (len)
  return bytes
}

function addressFromByteBuffer (addr) {
  const bytes = Buffer.from(addr, 'base64')

  // compute checksum
  const checksum = sha512.sha512_256.array(bytes).slice(28, 32)

  const c = new Uint8Array(bytes.length + checksum.length)
  c.set(bytes)
  c.set(checksum, bytes.length)

  const v = hibase32.encode(c)

  return v.toString().slice(0, ALGORAND_ADDRESS_SIZE)
}

function printAppCallDeltaArray (deltaArray) {
  for (let i = 0; i < deltaArray.length; i++) {
    if (deltaArray[i].address) {
      console.log('Local state change address: ' + deltaArray[i].address)
      for (let j = 0; j < deltaArray[i].delta.length; j++) {
        printAppCallDelta(deltaArray[i].delta[j])
      }
    } else {
      console.log('Global state change')
      printAppCallDelta(deltaArray[i])
    }
  }
}

function printAppStateArray (stateArray) {
  for (let n = 0; n < stateArray.length; n++) {
    printAppState(stateArray[n])
  }
}

function appValueState (stateValue) {
  let text = ''

  if (stateValue.type == 1) {
    const addr = addressFromByteBuffer(stateValue.bytes)
    if (addr.length == ALGORAND_ADDRESS_SIZE) {
      text += addr
    } else {
      text += stateValue.bytes
    }
  } else if (stateValue.type == 2) {
    text = stateValue.uint
  } else {
    text += stateValue.bytes
  }

  return text
}

function appValueStateString (stateValue) {
  let text = ''

  if (stateValue.type == 1) {
    const addr = addressFromByteBuffer(stateValue.bytes)
    if (addr.length == ALGORAND_ADDRESS_SIZE) {
      text += addr
    } else {
      text += stateValue.bytes
    }
  } else if (stateValue.type == 2) {
    text += stateValue.uint
  } else {
    text += stateValue.bytes
  }

  return text
}

function printAppState (state) {
  let text = Buffer.from(state.key, 'base64').toString() + ': '

  text += appValueStateString(state.value)

  console.log(text)
}

async function printAppLocalState (algodClient, appId, accountAddr) {
  const ret = await readAppLocalState(algodClient, appId, accountAddr)
  if (ret) {
    console.log('Application %d local state for account %s:', appId, accountAddr)
    printAppStateArray(ret)
  }
}

async function printAppGlobalState (algodClient, appId, accountAddr) {
  const ret = await readAppGlobalState(algodClient, appId, accountAddr)
  if (ret) {
    console.log('Application %d global state:', appId)
    printAppStateArray(ret)
  }
}

async function readCreatedApps (algodClient, accountAddr) {
  const accountInfoResponse = await algodClient.accountInformation(accountAddr).do()
  return accountInfoResponse['created-apps']
}

async function readOptedInApps (algodClient, accountAddr) {
  const accountInfoResponse = await algodClient.accountInformation(accountAddr).do()
  return accountInfoResponse['apps-local-state']
}

// read global state of application
async function readAppGlobalState (algodClient, appId, accountAddr) {
  const accountInfoResponse = await algodClient.accountInformation(accountAddr).do()
  for (let i = 0; i < accountInfoResponse['created-apps'].length; i++) {
    if (accountInfoResponse['created-apps'][i].id === appId) {
      const globalState = accountInfoResponse['created-apps'][i].params['global-state']

      return globalState
    }
  }
}

async function readAppGlobalStateByKey (algodClient, appId, accountAddr, key) {
  const accountInfoResponse = await algodClient.accountInformation(accountAddr).do()
  for (let i = 0; i < accountInfoResponse['created-apps'].length; i++) {
    if (accountInfoResponse['created-apps'][i].id === appId) {
      // console.log("Application's global state:")
      const stateArray = accountInfoResponse['created-apps'][i].params['global-state']
      for (let j = 0; j < stateArray.length; j++) {
        const text = Buffer.from(stateArray[j].key, 'base64').toString()

        if (key === text) {
          return appValueState(stateArray[j].value)
        }
      }
    }
  }
}

// read local state of application from user account
async function readAppLocalState (algodClient, appId, accountAddr) {
  const accountInfoResponse = await algodClient.accountInformation(accountAddr).do()
  for (let i = 0; i < accountInfoResponse['apps-local-state'].length; i++) {
    if (accountInfoResponse['apps-local-state'][i].id === appId) {
      // console.log(accountAddr + " opted in, local state:")

      if (accountInfoResponse['apps-local-state'][i]['key-value']) {
        return accountInfoResponse['apps-local-state'][i]['key-value']
      }
    }
  }
}

async function readAppLocalStateByKey (algodClient, appId, accountAddr, key) {
  const accountInfoResponse = await algodClient.accountInformation(accountAddr).do()
  for (let i = 0; i < accountInfoResponse['apps-local-state'].length; i++) {
    if (accountInfoResponse['apps-local-state'][i].id === appId) {
      const stateArray = accountInfoResponse['apps-local-state'][i]['key-value']

      if (!stateArray) {
        return null
      }
      for (let j = 0; j < stateArray.length; j++) {
        const text = Buffer.from(stateArray[j].key, 'base64').toString()

        if (key === text) {
          return appValueState(stateArray[j].value)
        }
      }
      // not found assume 0
      return 0
    }
  }
}

function uintArray8ToString (byteArray) {
  return Array.from(byteArray, function (byte) {
    // eslint-disable-next-line no-bitwise
    return ('0' + (byte & 0xFF).toString(16)).slice(-2)
  }).join('')
}

/**
 * Verify if transactionResponse has any information about a transaction local or global state change.
 * @param  {Object} transactionResponse object containing the transaction response of an application call
 * @return {Boolean} returns true if there is a local or global delta meanining that
 * the transaction made a change in the local or global state
 */
function anyAppCallDelta (transactionResponse) {
  return (transactionResponse['global-state-delta'] || transactionResponse['local-state-delta'])
}

/**
 * Print to stdout the changes introduced by the transaction that generated the transactionResponse if any.
 * @param  {Object} transactionResponse object containing the transaction response of an application call
 * @return {void}
 */
function printAppCallDelta (transactionResponse) {
  if (transactionResponse['global-state-delta'] !== undefined) {
    console.log('Global State updated:')
    printAppCallDeltaArray(transactionResponse['global-state-delta'])
  }
  if (transactionResponse['local-state-delta'] !== undefined) {
    console.log('Local State updated:')
    printAppCallDeltaArray(transactionResponse['local-state-delta'])
  }
}

module.exports = {
  timeoutPromise,
  getInt64Bytes,
  addressFromByteBuffer,
  printAppStateArray,
  printAppState,
  printAppLocalState,
  printAppGlobalState,
  readCreatedApps,
  readOptedInApps,
  readAppGlobalState,
  readAppGlobalStateByKey,
  readAppLocalState,
  readAppLocalStateByKey,
  uintArray8ToString,
  anyAppCallDelta,
  printAppCallDelta
}
