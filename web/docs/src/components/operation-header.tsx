"use client";
import { Fragment, useState } from "react";
import type { OperationData } from "@/data/api";
import { VerbBadge } from "./verb-badge";

type OperationHeaderProps = {
    method: string;
    path: string;
    parameters?: OperationData["parameters"];
};

export function OperationHeader({ method, path, parameters }: OperationHeaderProps) {
    const [hoveredParam, setHoveredParam] = useState<string | null>(null);

    // Split the path into segments and identify parameters
    const segments = path.split(/(\{[^}]+\})/g).filter(Boolean);

    return (
        <div>
            <div className='flex items-center gap-3 mb-4'>
                <VerbBadge
                    verb={method}
                    size='lg'
                />
                <h2 className='text-xl font-mono font-semibold'>
                    <span>
                        {segments.map((segment, idx) => {
                            const uniqueKey = `${segment}-${idx}`;
                            const isParam = segment.startsWith("{") && segment.endsWith("}");
                            const paramName = isParam ? segment.slice(1, -1) : null;
                            const isHovered = hoveredParam === paramName;
                            if (!isParam) {
                                return (
                                    <span
                                        key={uniqueKey}
                                        className='text-text-secondary'>
                                        {segment}
                                    </span>
                                );
                            }

                            return (
                                <Fragment key={uniqueKey}>
                                    <span className='text-type-enum'>{"{"}</span>
                                    <button
                                        type='button'
                                        className={`font-bold transition-colors bg-transparent border-0 p-0 font-mono text-xl ${
                                            isHovered ? "text-accent-blue-light" : "text-accent-blue"
                                        }`}
                                        onMouseEnter={() => setHoveredParam(paramName)}
                                        onMouseLeave={() => setHoveredParam(null)}
                                        aria-label={`Highlight ${paramName} parameter`}>
                                        {paramName}
                                    </button>
                                    <span className='text-type-enum'>{"}"}</span>
                                </Fragment>
                            );
                        })}
                    </span>
                </h2>
            </div>

            {parameters && parameters.length > 0 && (
                <div className='mb-8'>
                    <h3 className='text-2xl font-bold text-text-primary mb-4'>Parameters</h3>
                    <div className='space-y-3'>
                        {parameters.map((param) => {
                            const isHovered = hoveredParam === param.name;
                            return (
                                <button
                                    key={param.name}
                                    type='button'
                                    className={`w-full p-4 rounded-lg border-2 transition-colors text-left ${
                                        isHovered
                                            ? "bg-accent-blue/5 border-accent-blue/50"
                                            : "bg-bg-tertiary border-border-primary"
                                    }`}
                                    onMouseEnter={() => setHoveredParam(param.name)}
                                    onMouseLeave={() => setHoveredParam(null)}
                                    aria-label={`Highlight ${param.name} parameter in path`}>
                                    <div className='flex items-start justify-between gap-4 mb-2'>
                                        <div className='flex items-center gap-2 flex-wrap'>
                                            <code className='text-base font-semibold text-text-primary'>
                                                {param.name}
                                            </code>
                                            {param.required && (
                                                <span className='text-xs px-2 py-0.5 rounded bg-red-500/20 text-red-400 border border-red-500/30 font-semibold'>
                                                    required
                                                </span>
                                            )}
                                            <span className='text-xs px-2 py-0.5 rounded bg-blue-500/20 text-blue-400 border border-blue-500/30'>
                                                {param.in}
                                            </span>
                                        </div>
                                        <code className='px-3 py-1.5 rounded-lg bg-type-primitive/10 text-type-primitive border-2 border-type-primitive/30 font-mono text-sm font-semibold'>
                                            {param.type}
                                        </code>
                                    </div>
                                    {param.description && (
                                        <p className='text-sm text-text-tertiary'>{param.description}</p>
                                    )}
                                </button>
                            );
                        })}
                    </div>
                </div>
            )}
        </div>
    );
}
