all: vaa-processor-approval.teal vaa-processor-clear.teal vaa-verify.teal

PYTEAL=../../staging/algorand/teal/wormhole/pyteal

vaa-processor-approval.teal vaa-processor-clear.teal: ${PYTEAL}/vaa-processor.py
	python3 ${PYTEAL}/vaa-processor.py vaa-processor-approval.teal vaa-processor-clear.teal

vaa-verify.teal: ${PYTEAL}/vaa-verify.py
	python3 ${PYTEAL}/vaa-verify.py 0 vaa-verify.teal
