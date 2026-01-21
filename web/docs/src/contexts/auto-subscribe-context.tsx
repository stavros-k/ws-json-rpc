"use client";

import type { ReactNode } from "react";
import { createContext, useContext, useEffect, useState } from "react";

type AutoSubscribeContextValue = {
    autoSubscribe: boolean;
    toggleAutoSubscribe: () => void;
    settled: boolean;
};

const AutoSubscribeContext = createContext<AutoSubscribeContextValue | null>(null);

type AutoSubscribeProviderProps = {
    children: ReactNode;
};

const STORAGE_KEY = "autoSubscribe";

export function AutoSubscribeProvider({ children }: AutoSubscribeProviderProps) {
    const [autoSubscribe, setAutoSubscribe] = useState<boolean>(true);
    const [settled, setSettled] = useState(false);

    useEffect(() => {
        const saved = localStorage.getItem(STORAGE_KEY);
        setAutoSubscribe(saved !== null ? saved === "true" : true);
        setSettled(true);
    }, []);

    useEffect(() => {
        if (settled) localStorage.setItem(STORAGE_KEY, String(autoSubscribe));
    }, [autoSubscribe, settled]);

    const toggleAutoSubscribe = () => {
        setAutoSubscribe((prev) => !prev);
    };

    return (
        <AutoSubscribeContext.Provider
            value={{
                autoSubscribe,
                toggleAutoSubscribe,
                settled,
            }}>
            {children}
        </AutoSubscribeContext.Provider>
    );
}

export function useAutoSubscribe() {
    const context = useContext(AutoSubscribeContext);
    if (!context) {
        throw new Error("useAutoSubscribe must be used within AutoSubscribeProvider");
    }
    return context;
}
