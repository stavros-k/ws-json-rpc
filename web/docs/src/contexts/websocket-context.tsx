"use client";

import type { ReactNode } from "react";
import { createContext, useContext, useEffect, useRef, useState } from "react";
import type { ApiEvents, ApiMethods } from "@/../../artifacts/types";
import { WebSocketClient } from "@/../../ws-client";

type WebSocketContextValue = {
    client: WebSocketClient<ApiMethods, ApiEvents> | null;
    connected: boolean;
    error: string | null;
};

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

type WebSocketProviderProps = {
    children: ReactNode;
};

function getHost() {
    if (process.env.NODE_ENV === "development") return "localhost:8080";
    if (typeof window !== "undefined") return window.location.host;

    return "localhost:8080";
}

export function WebSocketProvider({ children }: WebSocketProviderProps) {
    const [client, setClient] = useState<WebSocketClient<
        ApiMethods,
        ApiEvents
    > | null>(null);
    const [connected, setConnected] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const initializedRef = useRef(false);

    useEffect(() => {
        // Return early if client already initialized
        if (initializedRef.current) return;

        initializedRef.current = true;

        // Calculate WS_URL inside effect to prevent unnecessary re-renders
        const WS_URL = `ws://${getHost()}/ws`;

        // Initialize client
        const newClient = new WebSocketClient<ApiMethods, ApiEvents>({
            url: WS_URL,
            clientId: `docs-client-${Date.now()}`,
            reconnectDelay: 1000,
            maxReconnectAttempts: Number.POSITIVE_INFINITY,
        });

        newClient.onConnect(() => {
            setConnected(true);
            setError(null);
        });

        newClient.onDisconnect(() => {
            setConnected(false);
        });

        newClient.onError((err) => {
            setError(`WebSocket error: ${err.type}`);
            setConnected(false);
        });

        // Auto-connect
        newClient.connect().catch((err) => {
            setError(`Connection failed: ${err.message}`);
        });

        setClient(newClient);

        return () => {
            newClient.disconnect();
            setClient(null);
            initializedRef.current = false;
        };
    }, []);

    return (
        <WebSocketContext.Provider
            value={{
                client,
                connected,
                error,
            }}>
            {children}
        </WebSocketContext.Provider>
    );
}

export function useWebSocket() {
    const context = useContext(WebSocketContext);
    if (!context) {
        throw new Error("useWebSocket must be used within WebSocketProvider");
    }
    return context;
}
