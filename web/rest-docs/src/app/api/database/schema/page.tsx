import { CardBoxWrapper } from "@/components/card-box-wrapper";
import { CodeWrapper } from "@/components/code-wrapper";
import { docs } from "@/data/api";
export async function generateMetadata() {
    return {
        title: "Database Schema",
    };
}

export default function DatabaseSchema() {
    const tableCount = docs.database.tableCount || 0;

    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <div>
                <h1 className='text-4xl font-bold mb-3 text-text-primary'>Database Schema</h1>
                <h2 className='text-xl text-text-primary mb-4'>Database structure definition</h2>

                <div className='text-text-tertiary mb-8 pb-6 border-b-2 border-border-primary'>
                    <p>This page displays the database schema used by the application.</p>
                </div>
            </div>

            <div className='mb-8'>
                <div className='inline-block p-6 bg-bg-secondary rounded-xl border-2 border-border-primary'>
                    <div className='flex items-center gap-4'>
                        <div className='text-center'>
                            <div className='text-4xl font-bold text-accent-blue mb-1'>{tableCount}</div>
                            <div className='text-sm text-text-secondary font-semibold'>
                                {tableCount === 1 ? "Table" : "Tables"}
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <CardBoxWrapper title='Schema'>
                <CodeWrapper
                    code={docs.database.schema}
                    label={{ text: "schema.sql" }}
                    lang='sql'
                />
            </CardBoxWrapper>
        </main>
    );
}
