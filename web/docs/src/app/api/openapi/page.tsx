import { Breadcrumbs } from "@/components/breadcrumbs";
import { CardBoxWrapper } from "@/components/card-box-wrapper";
import { CodeWrapper } from "@/components/code-wrapper";
import { docs } from "@/data/api";

export async function generateMetadata() {
    return {
        title: "OpenAPI Specification",
    };
}

export default function OpenAPISpec() {
    const hasSpec = docs.openapiSpec && docs.openapiSpec.trim() !== "";

    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <Breadcrumbs items={[{ label: "OpenAPI Specification" }]} />

            <div>
                <h1 className='text-4xl font-bold mb-3 text-text-primary'>OpenAPI Specification</h1>
                <h2 className='text-xl text-text-primary mb-4'>Complete API specification in OpenAPI 3.0 format</h2>

                <div className='text-text-tertiary mb-8 pb-6 border-b-2 border-border-primary'>
                    <p>
                        This page displays the complete OpenAPI 3.0 specification for the API.
                        You can use this specification with tools like Swagger UI, Postman, or other
                        OpenAPI-compatible clients.
                    </p>
                </div>
            </div>

            {hasSpec ? (
                <CardBoxWrapper title='OpenAPI 3.0 YAML'>
                    <CodeWrapper
                        code={docs.openapiSpec}
                        label={{ text: "openapi.yaml" }}
                        lang='yaml'
                    />
                </CardBoxWrapper>
            ) : (
                <div className='p-6 bg-bg-secondary rounded-xl border-2 border-border-primary text-text-secondary'>
                    No OpenAPI specification available.
                </div>
            )}
        </main>
    );
}
