import { PageHeader } from "@/components/page-header";
import { MethodsList } from "./methods-list";

export const metadata = {
    title: "Methods - API Documentation",
};

export default function MethodsPage() {
    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <PageHeader
                title='Methods'
                description='Browse all available RPC methods in the API'
            />
            <MethodsList />
        </main>
    );
}
