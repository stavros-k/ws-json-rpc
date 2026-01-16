"use client";

import type { Route } from "next";
import { useState } from "react";
import { CollapsibleGroup } from "@/components/collapsible-group";
import { FilterControls, type FilterState } from "@/components/filter-controls";
import { ItemCard } from "@/components/item-card";
import { ProtocolBadge } from "@/components/protocol-badge";
import { docs } from "@/data/api";
import { groupBy } from "@/data/utils";

export function MethodsList() {
    const [filter, setFilter] = useState<FilterState>({
        protocol: "all",
        hideDeprecated: false,
    });

    const methods = Object.entries(docs.methods);

    // Filter methods based on protocol and deprecated status
    const filteredMethods = methods.filter(([, method]) => {
        // Filter by protocol
        if (filter.protocol !== "all") {
            if (filter.protocol === "http" && !method.protocols.http) {
                return false;
            }
            if (filter.protocol === "ws" && !method.protocols.ws) {
                return false;
            }
        }

        // Filter by deprecated
        if (filter.hideDeprecated && method.deprecated) {
            return false;
        }

        return true;
    });

    // Group filtered methods by their group property
    const groupedMethods = groupBy(filteredMethods, ([, method]) => method.group);

    return (
        <>
            <FilterControls onFilterChange={setFilter} />

            {Object.keys(groupedMethods).length === 0 ? (
                <div className='text-center py-12 text-text-secondary'>
                    No methods found matching the selected filters.
                </div>
            ) : (
                Object.entries(groupedMethods).map(([group, items]) => (
                    <CollapsibleGroup
                        key={group}
                        title={group}>
                        {items.map(([key, method]) => (
                            <ItemCard
                                key={key}
                                href={`/api/method/${key}` as Route}
                                title={method.title}
                                subtitle={key}
                                description={method.description}
                                deprecated={method.deprecated}
                                badges={
                                    <>
                                        <ProtocolBadge
                                            title='HTTP'
                                            supported={method.protocols.http}
                                        />
                                        <ProtocolBadge
                                            title='WS'
                                            supported={method.protocols.ws}
                                        />
                                    </>
                                }
                                tags={method.tags.map((tag) => (
                                    <span
                                        key={tag}
                                        className='px-3 py-1.5 bg-tag-blue-bg border border-tag-blue-border rounded-lg text-tag-blue-text text-xs font-medium'>
                                        {tag}
                                    </span>
                                ))}
                            />
                        ))}
                    </CollapsibleGroup>
                ))
            )}
        </>
    );
}
