"use client";

import { useEffect, useState } from "react";
import type { BundledLanguage } from "shiki";
import { codeToHtml } from "shiki";
import { getShikiOptions } from "@/utils/code-theme";
import { CopyButton } from "./copy-button";

type Props = {
    code: string;
    lang: BundledLanguage;
};

export function CodeWrapperClient({ code, lang }: Props) {
    const [html, setHtml] = useState<string>("");

    useEffect(() => {
        codeToHtml(code, getShikiOptions(lang))
            .then((result) => {
                setHtml(result);
            })
            .catch((err) => {
                console.error("Failed to highlight code:", err);
            });
    }, [code, lang]);

    return (
        <div className='relative'>
            <CopyButton code={code} />
            {/* biome-ignore lint/security/noDangerouslySetInnerHtml: This is sanitized by shiki */}
            <div dangerouslySetInnerHTML={{ __html: html }} />
        </div>
    );
}
