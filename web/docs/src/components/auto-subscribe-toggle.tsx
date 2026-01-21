"use client";

import { useAutoSubscribe } from "@/contexts/auto-subscribe-context";

export function AutoSubscribeToggle() {
    const { autoSubscribe, toggleAutoSubscribe, settled } = useAutoSubscribe();

    if (!settled) {
        return (
            <div className='w-full rounded-lg border border-border-primary bg-bg-secondary px-3 py-2 text-sm font-medium opacity-50'>
                <div className='flex items-center justify-between'>
                    <span>Auto-subscribe Events</span>
                    <span className='text-xs'>...</span>
                </div>
            </div>
        );
    }

    return (
        <button
            onClick={toggleAutoSubscribe}
            className={`w-full rounded-lg border transition-all duration-200 px-3 py-2 text-sm font-medium ${
                autoSubscribe
                    ? "bg-green-500/10 border-green-500 text-green-500"
                    : "bg-bg-secondary border-border-primary text-text-primary"
            }`}
            type='button'>
            <div className='flex items-center justify-between'>
                <span>Auto-subscribe Events</span>
                <span className='text-xs'>{autoSubscribe ? "ON" : "OFF"}</span>
            </div>
        </button>
    );
}
