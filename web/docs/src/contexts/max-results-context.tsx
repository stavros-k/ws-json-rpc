"use client";

import type { ReactNode } from "react";
import { createContext, useContext, useEffect, useState } from "react";

type MaxResultsContextValue = {
    maxResults: number;
    setMaxResults: (max: number) => void;
    settled: boolean;
};

const MaxResultsContext = createContext<MaxResultsContextValue | null>(null);

type MaxResultsProviderProps = {
    children: ReactNode;
};

const STORAGE_KEY = "maxResults";

export function MaxResultsProvider({ children }: MaxResultsProviderProps) {
    const [maxResults, setMaxResults] = useState<number>(10);
    const [settled, setSettled] = useState(false);

    useEffect(() => {
        const saved = localStorage.getItem(STORAGE_KEY);
        if (saved !== null) {
            const value = Number.parseInt(saved, 10);
            setMaxResults(!Number.isNaN(value) && value >= 1 && value <= 100 ? value : 10);
        }
        setSettled(true);
    }, []);

    useEffect(() => {
        if (settled) localStorage.setItem(STORAGE_KEY, String(maxResults));
    }, [maxResults, settled]);

    return (
        <MaxResultsContext.Provider
            value={{
                maxResults,
                setMaxResults,
                settled,
            }}>
            {children}
        </MaxResultsContext.Provider>
    );
}

export function useMaxResults() {
    const context = useContext(MaxResultsContext);
    if (!context) {
        throw new Error("useMaxResults must be used within MaxResultsProvider");
    }
    return context;
}
