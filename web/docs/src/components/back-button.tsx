import type { Route } from "next";
import Link from "next/link";
import { IoArrowBack } from "react-icons/io5";

type BackButtonProps = {
    href: Route;
    label: string;
};

export function BackButton({ href, label }: BackButtonProps) {
    return (
        <Link
            href={href}
            className='inline-flex items-center gap-2 text-sm text-text-secondary hover:text-accent-blue transition-colors mb-6 group'>
            <IoArrowBack className='w-4 h-4 group-hover:-translate-x-1 transition-transform' />
            <span>Back to {label}</span>
        </Link>
    );
}
