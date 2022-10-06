#
curl -X POST -d '{"FixedAmountRequest":{"recipient": "'"$1"'"}}' -H 'Content-Type: application/json' http://127.0.0.1:5003/gas