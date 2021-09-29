import { CssBaseline } from "@material-ui/core";
import { ThemeProvider } from "@material-ui/core/styles";
import ReactDOM from "react-dom";
import App from "./App";
import ErrorBoundary from "./components/ErrorBoundary";
import { LoggerProvider } from "./contexts/Logger";
import { SolanaWalletProvider } from "./contexts/SolanaWalletContext";
import { theme } from "./muiTheme";
import { SnackbarProvider } from "notistack";
import { EthereumProviderProvider } from "./contexts/EthereumProviderContext";
ReactDOM.render(
  <ErrorBoundary>
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <SolanaWalletProvider>
        <EthereumProviderProvider>
          <SnackbarProvider maxSnack={3}>
            <LoggerProvider>
              <App />
            </LoggerProvider>
          </SnackbarProvider>
        </EthereumProviderProvider>
      </SolanaWalletProvider>
    </ThemeProvider>
  </ErrorBoundary>,
  document.getElementById("root")
);
