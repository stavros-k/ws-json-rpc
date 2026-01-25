"use client";
import Link from "next/link";
import { useParams } from "next/navigation";
import type { getItemData } from "./sidebar";
import type { ItemType } from "@/data/api";

type Props = {
    type: ItemType;
    item: ReturnType<typeof getItemData>;
};

export const SidebarItem = ({ type, item }: Props) => {
    const params = useParams();
    const currentName = type === "operation" ? params.operationId : params[type];
    const { urlPath, data, title } = item;
    const isDeprecated = !!data?.deprecated;
    const isActive = currentName === item.name;
    const subtitle = "subtitle" in item ? item.subtitle : "";

    return (
        <Link
            key={urlPath}
            href={urlPath}
            className={`block ${
                isActive ? "bg-accent-blue text-white shadow-md" : "bg-bg-secondary text-text-primary"
            } py-2 px-2.5 rounded-lg mb-1.5 text-sm no-underline transition-all duration-200 hover:bg-bg-tertiary hover:shadow-sm border-2 ${
                isActive ? "border-accent-blue-border" : "border-border-primary"
            } ${isDeprecated ? "opacity-40" : ""}`}>
            <div className='flex flex-col'>
                <span className='font-medium'>{title || item.name}</span>
                {subtitle && <span className='text-xs text-text-muted mt-0.5 font-mono'>{subtitle}</span>}
            </div>
        </Link>
    );
};
