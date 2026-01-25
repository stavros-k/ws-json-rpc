"use client";

import { useState } from "react";

type ProtocolType = "all" | "http" | "ws";

type FilterState = {
    protocol: ProtocolType;
    hideDeprecated: boolean;
};

type FilterControlsProps = {
    onFilterChange: (filter: FilterState) => void;
};

export function FilterControls({ onFilterChange }: FilterControlsProps) {
    const [selected, setSelected] = useState<ProtocolType>("all");
    const [hideDeprecated, setHideDeprecated] = useState(false);

    const handleProtocolChange = (filter: ProtocolType) => {
        setSelected(filter);
        onFilterChange({ protocol: filter, hideDeprecated });
    };

    const handleDeprecatedToggle = () => {
        const newValue = !hideDeprecated;
        setHideDeprecated(newValue);
        onFilterChange({ protocol: selected, hideDeprecated: newValue });
    };

    const filterOptions: { value: ProtocolType; label: string }[] = [
        { value: "all", label: "All" },
        { value: "http", label: "HTTP" },
        { value: "ws", label: "WebSocket" },
    ];

    return (
        <div className='flex flex-col sm:flex-row sm:items-center gap-6 mb-8 p-4 bg-bg-secondary rounded-xl border-2 border-border-primary'>
            <div className='flex items-center gap-3'>
                <span className='text-sm text-text-primary font-bold'>Protocol:</span>
                <div className='flex gap-2'>
                    {filterOptions.map((option) => (
                        <button
                            key={option.value}
                            type='button'
                            onClick={() => handleProtocolChange(option.value)}
                            className={`px-4 py-2 rounded-lg text-sm font-bold transition-all duration-200 ${
                                selected === option.value
                                    ? "bg-bg-tertiary text-text-primary shadow-md border-2 border-accent-blue-border"
                                    : "bg-bg-secondary text-text-primary border-2 border-border-primary hover:border-border-secondary"
                            }`}>
                            {option.label}
                        </button>
                    ))}
                </div>
            </div>

            <label className='flex items-center gap-2.5 cursor-pointer group'>
                <input
                    type='checkbox'
                    checked={hideDeprecated}
                    onChange={handleDeprecatedToggle}
                    className='w-5 h-5 rounded-md border-2 border-border-primary bg-bg-tertiary text-accent-blue focus:ring-accent-blue focus:ring-2 cursor-pointer'
                />
                <span className='text-sm text-text-primary font-bold group-hover:text-accent-blue transition-colors'>
                    Hide deprecated
                </span>
            </label>
        </div>
    );
}

export type { ProtocolType, FilterState };
