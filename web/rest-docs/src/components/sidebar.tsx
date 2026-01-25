import type { Route } from "next";
import Link from "next/link";
import { IoHome } from "react-icons/io5";
import { docs, getAllOperations, type ItemType, type TypeKeys } from "@/data/api";
import { CodeThemeToggle } from "./code-theme-toggle";
import { ConnectionIndicator } from "./connection-indicator";
import { SidebarSection } from "./sidebar-section";

const SidebarLink = ({ title, href }: { title: string; href: Route }) => {
    return (
        <div className='mt-6 pt-6 border-t-2 border-border-sidebar'>
            <Link
                href={href}
                className='block py-2.5 px-3 rounded-lg font-medium hover:bg-bg-hover transition-all duration-200'>
                {title}
            </Link>
        </div>
    );
};

export const Sidebar = () => {
    return (
        <aside className='w-80 text-sm bg-bg-sidebar text-text-primary sticky top-0 max-h-screen border-r-2 border-border-sidebar flex flex-col'>
            {/* Sticky Header */}
            <div className='p-6 pb-4 border-b-2 border-border-sidebar'>
                <div className='flex items-center justify-between mb-4'>
                    <h1 className='text-xl font-bold'>
                        <Link
                            href='/'
                            className='flex items-center gap-2.5 transition-colors'>
                            <IoHome className='w-8 h-8 shrink-0' />
                            <div className='flex flex-col'>
                                <span>{docs.info.title}</span>
                                <span className='text-xs text-text-muted font-normal'>v{docs.info.version}</span>
                            </div>
                        </Link>
                    </h1>
                    <ConnectionIndicator />
                </div>
                <div className='space-y-3'>
                    <CodeThemeToggle />
                </div>
            </div>

            {/* Scrollable Content */}
            <div
                className='flex-1 overflow-y-scroll p-6 pt-4'
                style={{
                    scrollbarWidth: "auto",
                    scrollbarColor: "rgb(100 116 139) transparent",
                }}>
                <SidebarSection
                    title='Operations'
                    type='operation'
                    overviewHref='/api/operations'
                />
                <SidebarSection
                    title='Types'
                    type='type'
                    overviewHref='/api/types'
                />

                <SidebarLink
                    title='Database Schema'
                    href='/api/database/schema'
                />
            </div>
        </aside>
    );
};

type getItemProps = {
    type: ItemType;
    itemName: string;
};

export function getItemData({ type, itemName }: getItemProps) {
    if (type === "type") {
        const name = itemName as TypeKeys;
        if (!docs.types[name]) {
            throw new Error(`Type ${name} not found`);
        }
        return {
            type: type,
            name: name,
            urlPath: `/api/${type}/${name}`,
            data: docs.types[name],
            title: "",
            group: "",
        } as const;
    }
    if (type === "operation") {
        // itemName is operationID for operations
        const allOperations = getAllOperations();
        const operation = allOperations.find((op) => op.operationID === itemName);
        if (!operation) {
            throw new Error(`Operation ${itemName} not found`);
        }
        return {
            type: type,
            name: itemName,
            urlPath: `/api/operation/${itemName}`,
            data: operation,
            title: operation.operationID,
            verb: operation.verb,
            route: operation.route,
            group: operation.group || "",
        } as const;
    }
    throw new Error(`Invalid type: ${type}`);
}
