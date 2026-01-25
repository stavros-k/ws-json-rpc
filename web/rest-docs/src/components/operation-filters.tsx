"use client";

import { useState } from "react";

export type OperationFilterState = {
    verb: string;
    group: string;
    hideDeprecated: boolean;
    search: string;
};

type OperationFiltersProps = {
    verbs: string[];
    groups: string[];
    onFilterChange: (filter: OperationFilterState) => void;
};

export function OperationFilters({ verbs, groups, onFilterChange }: OperationFiltersProps) {
    const [selectedVerb, setSelectedVerb] = useState<string>("all");
    const [selectedGroup, setSelectedGroup] = useState<string>("all");
    const [hideDeprecated, setHideDeprecated] = useState(false);
    const [search, setSearch] = useState("");

    const handleFilterChange = (updates: Partial<OperationFilterState>) => {
        const newFilter = {
            verb: updates.verb !== undefined ? updates.verb : selectedVerb,
            group: updates.group !== undefined ? updates.group : selectedGroup,
            hideDeprecated: updates.hideDeprecated !== undefined ? updates.hideDeprecated : hideDeprecated,
            search: updates.search !== undefined ? updates.search : search,
        };
        onFilterChange(newFilter);
    };

    const handleVerbChange = (verb: string) => {
        setSelectedVerb(verb);
        handleFilterChange({ verb });
    };

    const handleGroupChange = (group: string) => {
        setSelectedGroup(group);
        handleFilterChange({ group });
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
                {/* Verb Filter */}
                <div className='flex items-center gap-3'>
                    <span className='text-sm text-text-primary font-bold'>Method:</span>
                    <div className='flex gap-2 flex-wrap'>
                        <button
                            type='button'
                            onClick={() => handleVerbChange("all")}
                            className={`px-3 py-1.5 rounded-lg text-sm font-bold transition-all duration-200 ${
                                selectedVerb === "all"
                                    ? "bg-accent-blue text-text-primary shadow-md"
                                    : "bg-bg-tertiary text-text-secondary hover:bg-bg-hover border-2 border-border-primary hover:border-accent-blue"
                            }`}>
                            All
                        </button>
                        {verbs.map((verb) => (
                            <button
                                key={verb}
                                type='button'
                                onClick={() => handleVerbChange(verb)}
                                className={`px-3 py-1.5 rounded-lg text-sm font-bold transition-all duration-200 ${
                                    selectedVerb === verb
                                        ? "bg-accent-blue text-text-primary shadow-md"
                                        : "bg-bg-tertiary text-text-secondary hover:bg-bg-hover border-2 border-border-primary hover:border-accent-blue"
                                }`}>
                                {verb}
                            </button>
                        ))}
                    </div>
                </div>

                {/* Group Filter */}
                {groups.length > 0 && (
                    <div className='flex items-center gap-3'>
                        <span className='text-sm text-text-primary font-bold'>Group:</span>
                        <select
                            value={selectedGroup}
                            onChange={(e) => handleGroupChange(e.target.value)}
                            className='px-3 py-1.5 rounded-lg text-sm font-bold bg-bg-tertiary text-text-primary border-2 border-border-primary hover:border-accent-blue focus:border-accent-blue focus:ring-2 focus:ring-accent-blue cursor-pointer'>
                            <option value='all'>All</option>
                            {groups.map((group) => (
                                <option
                                    key={group}
                                    value={group}>
                                    {group}
                                </option>
                            ))}
                        </select>
                    </div>
                )}

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
                    placeholder='Search by operation name...'
                    className='flex-1 px-3 py-2 rounded-lg text-sm bg-bg-tertiary text-text-primary border-2 border-border-primary hover:border-accent-blue focus:border-accent-blue focus:ring-2 focus:ring-accent-blue placeholder-text-muted'
                />
            </div>
        </div>
    );
}
