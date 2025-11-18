"use client";

import type { ReactNode } from "react";
import { createContext, useContext, useState } from "react";

type AutoSubscribeContextValue = {
    autoSubscribe: boolean;
    toggleAutoSubscribe: () => void;
};

const AutoSubscribeContext = createContext<AutoSubscribeContextValue | null>(
    null
);

type AutoSubscribeProviderProps = {
    children: ReactNode;
};

export function AutoSubscribeProvider({
    children,
}: AutoSubscribeProviderProps) {
    const [autoSubscribe, setAutoSubscribe] = useState(true);

    const toggleAutoSubscribe = () => {
        setAutoSubscribe((prev) => !prev);
    };

    return (
        <AutoSubscribeContext.Provider
            value={{
                autoSubscribe,
                toggleAutoSubscribe,
            }}>
            {children}
        </AutoSubscribeContext.Provider>
    );
}

export function useAutoSubscribe() {
    const context = useContext(AutoSubscribeContext);
    if (!context) {
        throw new Error(
            "useAutoSubscribe must be used within AutoSubscribeProvider"
        );
    }
    return context;
}
