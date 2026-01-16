import type { Route } from "next";
import { CardBoxWrapper } from "@/components/card-box-wrapper";
import { CodeWrapper } from "@/components/code-wrapper";
import { Deprecation } from "@/components/deprecation";
import { Errors } from "@/components/errors";
import { Examples } from "@/components/examples";
import { GroupAndTags } from "@/components/group-and-tags";
import { MethodCaller } from "@/components/method-caller";
import { ProtocolBadge } from "@/components/protocol-badge";
import { docs, getTypeJson, type MethodKeys, type TypeKeys } from "@/data/api";

export function generateStaticParams() {
    return Object.keys(docs.methods).map((method) => ({
        method: method,
    }));
}

export async function generateMetadata(props: PageProps<"/api/method/[method]">) {
    const params = await props.params;
    const { method } = params as { method: MethodKeys };
    return {
        title: `Method - [${method}]`,
    };
}

export default async function Method(props: PageProps<"/api/method/[method]">) {
    const params = await props.params;
    const { method } = params as { method: MethodKeys };
    const data = docs.methods[method];
    const resultJson = getTypeJson(data.resultType.$ref as TypeKeys);
    const paramsJson = getTypeJson(data.paramType.$ref as TypeKeys);
    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <div>
                <h1 className='text-4xl font-bold mb-3 text-text-primary'>{method}</h1>
                <h2 className='text-xl text-text-primary mb-4'>{data.title}</h2>

                <Deprecation
                    type='method'
                    deprecated={data.deprecated}
                />

                <div className='flex gap-2 mb-4'>
                    <ProtocolBadge
                        title='WebSocket'
                        supported={data.protocols.ws}
                    />
                    <ProtocolBadge
                        title='HTTP'
                        supported={data.protocols.http}
                    />
                </div>

                <div className='text-text-tertiary mb-8 pb-6 border-b-2 border-border-primary'>
                    <p>{data.description}</p>
                    <GroupAndTags
                        group={data.group}
                        tags={data.tags}
                    />
                </div>
            </div>

            <CardBoxWrapper title='Parameters'>
                <CodeWrapper
                    label={{
                        text: data.paramType.$ref,
                        href: `/api/type/${data.paramType.$ref}` as Route,
                    }}
                    code={paramsJson}
                    noCodeMessage='No parameters'
                    lang='json'
                />
            </CardBoxWrapper>

            <CardBoxWrapper title='Result'>
                <CodeWrapper
                    label={{
                        text: data.resultType.$ref,
                        href: `/api/type/${data.resultType.$ref}` as Route,
                    }}
                    code={resultJson}
                    noCodeMessage='No result'
                    lang='json'
                />
            </CardBoxWrapper>

            <Examples
                examples={data.examples}
                isMethod={true}
            />
            <Errors errors={data.errors} />

            {data.protocols.ws && (
                <MethodCaller
                    methodName={method}
                    defaultParams={paramsJson || "{}"}
                />
            )}
        </main>
    );
}
