import type { Route } from "next";
import { ItemCard } from "@/components/item-card";
import { PageHeader } from "@/components/page-header";
import { docs } from "@/data/api";

export const metadata = {
    title: "Types - API Documentation",
};

export default function TypesPage() {
    const types = Object.entries(docs.types);

    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <PageHeader
                title='Types'
                description='Browse all type definitions used in the API'
            />

            <div className='grid gap-5'>
                {types.map(([key, type]) => (
                    <ItemCard
                        key={key}
                        href={`/api/type/${key}` as Route}
                        title={key}
                        description={type.description}
                        hoverBorderColor='hover:border-accent-blue-light'
                    />
                ))}
            </div>
        </main>
    );
}
