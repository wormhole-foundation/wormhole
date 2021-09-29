import React, {
  ReactChildren,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";
import { useSnackbar } from "notistack";

interface LoggerContext {
  log: (value: string, type?: "error" | "info" | "success" | undefined) => void;
  clear: () => void;
  logs: string[];
}

const LoggerProviderContext = React.createContext<LoggerContext>({
  log: (value: string, type?: "error" | "info" | "success" | undefined) => {},
  clear: () => {},
  logs: [],
});

export const LoggerProvider = ({ children }: { children: ReactChildren }) => {
  const [logs, setLogs] = useState<string[]>(["Instantiated the logger."]);
  const clear = useCallback(() => setLogs([]), [setLogs]);
  const { enqueueSnackbar } = useSnackbar();

  const log = useCallback(
    (value: string, type?: "error" | "info" | "success" | undefined) => {
      setLogs((logs: any) => [...logs, value]);
      if (type === "error") {
        console.error(value);
        enqueueSnackbar(value, { variant: "error" });
      } else if (type === "success") {
        console.log(value);
        enqueueSnackbar(value, { variant: "success" });
      } else if (type === "info") {
        console.log(value);
        enqueueSnackbar(value, { variant: "info" });
      } else {
        console.log(value);
      }
    },
    [setLogs, enqueueSnackbar]
  );

  const contextValue = useMemo(
    () => ({
      logs,
      clear,
      log,
    }),
    [logs, clear, log]
  );
  return (
    <LoggerProviderContext.Provider value={contextValue}>
      {children}
    </LoggerProviderContext.Provider>
  );
};
export const useLogger = () => {
  return useContext(LoggerProviderContext);
};
