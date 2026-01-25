"use client";
import Link from "next/link";
import { useParams } from "next/navigation";
import type { ItemType } from "@/data/api";
import { RoutePath } from "./route-path";
import type { getItemData } from "./sidebar";
import { TypeKindBadge } from "./type-kind-badge";
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
    const method = "method" in item ? item.method : undefined;
    const path = "path" in item ? item.path : undefined;
    const kind = data && "kind" in data ? data.kind : undefined;

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
                <div className='flex items-center justify-between gap-1.5'>
                    <span className='font-semibold text-base truncate'>{title || item.name}</span>
                    {kind && type === "type" && (
                        <TypeKindBadge
                            kind={kind}
                            size='xs'
                        />
                    )}
                </div>
                {method && path && (
                    <div className='flex items-center gap-1'>
                        <VerbBadge
                            verb={method}
                            size='xs'
                        />
                        <RoutePath
                            path={path}
                            className='text-xs text-text-muted font-mono truncate'
                        />
                    </div>
                )}
            </div>
        </Link>
    );
};
