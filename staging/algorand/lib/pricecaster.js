/**
 *
 * Pricecaster Service Utility Library.
 *
 * Copyright 2022 Wormhole Project Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

const algosdk = require('algosdk')
const fs = require('fs')
// eslint-disable-next-line camelcase
const tools = require('../tools/app-tools')
const crypto = require('crypto')

const ContractInfo = {
  pricekeeper: {
    approvalProgramFile: 'teal/wormhole/build/pricekeeper-v2-approval.teal',
    clearStateProgramFile: 'teal/wormhole/build/pricekeeper-v2-clear.teal',
    compiledApproval: {
      bytes: undefined,
      hash: undefined
    },
    compiledClearState: {
      bytes: undefined,
      hash: undefined
    },
    appId: 0
  },
  vaaProcessor: {
    approvalProgramFile: 'teal/wormhole/build/vaa-processor-approval.teal',
    clearStateProgramFile: 'teal/wormhole/build/vaa-processor-clear.teal',
    approvalProgramHash: '',
    compiledApproval: {
      bytes: undefined,
      hash: undefined
    },
    compiledClearState: {
      bytes: undefined,
      hash: undefined
    },
    appId: 0
  }
}

// --------------------------------------------------------------------------------------

class PricecasterLib {
  constructor(algodClient, ownerAddr = undefined) {
    this.algodClient = algodClient
    this.ownerAddr = ownerAddr
    this.minFee = 1000
    this.groupTxSet = {}
    this.lsigs = {}
    this.dumpFailedTx = false
    this.dumpFailedTxDirectory = './'

    /** Set the file dumping feature on failed group transactions
     * @param {boolean} f Set to true to enable function, false to disable.
     */
    this.enableDumpFailedTx = function (f) {
      this.dumpFailedTx = f
    }

    /** Set the file dumping feature output directory
     * @param {string} dir The output directory.
     */
    this.setDumpFailedTxDirectory = function (dir) {
      this.dumpFailedTxDirectory = dir
    }

    /** Sets a contract approval program filename
     * @param {string} filename New file name to use.
     */
    this.setApprovalProgramFile = function (contract, filename) {
      ContractInfo[contract].approvalProgramFile = filename
    }

    /** Sets a contract clear state program filename
     * @param {string} filename New file name to use.
     */
    this.setClearStateProgramFile = function (contract, filename) {
      ContractInfo[contract].clearStateProgramFile = filename
    }

    /**
     * Set Application Id for a contract.
     * @param {number} applicationId application id
     * @returns {void}
     */
    this.setAppId = function (contract, applicationId) {
      ContractInfo[contract].appId = applicationId
    }

    /**
     * Get the Application id for a specific contract
     * @returns The requested application Id
     */
    this.getAppId = function (contract) {
      return ContractInfo[contract].appId
    }

    /**
     * Get minimum fee to pay for transactions.
     * @return {Number} minimum transaction fee
     */
    this.minTransactionFee = function () {
      return this.minFee
    }

    /**
     * Internal function.
     * Read application local state related to the account.
     * @param  {String} accountAddr account to retrieve local state
     * @return {Array} an array containing all the {key: value} pairs of the local state
     */
    this.readLocalState = function (accountAddr) {
      return tools.readAppLocalState(this.algodClient, this.appId, accountAddr)
    }

    /**
     * Internal function.
     * Read application global state.
     * @return {Array} an array containing all the {key: value} pairs of the global state
     * @returns {void}
     */
    this.readGlobalState = function () {
      return tools.readAppGlobalState(this.algodClient, this.appId, this.ownerAddr)
    }

    /**
     * Print local state of accountAddr on stdout.
     * @param  {String} accountAddr account to retrieve local state
     * @returns {void}
     */
    this.printLocalState = async function (accountAddr) {
      await tools.printAppLocalState(this.algodClient, this.appId, accountAddr)
    }

    /**
     * Print application global state on stdout.
     * @returns {void}
     */
    this.printGlobalState = async function () {
      await tools.printAppGlobalState(this.algodClient, this.appId, this.ownerAddr)
    }

    /**
     * Internal function.
     * Read application local state variable related to accountAddr.
     * @param  {String} accountAddr account to retrieve local state
     * @param  {String} key variable key to get the value associated
     * @return {String/Number} it returns the value associated to the key that could be an address, a number or a
     * base64 string containing a ByteArray
     */
    this.readLocalStateByKey = function (accountAddr, key) {
      return tools.readAppLocalStateByKey(this.algodClient, this.appId, accountAddr, key)
    }

    /**
     * Internal function.
     * Read application global state variable.
     * @param  {String} key variable key to get the value associated
     * @return {String/Number} it returns the value associated to the key that could be an address,
     * a number or a base64 string containing a ByteArray
     */
    this.readGlobalStateByKey = function (key) {
      return tools.readAppGlobalStateByKey(this.algodClient, this.appId, this.ownerAddr, key)
    }

    /**
     * Compile program that programFilename contains.
     * @param  {String} programFilename filepath to the program to compile
     * @return {String} base64 string containing the compiled program
     */
    this.compileProgram = async function (programBytes) {
      const compileResponse = await this.algodClient.compile(programBytes).do()
      const compiledBytes = new Uint8Array(Buffer.from(compileResponse.result, 'base64'))
      return { bytes: compiledBytes, hash: compileResponse.hash }
    }

    /**
     * Compile clear state program.
     */
    this.compileClearProgram = async function (contract) {
      const program = fs.readFileSync(ContractInfo[contract].clearStateProgramFile, 'utf8')
      ContractInfo[contract].compiledClearState = await this.compileProgram(program)
    }

    /**
     * Compile approval program.
     */
    this.compileApprovalProgram = async function (contract) {
      const program = fs.readFileSync(ContractInfo[contract].approvalProgramFile, 'utf8')
      ContractInfo[contract].compiledApproval = await this.compileProgram(program)
    }

    /**
     * Helper function to retrieve the application id from a createApp transaction response.
     * @param  {Object} txResponse object containig the transactionResponse of the createApp call
     * @return {Number} application id of the created application
     */
    this.appIdFromCreateAppResponse = function (txResponse) {
      return txResponse['application-index']
    }

    /**
     * Create an application based on the default approval and clearState programs or based on the specified files.
     * @param  {String} sender account used to sign the createApp transaction
     * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
     * @return {String} transaction id of the created application
     */
    this.createApp = async function (sender, contract, localInts, localBytes, globalInts, globalBytes, appArgs, signCallback) {
      const onComplete = algosdk.OnApplicationComplete.NoOpOC

      // get node suggested parameters
      const params = await algodClient.getTransactionParams().do()
      params.fee = this.minFee
      params.flatFee = true

      await this.compileApprovalProgram(contract)
      await this.compileClearProgram(contract)

      // create unsigned transaction
      const txApp = algosdk.makeApplicationCreateTxn(
        sender, params, onComplete,
        ContractInfo[contract].compiledApproval.bytes,
        ContractInfo[contract].compiledClearState.bytes,
        localInts, localBytes, globalInts, globalBytes, appArgs
      )
      const txId = txApp.txID().toString()

      // Sign the transaction
      const txAppSigned = signCallback(sender, txApp)

      // Submit the transaction
      await algodClient.sendRawTransaction(txAppSigned).do()
      return txId
    }

    /**
     * Create the VAA Processor application based on the default approval and clearState programs or based on the specified files.
     * @param  {String} sender account used to sign the createApp transaction
     * @param  {String} gexpTime Guardian key set expiration time
     * @param  {String} gsindex Index of the guardian key set
     * @param  {String} gkeys Guardian keys listed as a single array
     * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
     * @return {String} transaction id of the created application
     */
    this.createVaaProcessorApp = async function (sender, gexpTime, gsindex, gkeys, signCallback) {
      return await this.createApp(sender, 'vaaProcessor', 0, 0, 5, 20,
        [new Uint8Array(Buffer.from(gkeys, 'hex')),
        algosdk.encodeUint64(parseInt(gexpTime)),
        algosdk.encodeUint64(parseInt(gsindex))], signCallback)
    }

    /**
       * Create the Pricekeeper application based on the default approval and clearState programs or based on the specified files.
       * @param  {String} sender account used to sign the createApp transaction
       * @param  {String} vaaProcessorAppId The application id of the VAA Processor program associated.
       * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
       * @return {String} transaction id of the created application
       */
    this.createPricekeeperApp = async function (sender, vaaProcessorAppId, signCallback) {
      return await this.createApp(sender, 'pricekeeper', 0, 0, 1, 63,
        [algosdk.encodeUint64(parseInt(vaaProcessorAppId))], signCallback)
    }

    /**
     * Internal function.
     * Call application specifying args and accounts.
     * @param  {String} sender caller address
     * @param  {Array} appArgs array of arguments to pass to application call
     * @param  {Array} appAccounts array of accounts to pass to application call
     * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
     * @return {String} transaction id of the transaction
     */
    this.callApp = async function (sender, contract, appArgs, appAccounts, signCallback) {
      // get node suggested parameters
      const params = await this.algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      // create unsigned transaction
      const txApp = algosdk.makeApplicationNoOpTxn(sender, params, ContractInfo[contract].appId, appArgs, appAccounts.length === 0 ? undefined : appAccounts)
      const txId = txApp.txID().toString()

      // Sign the transaction
      const txAppSigned = signCallback(sender, txApp)

      // Submit the transaction
      await this.algodClient.sendRawTransaction(txAppSigned).do()

      return txId
    }

    /**
     * ClearState sender. Remove all the sender associated local data.
     * @param  {String} sender account to ClearState
     * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
     * @return {[String]} transaction id of one of the transactions of the group
     */
    this.clearApp = async function (sender, signCallback, forcedAppId) {
      // get node suggested parameters
      const params = await this.algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      let appId = this.appId
      if (forcedAppId) {
        appId = forcedAppId
      }

      // create unsigned transaction
      const txApp = algosdk.makeApplicationClearStateTxn(sender, params, appId)
      const txId = txApp.txID().toString()

      // Sign the transaction
      const txAppSigned = signCallback(sender, txApp)

      // Submit the transaction
      await this.algodClient.sendRawTransaction(txAppSigned).do()

      return txId
    }

    /**
      * Permanent delete the application.
      * @param  {String} sender owner account
      * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
      * @param  {Function} applicationId use this application id instead of the one set
      * @return {String}      transaction id of one of the transactions of the group
      */
    this.deleteApp = async function (sender, signCallback, applicationId) {
      // get node suggested parameters
      const params = await this.algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      if (!applicationId) {
        applicationId = this.appId
      }

      // create unsigned transaction
      const txApp = algosdk.makeApplicationDeleteTxn(sender, params, applicationId)
      const txId = txApp.txID().toString()

      // Sign the transaction
      const txAppSigned = signCallback(sender, txApp)

      // Submit the transaction
      await this.algodClient.sendRawTransaction(txAppSigned).do()

      return txId
    }

    /**
     * Helper function to wait until transaction txId is included in a block/round.
     * @param  {String} txId transaction id to wait for
     * @return {VOID} VOID
     */
    this.waitForConfirmation = async function (txId) {
      const status = (await this.algodClient.status().do())
      let lastRound = status['last-round']
      // eslint-disable-next-line no-constant-condition
      while (true) {
        const pendingInfo = await this.algodClient.pendingTransactionInformation(txId).do()
        if (pendingInfo['confirmed-round'] !== null && pendingInfo['confirmed-round'] > 0) {
          // Got the completed Transaction

          return pendingInfo['confirmed-round']
        }
        lastRound += 1
        await this.algodClient.statusAfterBlock(lastRound).do()
      }
    }

    /**
     * Helper function to wait until transaction txId is included in a block/round
     * and returns the transaction response associated to the transaction.
     * @param  {String} txId transaction id to get transaction response
     * @return {Object}      returns an object containing response information
     */
    this.waitForTransactionResponse = async function (txId) {
      // Wait for confirmation
      await this.waitForConfirmation(txId)

      // display results
      return this.algodClient.pendingTransactionInformation(txId).do()
    }

    /**
     * VAA Processor: Sets the stateless logic program hash
     * @param {*} sender Sender account
     * @param {*} hash  The stateless logic program hash
     * @returns Transaction identifier.
     */
    this.setVAAVerifyProgramHash = async function (sender, hash, signCallback) {
      if (!algosdk.isValidAddress(sender)) {
        throw new Error('Invalid sender address: ' + sender)
      }
      const appArgs = []
      appArgs.push(new Uint8Array(Buffer.from('setvphash')),
        algosdk.decodeAddress(hash).publicKey)
      return await this.callApp(sender, 'vaaProcessor', appArgs, [], signCallback)
    }

    /**
     * VAA Processor: Sets the authorized application id for last call
     * @param {*} sender Sender account
     * @param {*} appId  The assigned appId
     * @returns Transaction identifier.
     */
    this.setAuthorizedAppId = async function (sender, appId, signCallback) {
      if (!algosdk.isValidAddress(sender)) {
        throw new Error('Invalid sender address: ' + sender)
      }
      const appArgs = []
      appArgs.push(new Uint8Array(Buffer.from('setauthid')),
        algosdk.encodeUint64(appId))
      return await this.callApp(sender, 'vaaProcessor', appArgs, [], signCallback)
    }

    /**
     * Starts a begin...commit section for commiting grouped transactions.
     */
    this.beginTxGroup = function () {
      const gid = crypto.randomBytes(16).toString('hex')
      this.groupTxSet[gid] = []
      return gid
    }

    /**
       * Adds a transaction to the group.
       * @param {} tx Transaction to add.
       */
    this.addTxToGroup = function (gid, tx) {
      if (this.groupTxSet[gid] === undefined) {
        throw new Error('unknown tx group id')
      }
      this.groupTxSet[gid].push(tx)
    }

    /**
     * @param {*} sender The sender account.
     * @param {function} signCallback The sign callback routine.
     * @returns Transaction id.
     */
    this.commitTxGroup = async function (gid, sender, signCallback) {
      if (this.groupTxSet[gid] === undefined) {
        throw new Error('unknown tx group id')
      }
      algosdk.assignGroupID(this.groupTxSet[gid])

      // Sign the transactions
      const signedTxns = []
      for (const tx of this.groupTxSet[gid]) {
        signedTxns.push(signCallback(sender, tx))
      }

      // Submit the transaction
      const tx = await this.algodClient.sendRawTransaction(signedTxns).do()
      delete this.groupTxSet[gid]
      return tx.txId
    }

    /**
     * @param {*} sender The sender account.
     * @param {*} programBytes Compiled program bytes.
     * @param {*} totalSignatureCount Total signatures present in the VAA.
     * @param {*} sigSubsets An hex string with the signature subsets i..j for logicsig arguments.
     * @param {*} lastTxSender The sender of the last TX in the group.
     * @param {*} signCallback The signing callback function to use in the last TX of the group.
     * @returns Transaction id.
     */
    this.commitVerifyTxGroup = async function (gid, programBytes, totalSignatureCount, sigSubsets, lastTxSender, signCallback) {
      if (this.groupTxSet[gid] === undefined) {
        throw new Error('unknown group id')
      }
      algosdk.assignGroupID(this.groupTxSet[gid])
      const signedGroup = []
      let i = 0
      for (const tx of this.groupTxSet[gid]) {
        // All transactions except last must be signed by stateless code.

        // console.log(`sigSubsets[${i}]: ${sigSubsets[i])

        if (i === this.groupTxSet[gid].length - 1) {
          signedGroup.push(signCallback(lastTxSender, tx))
        } else {
          const lsig = new algosdk.LogicSigAccount(programBytes, [Buffer.from(sigSubsets[i], 'hex'), algosdk.encodeUint64(totalSignatureCount)])
          const stxn = algosdk.signLogicSigTransaction(tx, lsig)
          signedGroup.push(stxn.blob)
        }
        i++
      }

      // Submit the transaction
      let tx
      try {
        tx = await this.algodClient.sendRawTransaction(signedGroup).do()
      } catch (e) {
        if (this.dumpFailedTx) {
          const id = tx ? tx.txId : Date.now().toString()
          const filename = `${this.dumpFailedTxDirectory}/failed-${id}.stxn`
          if (fs.existsSync(filename)) {
            fs.unlinkSync(filename)
          }
          for (let i = 0; i < signedGroup.length; ++i) {
            fs.appendFileSync(filename, signedGroup[i])
          }
        }
        throw e
      }
      delete this.groupTxSet[gid]
      return tx.txId
    }

    /**
     * VAA Processor: Add a verification step to a transaction group.
     * @param {*} sender The sender account (typically the VAA verification stateless program)
     * @param {*} payload The VAA payload.
     * @param {*} gksubset An hex string containing the keys for the guardian subset in this step.
     * @param {*} totalguardians The total number of known guardians.
     */
    this.addVerifyTx = function (gid, sender, params, payload, gksubset, totalguardians) {
      if (this.groupTxSet[gid] === undefined) {
        throw new Error('unknown group id')
      }
      const appArgs = []
      appArgs.push(new Uint8Array(Buffer.from('verify')),
        new Uint8Array(Buffer.from(gksubset.join(''), 'hex')),
        algosdk.encodeUint64(parseInt(totalguardians)))

      const tx = algosdk.makeApplicationNoOpTxn(sender,
        params,
        ContractInfo.vaaProcessor.appId,
        appArgs, undefined, undefined, undefined,
        new Uint8Array(payload))
      this.groupTxSet[gid].push(tx)

      return tx.txID()
    }

    /**
     * Pricekeeper-V2: Add store price transaction to TX Group.
     * @param {*} sender The sender account (typically the VAA verification stateless program)
     * @param {*} sym The symbol identifying the product  to store price for.
     * @param {*} payload The VAA payload.
     */
    this.addPriceStoreTx = function (gid, sender, params, sym, payload) {
      if (this.groupTxSet[gid] === undefined) {
        throw new Error('unknown group id')
      }
      const appArgs = []
      appArgs.push(new Uint8Array(Buffer.from('store')),
        new Uint8Array(Buffer.from(sym)),
        new Uint8Array(payload))

      const tx = algosdk.makeApplicationNoOpTxn(sender,
        params,
        ContractInfo.pricekeeper.appId,
        appArgs)
      this.groupTxSet[gid].push(tx)

      return tx.txID()
    }
  }
}

module.exports = {
  PricecasterLib
}
