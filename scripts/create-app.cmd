goal clerk compile teal\pricedata.teal.tmpl -o teal\pricedata.teal.bin
goal app create --out create-app.txn --creator  OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU --local-byteslices 0 --local-ints 0 --global-byteslices 3 --global-ints 2 --approval-prog-raw teal\pricedata.teal.bin --clear-prog teal\clearstate.teal
algokey sign -t create-app.txn -m "assault approve result rare float sugar power float soul kind galaxy edit unusual pretty tone tilt net range pelican avoid unhappy amused recycle abstract master" -o create-app.stxn
