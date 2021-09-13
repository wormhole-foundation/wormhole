import React, {
  ReactChildren,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";

interface LoggerContext {
  log: (value: string) => void;
  clear: () => void;
  logs: string[];
}

const LoggerProviderContext = React.createContext<LoggerContext>({
  log: (value: string) => {},
  clear: () => {},
  logs: [],
});

export const LoggerProvider = ({ children }: { children: ReactChildren }) => {
  const [logs, setLogs] = useState<string[]>([]);
  const clear = useCallback(() => setLogs([]), [setLogs]);
  const log = useCallback(
    (value: string) => {
      setLogs((logs) => [...logs, value]);
    },
    [setLogs]
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
