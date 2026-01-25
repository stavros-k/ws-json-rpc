import { CardBoxWrapper } from "./card-box-wrapper";
import { CodeWrapper } from "./code-wrapper";
import { CollapsibleCard } from "./collapsible-group";

// Temporary type until we migrate to routes
type ExampleData = {
    title: string;
    description: string;
    params?: string;
    result?: string;
};

type Props = {
    isMethod?: boolean;
    examples?: ExampleData[];
};

export const Examples = ({ examples, isMethod }: Props) => {
    if (!examples || !examples.length) return null;

    return (
        <CardBoxWrapper title='Examples'>
            {examples.map((ex) => (
                <CollapsibleCard
                    key={ex.title}
                    title={ex.title}
                    subtitle={ex.description}>
                    <Content
                        example={ex}
                        isMethod={isMethod}
                    />
                </CollapsibleCard>
            ))}
        </CardBoxWrapper>
    );
};

type ContentProps = {
    example: ExampleData;
    isMethod?: boolean;
};

const Content = ({ example, isMethod }: ContentProps) => {
    return (
        <>
            {example.params && (
                <CodeWrapper
                    label={{ text: "Request Params" }}
                    code={example.params !== "null" ? example.params : null}
                    noCodeMessage='No parameters'
                    lang='json'
                />
            )}
            {example.result && (
                <CodeWrapper
                    label={{
                        text: isMethod ? "Response Result" : "Event Data",
                    }}
                    code={example.result !== "null" ? example.result : null}
                    noCodeMessage={isMethod ? "No result" : "No data"}
                    lang='json'
                />
            )}
        </>
    );
};
