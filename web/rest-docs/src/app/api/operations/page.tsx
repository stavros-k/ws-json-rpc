import type { Route } from "next";
import { ItemCard } from "@/components/item-card";
import { PageHeader } from "@/components/page-header";
import { GroupAndTags } from "@/components/group-and-tags";
import { getAllOperations } from "@/data/api";

export const metadata = {
    title: "Operations - API Documentation",
};

export default function OperationsPage() {
    const operations = getAllOperations();

    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <PageHeader
                title='Operations'
                description='Browse all API operations and endpoints'
            />

            <div className='grid gap-5'>
                {operations.map((operation) => (
                    <ItemCard
                        key={operation.operationID}
                        href={`/api/operation/${operation.operationID}` as Route}
                        title={operation.operationID}
                        subtitle={`${operation.verb.toUpperCase()} ${operation.route}`}
                        description={operation.summary || operation.description}
                        tags={
                            <GroupAndTags
                                group={operation.group || ""}
                                tags={operation.tags}
                            />
                        }
                        deprecated={!!operation.deprecated}
                        hoverBorderColor='hover:border-accent-green-light'
                    />
                ))}
            </div>
        </main>
    );
}
