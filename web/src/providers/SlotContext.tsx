import React, {createContext, FunctionComponent, useContext, useEffect, useState} from "react"
import ClientContext from "../providers/ClientContext";
import solanaWeb3 from "@solana/web3.js";

export const SlotContext = createContext(0)

export const SlotProvider: FunctionComponent = ({children}) => {
    let c = useContext<solanaWeb3.Connection>(ClientContext);

    let [slot, setSlot] = useState(0);
    useEffect(() => {
        c.onSlotChange(value => {
            setSlot(value.slot);
        });
    })

    return (
        <SlotContext.Provider value={slot}>
            {children}
        </SlotContext.Provider>
    )
}
