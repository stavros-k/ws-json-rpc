"use client";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Protocols } from "./protocols";
import type { getItemData } from "./sidebar";

type Props = {
    type: "method" | "event" | "type";
    item: ReturnType<typeof getItemData>;
};

export const SidebarItem = ({ type, item }: Props) => {
    const params = useParams();
    const currentName = params[type];
    const { urlPath, data, title } = item;
    const isDeprecated = "deprecated" in data ? data.deprecated : false;
    const isActive = currentName === item.name;

    return (
        <Link
            key={urlPath}
            href={urlPath}
            className={`block ${
                isActive ? "bg-accent-blue text-white shadow-md" : "bg-bg-secondary text-text-primary"
            } py-2 px-2.5 rounded-lg mb-1.5 text-sm no-underline transition-all duration-200 hover:bg-bg-tertiary hover:shadow-sm border-2 ${
                isActive ? "border-accent-blue-border" : "border-border-primary"
            } ${isDeprecated ? "opacity-40" : ""}`}>
            <div className='flex items-center justify-between'>
                <span className='font-medium'>{title || item.name}</span>
                <Protocols item={item} />
            </div>
        </Link>
    );
};
