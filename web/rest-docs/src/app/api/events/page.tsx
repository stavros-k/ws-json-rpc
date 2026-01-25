import { PageHeader } from "@/components/page-header";
import { EventsList } from "./events-list";

export const metadata = {
    title: "Events - API Documentation",
};

export default function EventsPage() {
    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <PageHeader
                title='Events'
                description='Browse all subscribable events in the API'
            />
            <EventsList />
        </main>
    );
}
