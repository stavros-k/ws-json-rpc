import type { Route } from "next";
import { BackButton } from "@/components/back-button";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { CardBoxWrapper } from "@/components/card-box-wrapper";
import { CodeWrapper } from "@/components/code-wrapper";
import { CollapsibleCard } from "@/components/collapsible-group";
import { CollapsibleResponse } from "@/components/collapsible-response";
import { Deprecation } from "@/components/deprecation";
import { Group } from "@/components/group";
import { OperationHeader } from "@/components/operation-header";
import { getAllOperations, getTypeJson, type Response, type TypeKeys } from "@/data/api";

export function generateStaticParams() {
    const operations = getAllOperations();
    return operations.map((operation) => ({
        operationId: operation.operationID,
    }));
}

export async function generateMetadata(props: PageProps<"/api/operation/[operationId]">) {
    const params = await props.params;
    const { operationId } = params;
    return {
        title: `Operation - [${operationId}]`,
    };
}

export default async function OperationPage(props: PageProps<"/api/operation/[operationId]">) {
    const params = await props.params;
    const { operationId } = params;

    const allOperations = getAllOperations();
    const operation = allOperations.find((op) => op.operationID === operationId);

    if (!operation) {
        return <div>Operation not found</div>;
    }

    const requestJson = operation.request ? getTypeJson(operation.request.type as TypeKeys) : null;

    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <Breadcrumbs
                items={[{ label: "Operations", href: "/api/operations" as Route }, { label: operation.operationID }]}
            />

            <BackButton
                href='/api/operations'
                label='Operations'
            />

            <div>
                <div className='flex items-center justify-between gap-3 mb-3'>
                    <h1 className='text-4xl font-bold text-text-primary'>{operation.operationID}</h1>
                    <Group
                        group={operation.group || ""}
                        size='md'
                    />
                </div>

                <Deprecation
                    deprecated={operation.deprecated}
                    itemType='operation'
                />

                <div className='text-text-tertiary mb-8 pb-6 border-b-2 border-border-primary'>
                    <p className='mb-2'>{operation.summary}</p>
                    {operation.description && operation.description !== operation.summary && (
                        <p className='text-sm'>{operation.description}</p>
                    )}
                </div>
            </div>

            <OperationHeader
                method={operation.method}
                path={operation.path}
                parameters={operation.parameters}
            />

            {operation.request && (
                <CardBoxWrapper title='Request Body'>
                    <CodeWrapper
                        label={{
                            text: operation.request.type,
                            href: `/api/type/${operation.request.type}` as Route,
                        }}
                        code={requestJson}
                        noCodeMessage='No request body'
                        lang='json'
                    />
                    {operation.request.description && (
                        <p className='text-sm text-text-tertiary mt-3'>{operation.request.description}</p>
                    )}
                </CardBoxWrapper>
            )}

            {operation.responses && Object.keys(operation.responses).length > 0 && (
                <CardBoxWrapper title='Responses'>
                    <div className='space-y-4'>
                        {Object.entries(operation.responses).map(([statusCode, response]) => {
                            const resp = response as Response;
                            const responseJson = getTypeJson(resp.type as TypeKeys);

                            return (
                                <CollapsibleResponse
                                    key={statusCode}
                                    statusCode={statusCode}
                                    description={resp.description}>
                                    <CodeWrapper
                                        label={{
                                            text: resp.type,
                                            href: `/api/type/${resp.type}` as Route,
                                        }}
                                        code={responseJson}
                                        noCodeMessage='No response body'
                                        lang='json'
                                    />
                                    {resp.examples && Object.keys(resp.examples).length > 0 && (
                                        <div className='mt-6 space-y-3'>
                                            {Object.entries(resp.examples).map(([exampleKey, exampleValue]) => (
                                                <CollapsibleCard
                                                    key={exampleKey}
                                                    title={exampleKey}
                                                    defaultOpen={false}>
                                                    <CodeWrapper
                                                        label={{ text: "Example" }}
                                                        code={exampleValue}
                                                        lang='json'
                                                    />
                                                </CollapsibleCard>
                                            ))}
                                        </div>
                                    )}
                                </CollapsibleResponse>
                            );
                        })}
                    </div>
                </CardBoxWrapper>
            )}
        </main>
    );
}
