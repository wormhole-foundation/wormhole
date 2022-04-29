import MyAlgoConnect, { Accounts } from "@randlabs/myalgo-connect";
import {
  createContext,
  ReactChildren,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";

interface IAlgorandContext {
  connect(): void;
  disconnect(): void;
  accounts: Accounts[];
}

const AlgorandContext = createContext<IAlgorandContext>({
  connect: () => {},
  disconnect: () => {},
  accounts: [],
});

export const AlgorandContextProvider = ({
  children,
}: {
  children: ReactChildren;
}) => {
  const myAlgoConnect = useMemo(() => new MyAlgoConnect(), []);
  const [accounts, setAccounts] = useState<Accounts[]>([]);
  const connect = useCallback(() => {
    let cancelled = false;
    (async () => {
      const accounts = await myAlgoConnect.connect({
        shouldSelectOneAccount: true,
      });
      if (!cancelled) {
        setAccounts(accounts);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [myAlgoConnect]);
  const disconnect = useCallback(() => {
    setAccounts([]);
  }, []);
  const value = useMemo(
    () => ({
      connect,
      disconnect,
      accounts,
    }),
    [connect, disconnect, accounts]
  );

  return (
    <AlgorandContext.Provider value={value}>
      {children}
    </AlgorandContext.Provider>
  );
};

export const useAlgorandContext = () => {
  return useContext(AlgorandContext);
};
