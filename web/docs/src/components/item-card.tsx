import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";

type ItemCardProps = {
    href: Route;
    title: string;
    subtitle?: string;
    subtitleColor?: string;
    description: string;
    badges?: ReactNode;
    tags?: ReactNode;
    deprecated?: boolean;
    hoverBorderColor?: string;
};

export const ItemCard = ({
    href,
    title,
    subtitle,
    subtitleColor = "text-accent-blue",
    description,
    badges,
    tags,
    deprecated,
    hoverBorderColor = "hover:border-accent-blue",
}: ItemCardProps) => {
    return (
        <Link
            href={href}
            className={`block p-6 bg-bg-secondary rounded-xl border-2 border-border-primary ${hoverBorderColor} hover:shadow-lg transition-all duration-200`}>
            <div className='flex items-start justify-between mb-3'>
                <div className='flex-1'>
                    <h3 className='text-xl font-bold text-text-primary mb-2'>
                        {title}
                    </h3>
                    {subtitle && (
                        <p
                            className={`text-base font-semibold mb-3 ${subtitleColor}`}>
                            {subtitle}
                        </p>
                    )}
                    <p className='text-text-secondary text-sm leading-relaxed'>
                        {description}
                    </p>
                </div>
                {badges && (
                    <div className='flex gap-2 ml-4 flex-shrink-0'>
                        {badges}
                    </div>
                )}
            </div>
            {deprecated && (
                <div className='mt-3 inline-block px-3 py-1.5 bg-warning-bg border-2 border-warning-border rounded-lg text-warning-text text-xs font-bold'>
                    Deprecated
                </div>
            )}
            {tags && <div className='mt-3 flex gap-2 flex-wrap'>{tags}</div>}
        </Link>
    );
};
