"use client";

import type { ReactNode } from "react";
import { useState } from "react";
import { MdExpandLess, MdExpandMore } from "react-icons/md";

type CollapsibleResponseProps = {
    statusCode: string;
    description: string;
    children: ReactNode;
};

export function CollapsibleResponse({ statusCode, description, children }: CollapsibleResponseProps) {
    const [isOpen, setIsOpen] = useState(false);

    return (
        <div className='border-2 border-border-secondary rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow'>
            <button
                type='button'
                onClick={() => setIsOpen(!isOpen)}
                className='w-full bg-bg-tertiary p-5 font-bold cursor-pointer flex justify-between items-center hover:bg-bg-hover text-text-primary transition-colors duration-200 group'>
                <div className='flex items-center gap-3'>
                    <span
                        className={`text-lg font-bold px-3 py-1.5 rounded-lg font-mono ${
                            statusCode.startsWith("2")
                                ? "bg-green-500/20 text-green-400 border-2 border-green-500/30"
                                : statusCode.startsWith("4")
                                  ? "bg-yellow-500/20 text-yellow-400 border-2 border-yellow-500/30"
                                  : "bg-red-500/20 text-red-400 border-2 border-red-500/30"
                        }`}>
                        {statusCode}
                    </span>
                    <span className='text-sm text-text-muted font-normal'>{description}</span>
                </div>
                <span className='text-text-muted group-hover:text-accent-blue transition-colors'>
                    {isOpen ? <MdExpandLess className='w-6 h-6' /> : <MdExpandMore className='w-6 h-6' />}
                </span>
            </button>
            {isOpen && <div className='p-5 bg-bg-secondary'>{children}</div>}
        </div>
    );
}
