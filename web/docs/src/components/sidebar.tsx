import type { Route } from "next";
import Link from "next/link";
import { IoHome } from "react-icons/io5";
import {
    docs,
    type EventKeys,
    type ItemType,
    type MethodKeys,
    type TypeKeys,
} from "@/data/api";
import { AutoSubscribeToggle } from "./auto-subscribe-toggle";
import { CodeThemeToggle } from "./code-theme-toggle";
import { ConnectionIndicator } from "./connection-indicator";
import { MaxResultsControl } from "./max-results-control";
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
        <aside className='w-80 p-6 text-sm bg-bg-sidebar text-text-primary overflow-y-auto sticky top-0 max-h-screen border-r-2 border-border-sidebar'>
            <div className='mb-8'>
                <div className='flex items-center justify-between mb-4'>
                    <h1 className='text-xl font-bold'>
                        <Link
                            href='/'
                            className='flex items-center gap-2.5 transition-colors'>
                            <IoHome className='w-8 h-8 shrink-0' />
                            <div className='flex flex-col'>
                                <span>{docs.info.title}</span>
                                <span className='text-xs text-text-muted font-normal'>
                                    v{docs.info.version}
                                </span>
                            </div>
                        </Link>
                    </h1>
                    <ConnectionIndicator />
                </div>
                <div className='space-y-3'>
                    <AutoSubscribeToggle />
                    <MaxResultsControl />
                    <CodeThemeToggle />
                </div>
            </div>

            <SidebarSection
                title='Methods'
                type='method'
                overviewHref='/api/methods'
            />
            <SidebarSection
                title='Events'
                type='event'
                overviewHref='/api/events'
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
        </aside>
    );
};

type getItemProps = {
    type: ItemType;
    itemName: string;
};

export function getItemData({ type, itemName }: getItemProps) {
    if (type === "method") {
        const name = itemName as MethodKeys;
        if (!docs.methods[name]) {
            throw new Error(`Method ${name} not found`);
        }
        return {
            type: type,
            name: name,
            urlPath: `/api/${type}/${name}`,
            data: docs.methods[name],
            title: docs.methods[name].title,
            group: docs.methods[name].group,
        } as const;
    } else if (type === "event") {
        const name = itemName as EventKeys;
        if (!docs.events[name]) {
            throw new Error(`Event ${name} not found`);
        }
        return {
            type: type,
            name: name,
            urlPath: `/api/${type}/${name}`,
            data: docs.events[name],
            title: docs.events[name].title,
            group: docs.events[name].group,
        } as const;
    } else if (type === "type") {
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
    throw new Error("Invalid type");
}
