import { CssBaseline } from "@material-ui/core";
import { ThemeProvider } from "@material-ui/core/styles";
import ReactDOM from "react-dom";
import App from "./App";
import { LoggerProvider } from "./contexts/Logger";
import { SolanaWalletProvider } from "./contexts/SolanaWalletContext";
import { theme } from "./muiTheme";
ReactDOM.render(
  <ThemeProvider theme={theme}>
    <CssBaseline />
    <SolanaWalletProvider>
      <LoggerProvider>
        <App />
      </LoggerProvider>
    </SolanaWalletProvider>
  </ThemeProvider>,
  document.getElementById("root")
);
