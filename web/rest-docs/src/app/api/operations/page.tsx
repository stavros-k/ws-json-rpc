"use client";

import type { Route } from "next";
import { useState, useMemo } from "react";
import { ItemCard } from "@/components/item-card";
import { PageHeader } from "@/components/page-header";
import { Group } from "@/components/group";
import { VerbBadge } from "@/components/verb-badge";
import { RoutePath } from "@/components/route-path";
import { OperationFilters, type OperationFilterState } from "@/components/operation-filters";
import { StatCard } from "@/components/stat-card";
import { EmptyState } from "@/components/empty-state";
import { CollapsibleGroup } from "@/components/collapsible-group";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { docs } from "@/data/api";
import { getAllOperations } from "@/data/api";

export default function OperationsPage() {
    const allOperations = getAllOperations();
    const [filters, setFilters] = useState<OperationFilterState>({
        verb: "all",
        group: "all",
        hideDeprecated: false,
        search: "",
    });

    // Extract unique verbs and groups
    const verbs = useMemo(() => {
        const uniqueVerbs = new Set(allOperations.map((op) => op.verb.toUpperCase()));
        return Array.from(uniqueVerbs).sort();
    }, [allOperations]);

    const groups = useMemo(() => {
        const uniqueGroups = new Set(allOperations.map((op) => op.group).filter(Boolean));
        return Array.from(uniqueGroups).sort() as string[];
    }, [allOperations]);

    // Filter operations
    const filteredOperations = useMemo(() => {
        return allOperations.filter((operation) => {
            // Verb filter
            if (filters.verb !== "all" && operation.verb.toUpperCase() !== filters.verb) {
                return false;
            }

            // Group filter
            if (filters.group !== "all" && operation.group !== filters.group) {
                return false;
            }

            // Deprecated filter
            if (filters.hideDeprecated && operation.deprecated) {
                return false;
            }

            // Search filter
            if (filters.search) {
                const searchLower = filters.search.toLowerCase();
                const matchesName = operation.operationID.toLowerCase().includes(searchLower);
                const matchesRoute = operation.route.toLowerCase().includes(searchLower);
                const matchesSummary = operation.summary?.toLowerCase().includes(searchLower);
                const matchesDescription = operation.description?.toLowerCase().includes(searchLower);

                if (!matchesName && !matchesRoute && !matchesSummary && !matchesDescription) {
                    return false;
                }
            }

            return true;
        });
    }, [allOperations, filters]);

    // Group operations by group
    const groupedOperations = useMemo(() => {
        const grouped = filteredOperations.reduce(
            (acc, operation) => {
                const group = operation.group || "Ungrouped";
                if (!acc[group]) {
                    acc[group] = [];
                }
                acc[group].push(operation);
                return acc;
            },
            {} as Record<string, typeof filteredOperations>
        );
        return Object.entries(grouped).sort(([a], [b]) => {
            // "Ungrouped" always last
            if (a === "Ungrouped") return 1;
            if (b === "Ungrouped") return -1;
            return a.localeCompare(b);
        });
    }, [filteredOperations]);

    // Calculate statistics
    const deprecatedCount = useMemo(() => {
        return allOperations.filter((op) => op.deprecated).length;
    }, [allOperations]);

    const verbCounts = useMemo(() => {
        return allOperations.reduce(
            (acc, op) => {
                const verb = op.verb.toUpperCase();
                acc[verb] = (acc[verb] || 0) + 1;
                return acc;
            },
            {} as Record<string, number>
        );
    }, [allOperations]);

    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <Breadcrumbs
                items={[
                    { label: "Operations" },
                ]}
            />

            <PageHeader
                title='Operations'
                description='Browse all API operations and endpoints'
            />

            {/* Statistics */}
            <div className='grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4 mb-8'>
                <StatCard
                    label='Total'
                    value={allOperations.length}
                    color='blue'
                />
                {Object.entries(verbCounts)
                    .sort(([a], [b]) => a.localeCompare(b))
                    .map(([verb, count]) => {
                        const colorMap: Record<string, "blue" | "green" | "yellow" | "red" | "purple"> = {
                            GET: "blue",
                            POST: "green",
                            PUT: "yellow",
                            DELETE: "red",
                            PATCH: "purple",
                        };
                        return (
                            <StatCard
                                key={verb}
                                label={verb}
                                value={count}
                                color={colorMap[verb] || "blue"}
                            />
                        );
                    })}
                {deprecatedCount > 0 && (
                    <StatCard
                        label='Deprecated'
                        value={deprecatedCount}
                        color='yellow'
                    />
                )}
            </div>

            <OperationFilters
                verbs={verbs}
                groups={groups}
                onFilterChange={setFilters}
            />

            {filteredOperations.length === 0 ? (
                <EmptyState
                    title='No operations found'
                    description="Try adjusting your filters or search query to find what you're looking for."
                    icon='🔍'
                />
            ) : (
                <div>
                    {groupedOperations.map(([groupName, operations]) => (
                        <CollapsibleGroup
                            key={groupName}
                            title={groupName}
                            defaultOpen={true}>
                            <div className='grid gap-5'>
                                {operations.map((operation) => (
                                    <ItemCard
                                        key={operation.operationID}
                                        href={`/api/operation/${operation.operationID}` as Route}
                                        title={operation.operationID}
                                        subtitle={
                                            <div className='flex items-center gap-2'>
                                                <VerbBadge
                                                    verb={operation.verb}
                                                    size='sm'
                                                />
                                                <RoutePath
                                                    path={operation.route}
                                                    className='font-mono'
                                                />
                                            </div>
                                        }
                                        description={operation.summary || operation.description}
                                        deprecated={!!operation.deprecated}
                                        hoverBorderColor='hover:border-accent-green-light'
                                    />
                                ))}
                            </div>
                        </CollapsibleGroup>
                    ))}
                </div>
            )}
        </main>
    );
}
