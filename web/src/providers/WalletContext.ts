import React from 'react'
import Wallet from '@project-serum/sol-wallet-adapter'

const WalletContext = React.createContext<Wallet | undefined>(undefined);
export default WalletContext
