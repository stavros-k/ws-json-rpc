"use client";

import { useMaxResults } from "@/contexts/max-results-context";

export function MaxResultsControl() {
    const { maxResults, setMaxResults } = useMaxResults();

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const value = Number.parseInt(e.target.value, 10);
        if (value >= 1 && value <= 100) {
            setMaxResults(value);
        }
    };

    return (
        <div className='px-3 py-2 rounded-lg border border-border-primary bg-bg-secondary'>
            <div className='flex items-center justify-between gap-3'>
                <label
                    htmlFor='max-results-input'
                    className='text-sm font-medium text-text-primary'>
                    Max Results
                </label>
                <div className='flex items-center gap-2'>
                    <input
                        id='max-results-input'
                        type='number'
                        min='1'
                        max='100'
                        value={maxResults}
                        onChange={handleChange}
                        className='w-16 px-2 py-1 text-xs border border-border-primary rounded bg-bg-primary text-text-primary focus:outline-none focus:ring-1 focus:ring-blue-500'
                    />
                    <span className='text-xs text-text-tertiary'>(1-100)</span>
                </div>
            </div>
        </div>
    );
}
