import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";

type ItemCardProps = {
    href: Route;
    title: string;
    subtitle?: ReactNode;
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
            className={`block p-6 bg-bg-secondary rounded-xl border-2 border-border-primary ${hoverBorderColor} hover:shadow-lg transition-all duration-200 relative ${
                deprecated ? "opacity-deprecated hover:opacity-deprecated-hover pb-12" : ""
            }`}>
            <div className='flex items-start justify-between gap-4'>
                <div className='flex-1 min-w-0'>
                    <h3 className='text-xl font-bold text-text-primary mb-2'>{title}</h3>
                    {subtitle && <div className={`text-base font-semibold mb-2 ${subtitleColor}`}>{subtitle}</div>}
                    <p className='text-text-secondary text-sm leading-relaxed'>{description}</p>
                </div>
                <div className='flex flex-col gap-2 items-end shrink-0'>
                    {tags && <div className='flex gap-2 flex-wrap justify-end'>{tags}</div>}
                    {badges && <div className='flex gap-2 flex-wrap justify-end'>{badges}</div>}
                </div>
            </div>
            {deprecated && (
                <div className='absolute bottom-4 right-4 px-3 py-1.5 bg-warning-bg border-2 border-warning-border rounded-lg text-warning-text text-xs font-bold'>
                    Deprecated
                </div>
            )}
        </Link>
    );
};
