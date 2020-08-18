import React, {useEffect, useMemo, useState} from 'react';
import './App.css';
import * as solanaWeb3 from '@solana/web3.js';
import ClientContext from '../providers/ClientContext';
import Transfer from "../pages/Transfer";
import {Layout} from 'antd';
import {SolanaTokenProvider} from "../providers/SolanaTokenContext";
import {SlotProvider} from "../providers/SlotContext";
import {BrowserRouter as Router, Link, Route, Switch} from 'react-router-dom';
import TransferSolana from "../pages/TransferSolana";
import WalletContext from '../providers/WalletContext';
import Wallet from "@project-serum/sol-wallet-adapter";
import {BridgeProvider} from "../providers/BridgeContext";

const {Header, Content, Footer} = Layout;

function App() {
    let c = new solanaWeb3.Connection("http://localhost:8899");
    const wallet = useMemo(() => new Wallet("https://www.sollet.io", "http://localhost:8899"), []);
    const [, setConnected] = useState(false);
    useEffect(() => {
        wallet.on('connect', () => {
            setConnected(true);
            console.log('Connected to wallet ' + wallet.publicKey.toBase58());
        });
        wallet.on('disconnect', () => {
            setConnected(false);
            console.log('Disconnected from wallet');
        });
        return () => {
            wallet.disconnect();
        };
    }, [wallet]);

    return (
        <div className="App">
            <Layout style={{height: '100%'}}>
                <Router>
                    <Header style={{position: 'fixed', zIndex: 1, width: '100%'}}>
                        <Link to="/" style={{paddingRight: 20}}>Ethereum</Link>
                        <Link to="/solana">Solana</Link>
                        <div className="logo"/>
                    </Header>
                    <Content style={{padding: '0 50px', marginTop: 64}}>
                        <div style={{padding: 24}}>
                            <ClientContext.Provider value={c}>
                                <SlotProvider>
                                    <WalletContext.Provider value={wallet}>
                                        <BridgeProvider>
                                            <SolanaTokenProvider>

                                                <Switch>
                                                    <Route path="/solana">
                                                        <TransferSolana/>
                                                    </Route>
                                                    <Route path="/">
                                                        <Transfer/>
                                                    </Route>
                                                </Switch>
                                            </SolanaTokenProvider>
                                        </BridgeProvider>
                                    </WalletContext.Provider>
                                </SlotProvider>
                            </ClientContext.Provider>
                        </div>
                    </Content>
                    <Footer style={{textAlign: 'center'}}>nexantic GmbH 2020</Footer>
                </Router>
            </Layout>
        </div>
    );
}

export default App;
