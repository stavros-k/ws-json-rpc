import type { Route } from "next";
import { TbBrandTypescript, TbInfoCircle, TbJson, TbFileCode } from "react-icons/tb";
import { CodeWrapper } from "@/components/code-wrapper";
import { Deprecation } from "@/components/deprecation";
import { TabbedCardWrapper } from "@/components/tabbed-card-wrapper-client";
import { TypeMetadata } from "@/components/type-metadata";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { BackButton } from "@/components/back-button";
import { docs, type TypeKeys } from "@/data/api";

export function generateStaticParams() {
    return Object.keys(docs.types).map((type) => ({
        type: type,
    }));
}

export async function generateMetadata(props: PageProps<"/api/type/[type]">) {
    const params = await props.params;
    const { type } = params as { type: TypeKeys };
    return {
        title: `Type - [${type}]`,
    };
}

export default async function Type(props: PageProps<"/api/type/[type]">) {
    const params = await props.params;
    const { type } = params as { type: TypeKeys };
    const data = docs.types[type];

    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <Breadcrumbs
                items={[
                    { label: "Types", href: "/api/types" as Route },
                    { label: type },
                ]}
            />

            <BackButton
                href='/api/types'
                label='Types'
            />

            <div>
                <h1 className='text-4xl font-bold mb-3 text-text-primary'>{type}</h1>

                <Deprecation deprecated={data.deprecated} />

                <div className='text-text-tertiary mb-8 pb-6 border-b-2 border-border-primary'>
                    <p>{data.description}</p>
                </div>
            </div>

            <TabbedCardWrapper
                tabs={[
                    {
                        title: "Overview",
                        icon: <TbInfoCircle className='w-8 h-8 text-lang-overview' />,
                        code: (
                            <TypeMetadata
                                data={data}
                                typeName={type}
                            />
                        ),
                    },
                    {
                        title: "TypeScript",
                        icon: <TbBrandTypescript className='w-8 h-8 text-lang-ts' />,
                        code:
                            "representations" in data && data.representations?.ts ? (
                                <CodeWrapper
                                    code={data.representations.ts}
                                    lang='typescript'
                                    label={{ text: type }}
                                />
                            ) : (
                                <p className='text-sm text-text-tertiary p-4'>
                                    No TypeScript representation available for this type.
                                </p>
                            ),
                    },
                    {
                        title: "JSON Example",
                        icon: <TbFileCode className='w-8 h-8 text-lang-json' />,
                        code:
                            "representations" in data && data.representations?.json && data.representations.json.trim() ? (
                                <CodeWrapper
                                    code={data.representations.json}
                                    lang='json'
                                    label={{ text: type }}
                                />
                            ) : (
                                <p className='text-sm text-text-tertiary p-4'>
                                    No JSON example available for this type.
                                </p>
                            ),
                    },
                    {
                        title: "JSON Schema",
                        icon: <TbJson className='w-8 h-8 text-purple-400' />,
                        code:
                            "representations" in data && data.representations?.jsonSchema ? (
                                <CodeWrapper
                                    code={data.representations.jsonSchema}
                                    lang='json'
                                    label={{ text: type }}
                                />
                            ) : (
                                <p className='text-sm text-text-tertiary p-4'>
                                    No JSON schema available for this type.
                                </p>
                            ),
                    },
                ]}
            />
        </main>
    );
}
