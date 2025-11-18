"use client";

import type { ReactNode } from "react";
import { createContext, useContext, useState } from "react";

type MaxResultsContextValue = {
    maxResults: number;
    setMaxResults: (max: number) => void;
};

const MaxResultsContext = createContext<MaxResultsContextValue | null>(null);

type MaxResultsProviderProps = {
    children: ReactNode;
};

export function MaxResultsProvider({ children }: MaxResultsProviderProps) {
    const [maxResults, setMaxResults] = useState(10);

    return (
        <MaxResultsContext.Provider
            value={{
                maxResults,
                setMaxResults,
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
