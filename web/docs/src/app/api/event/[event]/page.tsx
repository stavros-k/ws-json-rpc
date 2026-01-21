import type { Route } from "next";
import { CardBoxWrapper } from "@/components/card-box-wrapper";
import { CodeWrapper } from "@/components/code-wrapper";
import { Deprecation } from "@/components/deprecation";
import { EventSubscriber } from "@/components/event-subscriber";
import { Examples } from "@/components/examples";
import { GroupAndTags } from "@/components/group-and-tags";
import { ProtocolBadge } from "@/components/protocol-badge";
import { docs, type EventKeys, getTypeJson, type TypeKeys } from "@/data/api";

export function generateStaticParams() {
    return Object.keys(docs.events).map((event) => ({
        event: event,
    }));
}

export async function generateMetadata(props: PageProps<"/api/event/[event]">) {
    const params = await props.params;
    const { event } = params as { event: EventKeys };
    return {
        title: `Event - [${event}]`,
    };
}

export default async function Event(props: PageProps<"/api/event/[event]">) {
    const params = await props.params;
    const { event } = params as { event: EventKeys };
    const data = docs.events[event];

    const resultJson = getTypeJson(data.resultType.$ref as TypeKeys);

    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <div>
                <h1 className='text-4xl font-bold mb-3 text-text-primary'>{data.title}</h1>
                <h2 className='text-xl text-text-primary mb-4'>{event}</h2>

                <Deprecation
                    type='event'
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

            <Examples examples={data.examples} />

            {data.protocols.ws && (
                <CardBoxWrapper title='Live Event Monitor'>
                    <EventSubscriber eventName={event} />
                </CardBoxWrapper>
            )}
        </main>
    );
}
