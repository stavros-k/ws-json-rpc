"use client";

import { useState } from "react";

export type MQTTFilterState = {
    group: string;
    hideDeprecated: boolean;
    search: string;
};

type MQTTFiltersProps = {
    groups: string[];
    onFilterChange: (filter: MQTTFilterState) => void;
};

export function MQTTFilters({ groups, onFilterChange }: MQTTFiltersProps) {
    const [selectedGroup, setSelectedGroup] = useState<string>("all");
    const [hideDeprecated, setHideDeprecated] = useState(false);
    const [search, setSearch] = useState("");

    const handleFilterChange = (updates: Partial<MQTTFilterState>) => {
        const newFilter = {
            group: updates.group !== undefined ? updates.group : selectedGroup,
            hideDeprecated: updates.hideDeprecated !== undefined ? updates.hideDeprecated : hideDeprecated,
            search: updates.search !== undefined ? updates.search : search,
        };
        onFilterChange(newFilter);
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
                    placeholder='Search by operation name, topic, or description...'
                    className='flex-1 px-3 py-2 rounded-lg text-sm bg-bg-tertiary text-text-primary border-2 border-border-primary hover:border-accent-blue focus:border-accent-blue focus:ring-2 focus:ring-accent-blue placeholder-text-muted'
                />
            </div>
        </div>
    );
}
