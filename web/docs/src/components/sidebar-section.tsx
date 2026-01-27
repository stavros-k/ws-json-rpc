"use client";

import type { Route } from "next";
import Link from "next/link";
import { useState } from "react";
import { BiLinkExternal } from "react-icons/bi";
import { MdExpandLess, MdExpandMore } from "react-icons/md";
import {
    docs,
    getAllMQTTPublications,
    getAllMQTTSubscriptions,
    getAllOperations,
    type ItemType,
    type TypeKeys,
} from "@/data/api";
import { groupBy } from "@/data/utils";
import { getItemData } from "./sidebar";
import { SidebarGroupCollapsible } from "./sidebar-group-collapsible";
import { SidebarItem } from "./sidebar-item-client";

function getItems(type: ItemType) {
    if (type === "type") {
        return Object.keys(docs.types) as TypeKeys[];
    }
    if (type === "operation") {
        const operations = getAllOperations();
        return operations.map((op) => op.operationID);
    }
    if (type === "mqtt-publication") {
        const publications = getAllMQTTPublications();
        return publications.map((pub) => pub.operationID);
    }
    if (type === "mqtt-subscription") {
        const subscriptions = getAllMQTTSubscriptions();
        return subscriptions.map((sub) => sub.operationID);
    }
    throw new Error(`Invalid type: ${type}`);
}

type SidebarSectionProps = Readonly<{
    title: string;
    type: ItemType;
    overviewHref: Route;
}>;

export const SidebarSection = ({ title, type, overviewHref }: SidebarSectionProps) => {
    const items = getItems(type);

    // Special grouping for types based on HTTP/MQTT usage
    const groupedItems =
        type === "type"
            ? groupBy(items, (itemName) => {
                  const typeData = docs.types[itemName as TypeKeys];
                  if (!typeData || !("usedBy" in typeData) || !typeData.usedBy) {
                      return ""; // No group for unused types
                  }

                  const usedBy = typeData.usedBy;
                  const usedByHTTP = usedBy.some((usage) =>
                      ["request", "response", "parameter"].includes(usage.role)
                  );
                  const usedByMQTT = usedBy.some((usage) =>
                      ["mqtt_publication", "mqtt_subscription"].includes(usage.role)
                  );

                  if (usedByHTTP && usedByMQTT) {
                      return ""; // No group for types used by both
                  }
                  if (usedByHTTP) {
                      return "HTTP";
                  }
                  if (usedByMQTT) {
                      return "MQTT";
                  }
                  return ""; // No group for types not used
              })
            : groupBy(items, (itemName) => getItemData({ type, itemName }).group);

    const sortedGroups = Object.keys(groupedItems).sort((a, b) => {
        // Empty group first
        if (a === "") return -1;
        if (b === "") return 1;
        // Then alphabetical
        return a.localeCompare(b);
    });

    // Use type as storage key to keep state persistent
    const storageKey = `sidebar-section-${type}`;
    const [isOpen, setIsOpen] = useState(() => {
        if (typeof window === "undefined") return true;
        const stored = localStorage.getItem(storageKey);
        return stored === null ? true : stored === "true";
    });

    const toggleOpen = () => {
        const newState = !isOpen;
        setIsOpen(newState);
        if (typeof window !== "undefined") {
            localStorage.setItem(storageKey, String(newState));
        }
    };

    if (!items.length) return null;

    return (
        <div className='mb-8'>
            <div className='flex items-center justify-between mb-3'>
                <Link
                    href={overviewHref}
                    className='inline-flex items-center gap-1.5 p-1.5 rounded-lg hover:bg-bg-hover transition-all duration-200 group flex-1'>
                    <h2 className='text-sm font-bold text-text-secondary uppercase transition-colors'>{title}</h2>
                    <BiLinkExternal className='w-3.5 h-3.5 text-text-muted transition-colors' />
                </Link>
                <button
                    type='button'
                    onClick={toggleOpen}
                    className='p-1.5 rounded hover:bg-bg-hover transition-colors'
                    aria-label={isOpen ? "Collapse section" : "Expand section"}>
                    {isOpen ? (
                        <MdExpandLess className='w-4 h-4 text-text-secondary' />
                    ) : (
                        <MdExpandMore className='w-4 h-4 text-text-secondary' />
                    )}
                </button>
            </div>

            {isOpen &&
                sortedGroups.map((groupName) => {
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
