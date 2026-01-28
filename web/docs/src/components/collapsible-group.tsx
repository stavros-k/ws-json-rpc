"use client";

import { useState } from "react";
import { MdExpandLess, MdExpandMore } from "react-icons/md";

type CollapsibleGroupProps = {
    title: string;
    children: React.ReactNode;
    defaultOpen?: boolean;
};

export function CollapsibleGroup({ title, children, defaultOpen = true }: CollapsibleGroupProps) {
    const [isOpen, setIsOpen] = useState(defaultOpen);

    return (
        <div className='mb-12'>
            <button
                type='button'
                onClick={() => setIsOpen(prev => !prev)}
                className='w-full flex items-center justify-between text-2xl font-bold text-text-primary mb-6 pb-3 border-b-2 border-border-primary hover:text-accent-blue hover:border-accent-blue transition-all duration-200 group'>
                <h2 className='group-hover:scale-105 transition-transform'>{title}</h2>
                <span className='text-text-muted group-hover:text-accent-blue transition-colors'>
                    {isOpen ? <MdExpandLess className='w-7 h-7' /> : <MdExpandMore className='w-7 h-7' />}
                </span>
            </button>
            {isOpen && <div className='grid gap-5'>{children}</div>}
        </div>
    );
}

type CollapsibleCardProps = {
    title: string;
    subtitle?: string;
    children: React.ReactNode;
    defaultOpen?: boolean;
};

export function CollapsibleCard({ title, subtitle, children, defaultOpen = false }: CollapsibleCardProps) {
    const [isOpen, setIsOpen] = useState(defaultOpen);

    return (
        <div className='border-2 border-border-secondary rounded-xl mb-6 overflow-hidden shadow-sm hover:shadow-md transition-shadow'>
            <button
                type='button'
                onClick={() => setIsOpen(prev => !prev)}
                className='w-full bg-bg-tertiary p-5 font-bold cursor-pointer flex justify-between items-center hover:bg-bg-hover text-text-primary transition-colors duration-200 group'>
                <div className='text-left'>
                    <div className='group-hover:text-accent-blue transition-colors'>{title}</div>
                    {subtitle && <div className='text-sm text-text-muted mt-1.5 font-normal'>{subtitle}</div>}
                </div>
                <span className='text-text-muted group-hover:text-accent-blue transition-colors'>
                    {isOpen ? <MdExpandLess className='w-6 h-6' /> : <MdExpandMore className='w-6 h-6' />}
                </span>
            </button>
            {isOpen && <div className='p-5 bg-bg-secondary'>{children}</div>}
        </div>
    );
}
