import React from 'react';
import './App.css';
import * as solanaWeb3 from '@solana/web3.js';
import ClientContext from '../providers/ClientContext';
import Transfer from "../pages/Transfer";
import {Layout} from 'antd';
import {SolanaTokenProvider} from "../providers/SolanaTokenContext";
import {SlotProvider} from "../providers/SlotContext";
import {BrowserRouter as Router, Link, Route, Switch} from 'react-router-dom';
import TransferSolana from "../pages/TransferSolana";

const {Header, Content, Footer} = Layout;

function App() {
    let c = new solanaWeb3.Connection("http://localhost:8899");
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
