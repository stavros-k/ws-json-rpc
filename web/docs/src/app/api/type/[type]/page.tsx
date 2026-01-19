import { BsFileEarmarkCode } from "react-icons/bs";
import { TbInfoCircle, TbJson } from "react-icons/tb";
import { CodeWrapper } from "@/components/code-wrapper";
import { TabbedCardWrapper } from "@/components/tabbed-card-wrapper-client";
import { TypeMetadata } from "@/components/type-metadata";
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
            <div>
                <h1 className='text-4xl font-bold mb-3 text-text-primary'>{type}</h1>

                <div className='text-text-tertiary mb-8 pb-6 border-b-2 border-border-primary'>
                    <p>{data.description}</p>
                </div>
            </div>

            <TabbedCardWrapper
                tabs={[
                    {
                        title: "Overview",
                        icon: <TbInfoCircle className='w-8 h-8 text-accent-primary' />,
                        code: (
                            <TypeMetadata
                                typeName={type}
                                data={data}
                            />
                        ),
                    },
                    {
                        title: "JSON",
                        icon: <TbJson className='w-8 h-8 text-lang-json' />,
                        code: (
                            <CodeWrapper
                                code={data.jsonRepresentation}
                                lang='json'
                                label={{ text: type }}
                            />
                        ),
                    },
                    {
                        title: "JSON Schema",
                        icon: <BsFileEarmarkCode className='w-8 h-8 text-lang-schema' />,
                        code: (
                            <CodeWrapper
                                code={data.jsonSchema}
                                lang='json'
                                label={{ text: type }}
                            />
                        ),
                    },
                ]}
            />
        </main>
    );
}
