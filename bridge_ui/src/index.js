import { CssBaseline } from "@material-ui/core";
import { ThemeProvider } from "@material-ui/core/styles";
import ReactDOM from "react-dom";
import { Provider } from "react-redux";
import App from "./App";
import { store } from "./store";
import { EthereumProviderProvider } from "./contexts/EthereumProviderContext";
import { SolanaWalletProvider } from "./contexts/SolanaWalletContext.tsx";
import { theme } from "./muiTheme";

ReactDOM.render(
  <Provider store={store}>
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <SolanaWalletProvider>
        <EthereumProviderProvider>
          <App />
        </EthereumProviderProvider>
      </SolanaWalletProvider>
    </ThemeProvider>
  </Provider>,
  document.getElementById("root")
);
