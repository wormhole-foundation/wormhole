/**
 *
 * Pricecaster Service Utility Library.
 * (c) 2021-22 Randlabs, Inc.
 *
 */

const algosdk = require('algosdk')
const fs = require('fs')
// eslint-disable-next-line camelcase
const { sha512_256 } = require('js-sha512')
const tools = require('../tools/app-tools')

const approvalProgramFilename = 'teal/pricekeeper/pricekeeper.teal'
const clearProgramFilename = 'teal/pricekeeper/clearstate.teal'
let vaaProcessorApprovalProgramFilename = 'teal/wormhole/build/vaa-processor-approval.teal'
let vaaProcessorClearProgramFilename = 'teal/wormhole/build/vaa-processor-clear.teal'
const vaaVerifyStatelessProgramFilename = 'teal/wormhole/build/vaa-verify.teal'

class PricecasterLib {
  constructor (algodClient, ownerAddr = undefined) {
    this.algodClient = algodClient
    this.ownerAddr = ownerAddr
    this.minFee = 1000
    this.groupTx = []
    this.lsigs = {}

    /** Overrides the default VAA processor approval program filename
     * @param {string} filename New file name to use.
     */
    this.setVaaProcessorApprovalFile = function (filename) {
      vaaProcessorApprovalProgramFilename = filename
    }

    /** Overrides the default VAA processor clear-state program filename
     * @param {string} filename New file name to use.
     */
    this.setVaaProcessorClearStateFile = function (filename) {
      vaaProcessorClearProgramFilename = filename
    }

    /**
         * Set Application Id used in all the functions of this class.
         * @param {number} applicationId application id
         * @returns {void}
         */
    this.setAppId = function (applicationId) {
      this.appId = applicationId
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
      return { compiledBytes, hash: compileResponse.hash }
    }

    /**
         * Internal function.
         * Compile application clear state program.
         * @return {String} base64 string containing the compiled program
         */
    this.compileClearProgram = function () {
      const program = fs.readFileSync(clearProgramFilename, 'utf8')
      return this.compileProgram(program)
    }

    /**
         * Internal function.
         * Compile application clear state program.
         * @return {String} base64 string containing the compiled program
         */
    this.compileVAAProcessorClearProgram = function () {
      const program = fs.readFileSync(vaaProcessorClearProgramFilename, 'utf8')
      return this.compileProgram(program)
    }

    /**
         * Internal function.
         * Compile pricekeeper application approval program.
         * @return {String} base64 string containing the compiled program
         */
    this.compileApprovalProgram = async function () {
      const program = fs.readFileSync(approvalProgramFilename, 'utf8')
      const compiledApprovalProgram = await this.compileProgram(program)
      this.approvalProgramHash = compiledApprovalProgram.hash
      return compiledApprovalProgram
    }

    /**
     * Internal function.
     * Compile VAA Processor application approval program.
     * @return {String} base64 string containing the compiled program
     */
    this.compileVAAProcessorApprovalProgram = async function () {
      const program = fs.readFileSync(vaaProcessorApprovalProgramFilename, 'utf8')
      const compiledApprovalProgram = await this.compileProgram(program)
      this.approvalProgramHash = compiledApprovalProgram.hash
      return compiledApprovalProgram
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
    this.createApp = async function (sender, validatorAddr, symbol, signCallback) {
      if (symbol.length > 16) {
        throw new Error('Symbol exceeds 16 characters')
      }
      symbol = symbol.padEnd(16, ' ')
      const localInts = 0
      const localBytes = 0
      const globalInts = 4
      const globalBytes = 3

      // declare onComplete as NoOp
      const onComplete = algosdk.OnApplicationComplete.NoOpOC

      // get node suggested parameters
      const params = await algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      const compiledProgram = await this.compileApprovalProgram()
      const approvalProgramCompiled = compiledProgram.compiledBytes
      const clearProgramCompiled = (await this.compileClearProgram()).compiledBytes

      const enc = new TextEncoder()
      const appArgs = [new Uint8Array(algosdk.decodeAddress(validatorAddr).publicKey), enc.encode(symbol)]

      // create unsigned transaction
      const txApp = algosdk.makeApplicationCreateTxn(
        sender, params, onComplete,
        approvalProgramCompiled, clearProgramCompiled,
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
      const localInts = 0
      const localBytes = 0
      const globalInts = 4
      const globalBytes = 20

      // declare onComplete as NoOp
      const onComplete = algosdk.OnApplicationComplete.NoOpOC

      // get node suggested parameters
      const params = await algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      const compiledProgram = await this.compileVAAProcessorApprovalProgram()
      const approvalProgramCompiled = compiledProgram.compiledBytes
      const clearProgramCompiled = (await this.compileVAAProcessorClearProgram()).compiledBytes
      const appArgs = [new Uint8Array(Buffer.from(gkeys, 'hex')),
        algosdk.encodeUint64(parseInt(gexpTime)),
        algosdk.encodeUint64(parseInt(gsindex))]

      // create unsigned transaction
      const txApp = algosdk.makeApplicationCreateTxn(
        sender, params, onComplete,
        approvalProgramCompiled, clearProgramCompiled,
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
         * Internal function.
         * Call application specifying args and accounts.
         * @param  {String} sender caller address
         * @param  {Array} appArgs array of arguments to pass to application call
         * @param  {Array} appAccounts array of accounts to pass to application call
         * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
         * @return {String} transaction id of the transaction
         */
    this.callApp = async function (sender, appArgs, appAccounts, signCallback) {
      // get node suggested parameters
      const params = await this.algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      // create unsigned transaction
      const txApp = algosdk.makeApplicationNoOpTxn(sender, params, this.appId, appArgs, appAccounts.length === 0 ? undefined : appAccounts)
      const txId = txApp.txID().toString()

      // Sign the transaction
      const txAppSigned = signCallback(sender, txApp)

      // Submit the transaction
      await this.algodClient.sendRawTransaction(txAppSigned).do()

      return txId
    }

    /**
     * Internal function.
     * Call application specifying args and accounts.  Do it in a group of dummy TXs for maximizing computations.
     * @param  {String} sender caller address
     * @param  {Array} appArgs array of arguments to pass to application call
     * @param  {Array} appAccounts array of accounts to pass to application call
     * @param  {Function} signCallback callback with prototype signCallback(sender, tx) used to sign transactions
     * @param  {number} dummyTxCount the number of dummyTx to submit, with the real call last.
     * @return {String} transaction id of the transaction
     */
    this.callAppInDummyGroup = async function (sender, appArgs, appAccounts, signCallback, dummyTxCount) {
      // get node suggested parameters
      const params = await this.algodClient.getTransactionParams().do()

      params.fee = this.minFee
      params.flatFee = true

      // console.log(appArgs)

      const txns = []
      const enc = new TextEncoder()
      for (let i = 0; i < dummyTxCount; ++i) {
        txns.push(algosdk.makeApplicationNoOpTxn(sender,
          params,
          this.appId,
          undefined, undefined, undefined, undefined,
          enc.encode(`dummy_TX_${i}`)))
      }
      const appTx = algosdk.makeApplicationNoOpTxn(sender, params, this.appId, appArgs)
      txns.push(appTx)
      algosdk.assignGroupID(txns)
      const txId = appTx.txID().toString()

      // Sign the transactions
      const signedTxns = []
      for (const tx of txns) {
        signedTxns.push(signCallback(sender, tx))
      }

      // Submit the transaction
      await this.algodClient.sendRawTransaction(signedTxns).do()

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
     * Creates a message with price data for the PriceKeeper contract
     * @param {String} symbol Symbol, must match appid support, 16-char UTF long
     * @param {BigInt} price Aggregated price
     * @param {BigInt} confidence Confidence
     * @param {BigInt} exp Exponent (positive)
     * @param {BigInt} slot Valid-slot of price aggregation
     * @param {Uint8Array} sk Signing key.
     * @param {string} header (optional) Message header.  'PRICEDATA' if undefined.
     * @param {BigInt} appId (optional)  AppId. Default is this contract appId.
     * @param {number} version (optional) Version. Default is 1 if undefined.
     * @param {BigInt} ts (optional) Timestamp of message. Current system ts if undefined.
     * @returns A base64-encoded message.
     */
    this.createMessage = function (symbol, price, exp, confidence, slot, sk, header, appId, version, ts) {
      const buf = Buffer.alloc(138)
      buf.write(header === undefined ? 'PRICEDATA' : header, 0)
      buf.writeInt8(version === undefined ? 1 : version, 9)
      buf.writeBigUInt64BE(appId === undefined ? BigInt(this.appId) : appId, 10)
      buf.write(symbol, 18)
      buf.writeBigUInt64BE(price, 34)

      // (!) Libraries like Pyth publish negative exponents. Write as signed 64bit

      buf.writeBigInt64BE(exp, 42)
      buf.writeBigUInt64BE(confidence, 50)
      buf.writeBigUInt64BE(slot, 58)
      buf.writeBigUInt64BE(ts === undefined ? BigInt(Math.floor(Date.now() / 1000)) : ts, 66)

      const digestu8 = Buffer.from(sha512_256(buf.slice(0, 74)), 'hex')

      const signature = Buffer.from(algosdk.tealSign(sk, digestu8, this.approvalProgramHash))
      signature.copy(buf, 74)
      return buf
    }

    /**
     * Submits message to the PriceKeeper contract.
     * @param {*} sender Sender account
     * @param {*} msgb64 Base64-encoded message.
     * @returns Transaction identifier (txid)
     */
    this.submitMessage = async function (sender, msgBuffer, signCallback) {
      if (!algosdk.isValidAddress(sender)) {
        throw new Error('Invalid sender address: ' + sender)
      }
      const appArgs = []
      appArgs.push(new Uint8Array(msgBuffer))
      return await this.callAppInDummyGroup(sender, appArgs, [], signCallback, 3)
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
      return await this.callApp(sender, appArgs, [], signCallback)
    }

    /**
     * Starts a begin...commit section for commiting grouped transactions.
     */
    this.beginTxGroup = function () {
      this.groupTx = []
    }

    /**
     * Adds a transaction to the group.
     * @param {} tx Transaction to add.
     */
    this.addTxToGroup = function (tx) {
      this.groupTx.push(tx)
    }

    /**
     * @param {*} sender The sender account.
     * @param {function} signCallback The sign callback routine.
     * @returns Transaction id.
     */
    this.commitTxGroup = async function (sender, signCallback) {
      algosdk.assignGroupID(this.groupTx)

      // Sign the transactions
      const signedTxns = []
      for (const tx of this.groupTx) {
        signedTxns.push(signCallback(sender, tx))
      }

      // Submit the transaction
      const tx = await this.algodClient.sendRawTransaction(signedTxns).do()
      this.groupTx = []
      return tx.txId
    }

    /**
     * @param {*} sender The sender account.
     * @param {*} programBytes Compiled program bytes.
     * @param {*} sigSubsets The signature subsets i..j for logicsig arguments.
     * @returns Transaction id.
     */
    this.commitVerifyTxGroup = async function (programBytes, sigSubsets) {
      algosdk.assignGroupID(this.groupTx)
      const signedGroup = []
      let i = 0
      for (const tx of this.groupTx) {
        const lsig = new algosdk.LogicSigAccount(programBytes, [Buffer.from(sigSubsets[i++], 'hex')])
        const stxn = algosdk.signLogicSigTransaction(tx, lsig)
        signedGroup.push(stxn.blob)
      }

      // Save transaction for debugging.

      // fs.unlinkSync('signedgroup.stxn')

      // for (let i = 0; i < signedGroup.length; ++i) {
      //   fs.appendFileSync('signedgroup.stxn', signedGroup[i])
      // }

      // const dr = await algosdk.createDryrun({
      //   client: algodClient,
      //   txns: drtxns,
      //   sources: [new algosdk.modelsv2.DryrunSource('lsig', fs.readFileSync(vaaVerifyStatelessProgramFilename).toString('utf8'))]
      // })
      // // const drobj = await algodClient.dryrun(dr).do()
      // fs.writeFileSync('dump.dr', algosdk.encodeObj(dr.get_obj_for_encoding(true)))

      // Submit the transaction
      const tx = await this.algodClient.sendRawTransaction(signedGroup).do()
      this.groupTx = []
      return tx.txId
    }

    /**
     * VAA Processor: Add a verification step to a transaction group.
     * @param {*} sender The sender account (typically the VAA verification stateless program)
     * @param {*} payload The VAA payload.
     * @param {*} gksubset An hex string containing the keys for the guardian subset in this step.
     * @param {*} totalguardians The total number of known guardians.
     */
    this.addVerifyTx = function (sender, params, payload, gksubset, totalguardians) {
      const appArgs = []
      appArgs.push(new Uint8Array(Buffer.from('verify')),
        new Uint8Array(Buffer.from(gksubset.join(''), 'hex')),
        algosdk.encodeUint64(parseInt(totalguardians)))

      const tx = algosdk.makeApplicationNoOpTxn(sender,
        params,
        this.appId,
        appArgs, undefined, undefined, undefined,
        new Uint8Array(payload))
      this.groupTx.push(tx)

      return tx.txID()
    }
  }
}

module.exports = {
  PricecasterLib
}
