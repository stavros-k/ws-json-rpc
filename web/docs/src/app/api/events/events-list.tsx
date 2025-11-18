"use client";

import type { Route } from "next";
import { useState } from "react";
import { CollapsibleGroup } from "@/components/collapsible-group";
import { FilterControls, type FilterState } from "@/components/filter-controls";
import { ItemCard } from "@/components/item-card";
import { ProtocolBadge } from "@/components/protocol-badge";
import { docs } from "@/data/api";
import { groupBy } from "@/data/utils";

export function EventsList() {
    const [filter, setFilter] = useState<FilterState>({
        protocol: "all",
        hideDeprecated: false,
    });

    const events = Object.entries(docs.events);

    // Filter events based on protocol and deprecated status
    const filteredEvents = events.filter(([, event]) => {
        // Filter by protocol
        if (filter.protocol !== "all") {
            if (filter.protocol === "http" && !event.protocols.http) {
                return false;
            }
            if (filter.protocol === "ws" && !event.protocols.ws) {
                return false;
            }
        }

        // Filter by deprecated
        if (filter.hideDeprecated && event.deprecated) {
            return false;
        }

        return true;
    });

    // Group filtered events by their group property
    const groupedEvents = groupBy(filteredEvents, ([, event]) => event.group);

    return (
        <>
            <FilterControls onFilterChange={setFilter} />

            {Object.keys(groupedEvents).length === 0 ? (
                <div className='text-center py-12 text-text-secondary'>
                    No events found matching the selected filters.
                </div>
            ) : (
                Object.entries(groupedEvents).map(([group, items]) => (
                    <CollapsibleGroup
                        key={group}
                        title={group}>
                        {items.map(([key, event]) => (
                            <ItemCard
                                key={key}
                                href={`/api/event/${key}` as Route}
                                title={event.title}
                                subtitle={key}
                                subtitleColor='text-info-text'
                                description={event.description}
                                deprecated={event.deprecated}
                                hoverBorderColor='hover:border-info-border'
                                badges={
                                    <>
                                        <ProtocolBadge
                                            title='HTTP'
                                            supported={event.protocols.http}
                                        />
                                        <ProtocolBadge
                                            title='WS'
                                            supported={event.protocols.ws}
                                        />
                                    </>
                                }
                                tags={event.tags.map((tag) => (
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
