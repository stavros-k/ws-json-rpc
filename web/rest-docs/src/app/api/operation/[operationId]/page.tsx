import type { Route } from "next";
import { CardBoxWrapper } from "@/components/card-box-wrapper";
import { CodeWrapper } from "@/components/code-wrapper";
import { CollapsibleCard } from "@/components/collapsible-group";
import { Deprecation } from "@/components/deprecation";
import { GroupAndTags } from "@/components/group-and-tags";
import { getAllOperations, getTypeJson, type TypeKeys } from "@/data/api";

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
            <div>
                <h1 className='text-4xl font-bold mb-3 text-text-primary'>{operation.operationID}</h1>
                <h2 className='text-xl text-accent-blue mb-4 font-mono font-semibold'>
                    {operation.verb.toUpperCase()} {operation.route}
                </h2>

                <Deprecation
                    deprecated={{ message: operation.deprecated }}
                    itemType='operation'
                />

                <div className='text-text-tertiary mb-8 pb-6 border-b-2 border-border-primary'>
                    <p className='mb-2'>{operation.summary}</p>
                    {operation.description && operation.description !== operation.summary && (
                        <p className='text-sm'>{operation.description}</p>
                    )}
                    <GroupAndTags
                        group={operation.group || ""}
                        tags={operation.tags}
                    />
                </div>
            </div>

            {operation.parameters && operation.parameters.length > 0 && (
                <CardBoxWrapper title='Parameters'>
                    <div className='space-y-3'>
                        {operation.parameters.map((param) => (
                            <div
                                key={param.name}
                                className='p-4 rounded-lg bg-bg-tertiary border-2 border-border-primary'>
                                <div className='flex items-start justify-between gap-4 mb-2'>
                                    <div className='flex items-center gap-2 flex-wrap'>
                                        <code className='text-base font-semibold text-text-primary'>{param.name}</code>
                                        {param.required && (
                                            <span className='text-xs px-2 py-0.5 rounded bg-red-500/20 text-red-400 border border-red-500/30 font-semibold'>
                                                required
                                            </span>
                                        )}
                                        <span className='text-xs px-2 py-0.5 rounded bg-blue-500/20 text-blue-400 border border-blue-500/30'>
                                            {param.in}
                                        </span>
                                    </div>
                                    <code className='px-3 py-1.5 rounded-lg bg-type-primitive/10 text-type-primitive border-2 border-type-primitive/30 font-mono text-sm font-semibold'>
                                        {param.type}
                                    </code>
                                </div>
                                {param.description && <p className='text-sm text-text-tertiary'>{param.description}</p>}
                            </div>
                        ))}
                    </div>
                </CardBoxWrapper>
            )}

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
                            const responseJson = getTypeJson(response.type as TypeKeys);
                            const statusBadge = (
                                <span
                                    className={`text-lg font-bold px-3 py-1.5 rounded-lg font-mono ${
                                        statusCode.startsWith("2")
                                            ? "bg-green-500/20 text-green-400 border-2 border-green-500/30"
                                            : statusCode.startsWith("4")
                                              ? "bg-yellow-500/20 text-yellow-400 border-2 border-yellow-500/30"
                                              : "bg-red-500/20 text-red-400 border-2 border-red-500/30"
                                    }`}>
                                    {statusCode}
                                </span>
                            );

                            return (
                                <CollapsibleCard
                                    key={statusCode}
                                    title={statusCode}
                                    subtitle={response.description}
                                    defaultOpen={false}>
                                    <CodeWrapper
                                        label={{
                                            text: response.type,
                                            href: `/api/type/${response.type}` as Route,
                                        }}
                                        code={responseJson}
                                        noCodeMessage='No response body'
                                        lang='json'
                                    />
                                    {response.examples && Object.keys(response.examples).length > 0 && (
                                        <div className='mt-6 space-y-3'>
                                            {Object.entries(response.examples).map(([exampleKey, exampleValue]) => (
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
                                </CollapsibleCard>
                            );
                        })}
                    </div>
                </CardBoxWrapper>
            )}
        </main>
    );
}
