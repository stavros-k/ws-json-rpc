import { CardBoxWrapper } from "@/components/card-box-wrapper";
import { CodeWrapper } from "@/components/code-wrapper";
import { docs } from "@/data/api";
export async function generateMetadata() {
    return {
        title: "Database Schema",
    };
}

export default function DatabaseSchema() {
    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <div>
                <h1 className='text-4xl font-bold mb-3 text-text-primary'>
                    Database Schema
                </h1>
                <h2 className='text-xl text-text-primary mb-4'>
                    Database structure definition
                </h2>

                <div className='text-text-tertiary mb-8 pb-6 border-b-2 border-border-primary'>
                    <p>
                        This page displays the database schema used by the
                        application.
                    </p>
                </div>
            </div>

            <CardBoxWrapper title='Schema'>
                <CodeWrapper
                    code={docs.databaseSchema}
                    label={{ text: "schema.sql" }}
                    lang='sql'
                />
            </CardBoxWrapper>
        </main>
    );
}
