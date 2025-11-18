"use client";

import { useState } from "react";
import { MdExpandLess, MdExpandMore } from "react-icons/md";

type SidebarGroupCollapsibleProps = {
    groupName: string;
    children: React.ReactNode;
};

export const SidebarGroupCollapsible = ({
    groupName,
    children,
}: SidebarGroupCollapsibleProps) => {
    const [isOpen, setIsOpen] = useState(true);

    return (
        <div className='mb-3'>
            <button
                type='button'
                onClick={() => setIsOpen(!isOpen)}
                className='w-full flex items-center justify-between px-1 py-1 rounded hover:bg-bg-hover transition-colors group'>
                <h3 className='text-xs font-semibold text-text-dim group-hover:text-text-primary uppercase transition-colors'>
                    {groupName}
                </h3>
                <span className='text-text-dim group-hover:text-text-primary transition-colors'>
                    {isOpen ? (
                        <MdExpandLess className='w-4 h-4' />
                    ) : (
                        <MdExpandMore className='w-4 h-4' />
                    )}
                </span>
            </button>
            {isOpen && <div className='mt-1'>{children}</div>}
        </div>
    );
};
