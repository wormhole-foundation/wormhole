# nslookup eth-devnet
#echo 'all tests hit'
set -e
npm --prefix ../sdk/js run test-ci
npm --prefix ../spydk/js run test-ci