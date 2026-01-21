"use client";

import { useEffect, useRef, useState } from "react";
import type { APIEvents } from "../../../ws-client/events";
import { useAutoSubscribe } from "@/contexts/auto-subscribe-context";
import { useMaxResults } from "@/contexts/max-results-context";
import { useWebSocket } from "@/contexts/websocket-context";
import { CodeWrapperClient } from "./code-wrapper-client";

type EventSubscriberProps = {
    eventName: keyof APIEvents;
};

type EventEntry = {
    timestamp: number;
    data: unknown;
};

export function EventSubscriber({ eventName }: EventSubscriberProps) {
    const { client, connected, error } = useWebSocket();
    const { autoSubscribe, settled: autoSubSettled } = useAutoSubscribe();
    const { maxResults, settled: maxResSettled } = useMaxResults();
    const [events, setEvents] = useState<EventEntry[]>([]);
    const [isSubscribed, setIsSubscribed] = useState(false);
    const maxResultsRef = useRef(maxResults);
    const eventStr = String(eventName);

    // Keep ref in sync with maxResults
    useEffect(() => {
        maxResultsRef.current = maxResults;
    }, [maxResults]);

    // Trim events array when maxResults changes
    useEffect(() => {
        if (!maxResSettled) return;
        setEvents((prev) => prev.slice(0, maxResults));
    }, [maxResults, maxResSettled]);

    useEffect(() => {
        if (!autoSubSettled || !client || !connected || !autoSubscribe) return;

        client
            .subscribe(eventName)
            .then((res) => {
                if (!res.error) {
                    setIsSubscribed(true);
                } else {
                    console.error(`[${eventStr}] Failed to subscribe:`, res.error);
                }
            })
            .catch((err) => {
                console.error(`[${eventStr}] Network or client error while subscribing:`, err);
            });

        // Subscribe to the event - use ref to access current maxResults
        const handleEvent = (data: unknown) => {
            setEvents((prev) => {
                const newEvents = [{ timestamp: Date.now(), data }, ...prev];
                return newEvents.slice(0, maxResultsRef.current);
            });
        };

        client.on(eventName, handleEvent);

        // Cleanup: unsubscribe when event name changes or unmount
        return () => {
            client.off(eventName, handleEvent);
            setEvents([]);
            setIsSubscribed(false);
            if (connected) {
                client
                    .unsubscribe(eventName)
                    .then((res) => {
                        if (!res.error) return;
                        console.error(`[${eventStr}] Failed to unsubscribe:`, res.error);
                    })
                    .catch((err) => {
                        console.error(`[${eventStr}] Promise rejection while unsubscribing:`, err);
                    });
            }
        };
    }, [client, connected, eventName, autoSubscribe, autoSubSettled, eventStr]);

    const handleClear = () => {
        setEvents([]);
    };

    return (
        <div className='mt-6 p-4 border border-border-primary rounded-lg'>
            <div className='flex items-center justify-between mb-4'>
                <div className='flex items-center gap-3'>
                    <h3 className='text-lg font-semibold text-text-primary'>Event History (Max {maxResults})</h3>
                    {events.length > 0 && (
                        <span className='text-sm text-text-tertiary'>
                            ({events.length} event
                            {events.length !== 1 ? "s" : ""})
                        </span>
                    )}
                    {isSubscribed && (
                        <div className='flex items-center gap-1.5'>
                            <div className='w-2 h-2 rounded-full bg-green-500 animate-pulse' />
                            <span className='text-xs text-text-tertiary'>Subscribed</span>
                        </div>
                    )}
                </div>
                {events.length > 0 && (
                    <button
                        onClick={handleClear}
                        className='px-3 py-1 text-sm rounded bg-bg-secondary hover:bg-bg-tertiary border border-border-primary transition-colors'
                        type='button'>
                        Clear
                    </button>
                )}
            </div>

            {error && (
                <div className='mb-4 p-3 bg-red-500/10 border border-red-500 rounded text-red-500 text-sm'>{error}</div>
            )}

            <div className='space-y-4'>
                {events.length > 0
                    ? events.map((event, index) => (
                          <div
                              key={event.timestamp}
                              className='pb-4 border-b border-border-primary last:border-b-0 last:pb-0 animate-slide-down-fade-in'>
                              <div className='flex items-center justify-between mb-2'>
                                  <span className='text-xs text-text-tertiary'>Event #{events.length - index}</span>
                                  <span className='text-xs text-text-tertiary'>
                                      {new Date(event.timestamp).toLocaleTimeString()}
                                  </span>
                              </div>
                              <CodeWrapperClient
                                  code={JSON.stringify(event.data)}
                                  lang='json'
                              />
                          </div>
                      ))
                    : !error && (
                          <div className='p-4 rounded-xl border-2 border-border-primary bg-background-secondary'>
                              <p className='text-text-tertiary text-sm'>
                                  {autoSubscribe
                                      ? `Waiting for event "${eventStr}"...`
                                      : "Auto-subscribe is disabled. Enable it in the sidebar to receive events."}
                              </p>
                          </div>
                      )}
            </div>
        </div>
    );
}
