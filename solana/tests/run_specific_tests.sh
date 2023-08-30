#!/bin/bash

# core bridge
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/000__*.ts  # initialize
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/002__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/004__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/006__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/008__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/010__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/012__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/014__*.ts  # guardian set update
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/098__*.ts  # contract upgrade
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/100__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/102__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/104__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/110__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/112__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/01__**/114__*.ts

# token bridge
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/000__*.ts  # initialize
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/002__*.ts  # create wrapped
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/003__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/004__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/006__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/012__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/014__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/022__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/024__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/026__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/028__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/098__*.ts  # contract upgrade
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/100__*.ts  # register chain
npx ts-mocha -p ./tsconfig.json -t 60000 tests/02__**/102__*.ts

# mock cpi
npx ts-mocha -p ./tsconfig.json -t 60000 tests/03__**/100__*.ts
npx ts-mocha -p ./tsconfig.json -t 60000 tests/03__**/200__*.ts