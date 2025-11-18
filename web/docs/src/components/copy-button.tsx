"use client";

import { useEffect, useState } from "react";
import { FiCheck, FiCopy } from "react-icons/fi";

type Props = {
    code: string;
};

export const CopyButton = ({ code }: Props) => {
    const [copied, setCopied] = useState(false);

    useEffect(() => {
        if (!copied) return;

        const timeoutId = setTimeout(() => {
            setCopied(false);
        }, 2000);

        return () => {
            clearTimeout(timeoutId);
        };
    }, [copied]);

    const handleCopy = async () => {
        try {
            await navigator.clipboard.writeText(code);
            setCopied(true);
        } catch (err) {
            console.error("Failed to copy text:", err);
        }
    };

    return (
        <button
            onClick={handleCopy}
            className='absolute top-3 right-3 p-2 rounded-lg bg-bg-secondary hover:bg-bg-tertiary border border-border-primary transition-all duration-200 group'
            title={copied ? "Copied!" : "Copy code"}
            type='button'>
            {copied ? (
                <FiCheck className='w-4 h-4 text-green-500' />
            ) : (
                <FiCopy className='w-4 h-4 text-text-secondary group-hover:text-text-primary transition-colors' />
            )}
        </button>
    );
};
