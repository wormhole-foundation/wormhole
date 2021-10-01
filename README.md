# PricecasterService

This service consumes prices from Pyth and feeds a TEAL program with messages containing signed price data. The program code validates signature and message validity and authenticity and subsequently stores the price information in the global application information for other contracts to retrieve.

