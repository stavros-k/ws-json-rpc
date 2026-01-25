"use client";

import { useState } from "react";
import { getKindDisplayName } from "./type-kind-badge";

export type TypeFilterState = {
    kind: string;
    hideDeprecated: boolean;
    search: string;
};

type TypeFiltersProps = {
    kinds: string[];
    onFilterChange: (filter: TypeFilterState) => void;
};

export function TypeFilters({ kinds, onFilterChange }: TypeFiltersProps) {
    const [selectedKind, setSelectedKind] = useState<string>("all");
    const [hideDeprecated, setHideDeprecated] = useState(false);
    const [search, setSearch] = useState("");

    const handleFilterChange = (updates: Partial<TypeFilterState>) => {
        const newFilter = {
            kind: updates.kind !== undefined ? updates.kind : selectedKind,
            hideDeprecated: updates.hideDeprecated !== undefined ? updates.hideDeprecated : hideDeprecated,
            search: updates.search !== undefined ? updates.search : search,
        };
        onFilterChange(newFilter);
    };

    const handleKindChange = (kind: string) => {
        setSelectedKind(kind);
        handleFilterChange({ kind });
    };

    const handleDeprecatedToggle = () => {
        const newValue = !hideDeprecated;
        setHideDeprecated(newValue);
        handleFilterChange({ hideDeprecated: newValue });
    };

    const handleSearchChange = (newSearch: string) => {
        setSearch(newSearch);
        handleFilterChange({ search: newSearch });
    };

    return (
        <div className='flex flex-col gap-4 mb-8 p-4 bg-bg-secondary rounded-xl border-2 border-border-primary'>
            <div className='flex flex-col sm:flex-row sm:items-center gap-4 flex-wrap'>
                {/* Kind Filter */}
                <div className='flex items-center gap-3'>
                    <span className='text-sm text-text-primary font-bold'>Kind:</span>
                    <div className='flex gap-2 flex-wrap'>
                        <button
                            type='button'
                            onClick={() => handleKindChange("all")}
                            className={`px-3 py-1.5 rounded-lg text-sm font-bold transition-all duration-200 ${
                                selectedKind === "all"
                                    ? "bg-accent-blue text-text-primary shadow-md"
                                    : "bg-bg-tertiary text-text-secondary hover:bg-bg-hover border-2 border-border-primary hover:border-accent-blue"
                            }`}>
                            All
                        </button>
                        {kinds.map((kind) => (
                            <button
                                key={kind}
                                type='button'
                                onClick={() => handleKindChange(kind)}
                                className={`px-3 py-1.5 rounded-lg text-sm font-bold transition-all duration-200 ${
                                    selectedKind === kind
                                        ? "bg-accent-blue text-text-primary shadow-md"
                                        : "bg-bg-tertiary text-text-secondary hover:bg-bg-hover border-2 border-border-primary hover:border-accent-blue"
                                }`}>
                                {getKindDisplayName(kind)}
                            </button>
                        ))}
                    </div>
                </div>

                {/* Hide Deprecated */}
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

            {/* Search */}
            <div className='flex items-center gap-3'>
                <span className='text-sm text-text-primary font-bold'>Search:</span>
                <input
                    type='text'
                    value={search}
                    onChange={(e) => handleSearchChange(e.target.value)}
                    placeholder='Search by type name...'
                    className='flex-1 px-3 py-2 rounded-lg text-sm bg-bg-tertiary text-text-primary border-2 border-border-primary hover:border-accent-blue focus:border-accent-blue focus:ring-2 focus:ring-accent-blue placeholder-text-muted'
                />
            </div>
        </div>
    );
}
