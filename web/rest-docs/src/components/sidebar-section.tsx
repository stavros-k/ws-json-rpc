import type { Route } from "next";
import Link from "next/link";
import { BiLinkExternal } from "react-icons/bi";
import { docs, type EventKeys, type ItemType, type MethodKeys, type TypeKeys } from "@/data/api";
import { groupBy } from "@/data/utils";
import { getItemData } from "./sidebar";
import { SidebarGroupCollapsible } from "./sidebar-group-collapsible";
import { SidebarItem } from "./sidebar-item-client";

function getItems(type: ItemType) {
    if (type === "method") {
        return Object.keys(docs.methods) as MethodKeys[];
    } else if (type === "event") {
        return Object.keys(docs.events) as EventKeys[];
    } else if (type === "type") {
        return Object.keys(docs.types) as TypeKeys[];
    }
    throw new Error("Invalid type");
}

type SidebarSectionProps = Readonly<{
    title: string;
    type: ItemType;
    overviewHref: Route;
}>;

export const SidebarSection = ({ title, type, overviewHref }: SidebarSectionProps) => {
    const items = getItems(type);
    const groupedItems = groupBy(items, (itemName) => getItemData({ type, itemName }).group);

    const sortedGroups = Object.keys(groupedItems).sort();

    if (!items.length) return null;

    return (
        <div className='mb-8'>
            <Link
                href={overviewHref}
                className='inline-flex items-center gap-1.5 mb-3 p-1.5 rounded-lg hover:bg-bg-hover transition-all duration-200 group'>
                <h2 className='text-sm font-bold text-text-secondary uppercase transition-colors'>{title}</h2>
                <BiLinkExternal className='w-3.5 h-3.5 text-text-muted transition-colors' />
            </Link>
            {sortedGroups.map((groupName) => {
                const groupItems = groupedItems[groupName].map((itemName) => {
                    const item = getItemData({ type, itemName });
                    return (
                        <SidebarItem
                            key={item.urlPath}
                            type={type}
                            item={item}
                        />
                    );
                });

                // Don't use collapsible for empty group name
                if (groupName === "") {
                    return (
                        <div
                            key={groupName}
                            className='mb-3'>
                            {groupItems}
                        </div>
                    );
                }

                return (
                    <SidebarGroupCollapsible
                        key={groupName}
                        groupName={groupName}>
                        {groupItems}
                    </SidebarGroupCollapsible>
                );
            })}
        </div>
    );
};
