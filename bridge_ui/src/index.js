import { CssBaseline } from "@material-ui/core";
import { ThemeProvider } from "@material-ui/core/styles";
import ReactDOM from "react-dom";
import { Provider } from "react-redux";
import { HashRouter } from "react-router-dom";
import App from "./App";
import RadialGradient from "./components/RadialGradient";
import { EthereumProviderProvider } from "./contexts/EthereumProviderContext";
import { SolanaWalletProvider } from "./contexts/SolanaWalletContext.tsx";
import { TerraWalletProvider } from "./contexts/TerraWalletContext.tsx";
import { theme } from "./muiTheme";
import { store } from "./store";

ReactDOM.render(
  <Provider store={store}>
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <RadialGradient />
      <SolanaWalletProvider>
        <EthereumProviderProvider>
          <TerraWalletProvider>
            <HashRouter>
              <App />
            </HashRouter>
          </TerraWalletProvider>
        </EthereumProviderProvider>
      </SolanaWalletProvider>
    </ThemeProvider>
  </Provider>,
  document.getElementById("root")
);
