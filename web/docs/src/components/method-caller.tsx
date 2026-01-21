"use client";

import { useEffect, useRef, useState } from "react";
import type { APIMethods } from "../../../ws-client/methods";
import { useMaxResults } from "@/contexts/max-results-context";
import { useWebSocket } from "@/contexts/websocket-context";
import { CodeWrapperClient } from "./code-wrapper-client";

type MethodCallerProps = {
    methodName: keyof APIMethods;
    defaultParams?: string;
};

type CallResult = {
    timestamp: number;
    success: boolean;
    data: unknown;
    error?: string;
};

export function MethodCaller({ methodName, defaultParams = "{}" }: MethodCallerProps) {
    const { client, connected, error: connectionError } = useWebSocket();
    const { maxResults, settled: maxResSettled } = useMaxResults();
    const [params, setParams] = useState(defaultParams);
    const [results, setResults] = useState<CallResult[]>([]);
    const [isValidJson, setIsValidJson] = useState(true);
    const [isCalling, setIsCalling] = useState(false);
    const maxResultsRef = useRef(maxResults);

    // Check if method has parameters (defaultParams is null or empty object)
    const hasParams = defaultParams !== null && defaultParams !== "{}";

    // Keep ref in sync with maxResults
    useEffect(() => {
        maxResultsRef.current = maxResults;
    }, [maxResults]);

    // Trim results array when maxResults changes
    useEffect(() => {
        if (!maxResSettled) return;
        setResults((prev) => prev.slice(0, maxResults));
    }, [maxResults, maxResSettled]);

    const validateJson = (value: string) => {
        if (!value.trim()) {
            setIsValidJson(true);
            return true;
        }
        try {
            JSON.parse(value);
            setIsValidJson(true);
            return true;
        } catch {
            setIsValidJson(false);
            return false;
        }
    };

    const handleParamsChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
        const value = e.target.value;
        setParams(value);
        validateJson(value);
    };

    const handleCall = async () => {
        if (!client || !connected) {
            setResults((prev) => [
                ...prev,
                {
                    timestamp: Date.now(),
                    success: false,
                    data: null,
                    error: "Not connected to WebSocket",
                },
            ]);
            return;
        }

        if (!isValidJson) {
            return;
        }

        setIsCalling(true);

        try {
            let parsedParams: unknown;
            try {
                parsedParams = params.trim() ? JSON.parse(params) : undefined;
            } catch (parseError) {
                setResults((prev) => [
                    ...prev,
                    {
                        timestamp: Date.now(),
                        success: false,
                        data: null,
                        error: `Invalid JSON: ${parseError instanceof Error ? parseError.message : String(parseError)}`,
                    },
                ]);
                setIsCalling(false);
                return;
            }

            // @ts-expect-error: Dynamic method call
            const response = await client.call(methodName, parsedParams);

            if (response.error) {
                setResults((prev) => {
                    const newResults = [
                        {
                            timestamp: Date.now(),
                            success: false,
                            data: response.result,
                            error: `RPC Error: ${response.error.message} (Code: ${response.error.code})`,
                        },
                        ...prev,
                    ];
                    return newResults.slice(0, maxResultsRef.current);
                });
            } else {
                setResults((prev) => {
                    const newResults = [
                        {
                            timestamp: Date.now(),
                            success: true,
                            data: response.result,
                        },
                        ...prev,
                    ];
                    return newResults.slice(0, maxResultsRef.current);
                });
            }
        } catch (err) {
            setResults((prev) => [
                {
                    timestamp: Date.now(),
                    success: false,
                    data: null,
                    error: `Call failed: ${err instanceof Error ? err.message : String(err)}`,
                },
                ...prev,
            ]);
        } finally {
            setIsCalling(false);
        }
    };

    const handleClear = () => {
        setResults([]);
    };

    return (
        <div className='mt-6 p-4 border border-border-primary rounded-lg'>
            <div className='flex items-center justify-between mb-4'>
                <h3 className='text-lg font-semibold text-text-primary'>Call Method</h3>
                {connected && (
                    <div className='flex items-center gap-1.5'>
                        <div className='w-2 h-2 rounded-full bg-green-500' />
                        <span className='text-xs text-text-tertiary'>Connected</span>
                    </div>
                )}
            </div>

            {connectionError && (
                <div className='mb-4 p-3 bg-red-500/10 border border-red-500 rounded text-red-500 text-sm'>
                    {connectionError}
                </div>
            )}

            {!connectionError && (
                <>
                    {hasParams && (
                        <div className='space-y-2 mb-4'>
                            <label
                                htmlFor={`method-params-${methodName}`}
                                className='block text-sm font-medium text-text-secondary'>
                                Parameters (JSON)
                            </label>
                            <div className='relative'>
                                <div
                                    className='invisible whitespace-pre-wrap wrap-break-word px-3 py-2 font-mono text-sm min-h-10'
                                    aria-hidden='true'>
                                    {params || "{}"}
                                </div>
                                <textarea
                                    id={`method-params-${methodName}`}
                                    value={params}
                                    onChange={handleParamsChange}
                                    className={`absolute inset-0 w-full h-full px-3 py-2 font-mono text-sm rounded border ${
                                        isValidJson
                                            ? "border-border-primary bg-bg-secondary"
                                            : "border-red-500 bg-red-500/5"
                                    } text-text-primary focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none overflow-hidden min-h-10`}
                                    placeholder='{"key": "value"}'
                                />
                            </div>
                            {!isValidJson && <p className='text-xs text-red-500'>Invalid JSON syntax</p>}
                        </div>
                    )}

                    <button
                        onClick={handleCall}
                        disabled={!connected || !isValidJson || isCalling}
                        className='w-full px-4 py-2 rounded bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-medium transition-colors mb-4'
                        type='button'>
                        {isCalling ? "Calling..." : "Call"}
                    </button>

                    <div className='border-t border-border-primary pt-4 space-y-4'>
                        {results.length > 0 ? (
                            results.map((result, index) => (
                                <div
                                    key={result.timestamp}
                                    className='pb-4 border-b border-border-primary last:border-b-0 last:pb-0 animate-slide-down-fade-in'>
                                    <div className='flex items-center justify-between mb-2'>
                                        <div className='flex items-center gap-2'>
                                            <span className='text-xs text-text-tertiary'>
                                                Result #{results.length - index}
                                            </span>
                                            <span
                                                className={`text-xs px-2 py-0.5 rounded ${
                                                    result.success
                                                        ? "bg-green-500/20 text-green-500"
                                                        : "bg-red-500/20 text-red-500"
                                                }`}>
                                                {result.success ? "Success" : "Error"}
                                            </span>
                                        </div>
                                        <span className='text-xs text-text-tertiary'>
                                            {new Date(result.timestamp).toLocaleTimeString()}
                                        </span>
                                    </div>
                                    {result.error && (
                                        <div className='mb-2 p-2 bg-red-500/10 border border-red-500 rounded text-red-500 text-xs'>
                                            {result.error}
                                        </div>
                                    )}
                                    {result.data !== null && result.data !== undefined && (
                                        <CodeWrapperClient
                                            code={JSON.stringify(result.data)}
                                            lang='json'
                                        />
                                    )}
                                </div>
                            ))
                        ) : (
                            <div className='p-4 rounded-xl border-2 border-border-primary bg-background-secondary'>
                                <p className='text-text-tertiary text-sm'>
                                    No results yet. Call the method to see results here.
                                </p>
                            </div>
                        )}
                    </div>
                </>
            )}

            {results.length > 0 && (
                <div className='mt-4 flex justify-end'>
                    <button
                        onClick={handleClear}
                        className='px-3 py-1 text-sm rounded bg-bg-secondary hover:bg-bg-tertiary border border-border-primary transition-colors'
                        type='button'>
                        Clear
                    </button>
                </div>
            )}
        </div>
    );
}
