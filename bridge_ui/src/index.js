import { CssBaseline } from "@material-ui/core";
import { ThemeProvider } from "@material-ui/core/styles";
import ReactDOM from "react-dom";
import App from "./App";
import { EthereumProviderProvider } from "./contexts/EthereumProviderContext";
import { SolanaWalletProvider } from "./contexts/SolanaWalletContext.tsx";
import { theme } from "./muiTheme";

ReactDOM.render(
  <ThemeProvider theme={theme}>
    <CssBaseline />
    <SolanaWalletProvider>
      <EthereumProviderProvider>
        <App />
      </EthereumProviderProvider>
    </SolanaWalletProvider>
  </ThemeProvider>,
  document.getElementById("root")
);
