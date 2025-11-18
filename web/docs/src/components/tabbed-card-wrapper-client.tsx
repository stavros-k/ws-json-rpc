"use client";
import { useState } from "react";

type Tab = {
    // https://react-icons.github.io/react-icons/
    icon: React.ReactNode;
    title: string;
    code: React.ReactNode;
};

type Props = {
    tabs: Tab[];
};

export const TabbedCardWrapper = ({ tabs }: Props) => {
    const [activeTab, setActiveTab] = useState(0);

    if (!tabs || tabs.length === 0) return null;

    return (
        <div className='bg-bg-secondary rounded-lg p-6 mb-6 shadow-sm border border-border-primary'>
            <div>
                <div className='flex border-b border-border-secondary mb-4 overflow-x-auto'>
                    {tabs.map((tab, index) => (
                        <button
                            key={tab.title}
                            type='button'
                            onClick={() => setActiveTab(index)}
                            className={`flex items-center gap-2 px-4 py-2 cursor-pointer border-b-2 transition-colors whitespace-nowrap ${
                                activeTab === index
                                    ? "border-accent-blue-hover text-accent-blue-hover font-semibold"
                                    : "border-transparent text-text-muted hover:text-text-secondary"
                            }`}>
                            {tab.icon}
                            <span className='text-xs'>{tab.title}</span>
                        </button>
                    ))}
                </div>
                {tabs.map((tab, index) => (
                    <div
                        key={tab.title}
                        className={activeTab === index ? "block" : "hidden"}>
                        {tab.code}
                    </div>
                ))}
            </div>
        </div>
    );
};
