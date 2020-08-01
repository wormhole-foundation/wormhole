/**
 * Exercises the token program
 *
 * @flow
 */

import {
  loadTokenProgram,
  createMint,
  createAccount,
  transfer,
  approveRevoke,
  invalidApprove,
  failOnApproveOverspend,
  setOwner,
  mintTo,
  multisig,
  burn,
  failOnCloseAccount,
  nativeToken,
} from './token-test';

async function main() {
  console.log('Run test: loadTokenProgram');
  await loadTokenProgram();
  console.log('Run test: createMint');
  await createMint();
}

main()
  .catch(err => {
    console.error(err);
    process.exit(-1);
  })
  .then(() => process.exit());
