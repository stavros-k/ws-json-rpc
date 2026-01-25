"use client";
import Link from "next/link";
import { useParams } from "next/navigation";
import type { ItemType } from "@/data/api";
import { RoutePath } from "./route-path";
import type { getItemData } from "./sidebar";
import { VerbBadge } from "./verb-badge";

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
    const verb = "verb" in item ? item.verb : undefined;
    const route = "route" in item ? item.route : undefined;

    return (
        <Link
            key={urlPath}
            href={urlPath}
            className={`block ${
                isActive ? "bg-bg-tertiary text-text-primary shadow-md" : "bg-bg-secondary text-text-primary"
            } py-2 px-2.5 rounded-lg mb-1.5 text-sm no-underline transition-all duration-200 hover:shadow-sm border-2 ${
                isActive ? "border-accent-blue-border" : "border-border-primary hover:border-border-secondary"
            } ${isDeprecated ? "opacity-deprecated" : ""}`}>
            <div className='flex flex-col gap-1'>
                <span className='font-semibold text-base'>{title || item.name}</span>
                {verb && route && (
                    <div className='flex items-center gap-1'>
                        <VerbBadge
                            verb={verb}
                            size='xs'
                        />
                        <RoutePath
                            path={route}
                            className='text-[10px] text-text-muted font-mono truncate'
                        />
                    </div>
                )}
            </div>
        </Link>
    );
};
