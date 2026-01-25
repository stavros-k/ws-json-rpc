import type { Route } from "next";
import Link from "next/link";
import { IoHome, IoChevronForward } from "react-icons/io5";

type BreadcrumbItem = {
    label: string;
    href?: Route;
};

type BreadcrumbsProps = {
    items: BreadcrumbItem[];
};

export function Breadcrumbs({ items }: BreadcrumbsProps) {
    return (
        <nav className='flex items-center gap-2 text-sm mb-6 text-text-secondary'>
            <Link
                href='/'
                className='flex items-center gap-1 hover:text-accent-blue transition-colors'>
                <IoHome className='w-4 h-4' />
                <span>Home</span>
            </Link>
            {items.map((item, index) => {
                const isLast = index === items.length - 1;
                return (
                    <div
                        key={index}
                        className='flex items-center gap-2'>
                        <IoChevronForward className='w-3 h-3' />
                        {item.href && !isLast ? (
                            <Link
                                href={item.href}
                                className='hover:text-accent-blue transition-colors'>
                                {item.label}
                            </Link>
                        ) : (
                            <span className={isLast ? "text-text-primary font-semibold" : ""}>{item.label}</span>
                        )}
                    </div>
                );
            })}
        </nav>
    );
}
