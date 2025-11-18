"use client";

import { useWebSocket } from "@/contexts/websocket-context";

export function ConnectionIndicator() {
    const { connected } = useWebSocket();

    return (
        <div
            className={`w-3 h-3 rounded-full ${
                connected ? "bg-green-500 animate-pulse" : "bg-red-500"
            }`}
            title={connected ? "WebSocket Connected" : "WebSocket Disconnected"}
        />
    );
}
