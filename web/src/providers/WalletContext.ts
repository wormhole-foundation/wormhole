import React from 'react'
import Wallet from '@project-serum/sol-wallet-adapter'
import {SOLANA_HOST} from "../config";

const WalletContext = React.createContext<Wallet>(new Wallet("https://www.sollet.io", SOLANA_HOST));
export default WalletContext
