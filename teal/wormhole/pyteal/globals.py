#!/usr/bin/python3

# The number of signatures verified by each transaction in the group.
# Since the last transaction of the group is the VAA processing one,
# the total count of required transactions to verify all guardian signatures is
#
# floor(guardian_count  / SIGNATURES_PER_TRANSACTION)
#
SIGNATURES_PER_VERIFICATION_STEP = 6