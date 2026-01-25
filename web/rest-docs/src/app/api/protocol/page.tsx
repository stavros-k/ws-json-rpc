import { CardBoxWrapper } from "@/components/card-box-wrapper";
import { CodeWrapper } from "@/components/code-wrapper";

export async function generateMetadata() {
    return {
        title: "JSON-RPC Protocol",
    };
}

const requestExample = JSON.stringify(
    {
        jsonrpc: "2.0",
        id: "550e8400-e29b-41d4-a716-446655440000",
        method: "user.create",
        params: {
            username: "john_doe",
            email: "john@example.com",
        },
    },
    null,
    2
);

const successResponseExample = JSON.stringify(
    {
        jsonrpc: "2.0",
        id: "550e8400-e29b-41d4-a716-446655440000",
        result: {
            id: "123e4567-e89b-12d3-a456-426614174000",
            username: "john_doe",
            email: "john@example.com",
        },
    },
    null,
    2
);

const errorResponseExample = JSON.stringify(
    {
        jsonrpc: "2.0",
        id: "550e8400-e29b-41d4-a716-446655440000",
        error: {
            code: -32602,
            message: "Invalid params",
            data: {
                field: "email",
                reason: "Invalid email format",
            },
        },
    },
    null,
    2
);

const eventExample = JSON.stringify(
    {
        event: "data.created",
        data: {
            id: "123e4567-e89b-12d3-a456-426614174000",
            username: "john_doe",
            timestamp: "2024-01-21T10:30:00Z",
        },
    },
    null,
    2
);

export default function ProtocolPage() {
    return (
        <main className='flex-1 p-10 overflow-y-auto'>
            <div>
                <h1 className='text-4xl font-bold mb-3 text-text-primary'>JSON-RPC Protocol</h1>
                <h2 className='text-xl text-text-primary mb-4'>Message envelope structure</h2>

                <div className='text-text-tertiary mb-8 pb-6 border-b-2 border-border-primary'>
                    <p className='mb-4'>
                        This API follows the{" "}
                        <a
                            href='https://www.jsonrpc.org/specification'
                            target='_blank'
                            rel='noopener noreferrer'
                            className='text-accent-blue hover:underline'>
                            JSON-RPC 2.0 specification
                        </a>{" "}
                        for all client-server communication. All messages are JSON objects with specific required
                        fields.
                    </p>
                </div>
            </div>

            {/* Request Format */}
            <CardBoxWrapper title='Request Format'>
                <div className='mb-4'>
                    <p className='text-text-secondary mb-4'>
                        All requests from the client must include the following fields:
                    </p>
                    <ul className='list-disc pl-6 space-y-2 text-text-secondary mb-6'>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>jsonrpc</code>: Must be exactly{" "}
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>"2.0"</code>
                        </li>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>id</code>: A UUID that uniquely
                            identifies this request. The response will include the same ID.
                        </li>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>method</code>: The name of the
                            method to invoke (e.g., "user.create", "ping")
                        </li>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>params</code>: (Optional) An
                            object containing the method parameters
                        </li>
                    </ul>
                </div>
                <CodeWrapper
                    code={requestExample}
                    label={{ text: "Request Example" }}
                    lang='json'
                />
            </CardBoxWrapper>

            {/* Success Response Format */}
            <CardBoxWrapper title='Success Response Format'>
                <div className='mb-4'>
                    <p className='text-text-secondary mb-4'>When a request succeeds, the server responds with:</p>
                    <ul className='list-disc pl-6 space-y-2 text-text-secondary mb-6'>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>jsonrpc</code>: Must be exactly{" "}
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>"2.0"</code>
                        </li>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>id</code>: The same UUID from the
                            request
                        </li>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>result</code>: The result data
                            from the method execution
                        </li>
                    </ul>
                </div>
                <CodeWrapper
                    code={successResponseExample}
                    label={{ text: "Success Response Example" }}
                    lang='json'
                />
            </CardBoxWrapper>

            {/* Error Response Format */}
            <CardBoxWrapper title='Error Response Format'>
                <div className='mb-4'>
                    <p className='text-text-secondary mb-4'>When a request fails, the server responds with:</p>
                    <ul className='list-disc pl-6 space-y-2 text-text-secondary mb-6'>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>jsonrpc</code>: Must be exactly{" "}
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>"2.0"</code>
                        </li>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>id</code>: The same UUID from the
                            request
                        </li>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>error</code>: An error object
                            containing:
                            <ul className='list-disc pl-6 mt-2 space-y-1'>
                                <li>
                                    <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>code</code>: An integer
                                    error code
                                </li>
                                <li>
                                    <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>message</code>: A
                                    human-readable error description
                                </li>
                                <li>
                                    <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>data</code>: (Optional)
                                    Additional error details
                                </li>
                            </ul>
                        </li>
                    </ul>
                </div>
                <CodeWrapper
                    code={errorResponseExample}
                    label={{ text: "Error Response Example" }}
                    lang='json'
                />

                <div className='mt-6 p-4 bg-bg-tertiary rounded-lg border-2 border-border-primary'>
                    <h4 className='font-bold text-text-primary mb-3'>Standard Error Codes</h4>
                    <table className='w-full text-sm'>
                        <thead>
                            <tr className='border-b-2 border-border-primary'>
                                <th className='text-left py-2 text-text-primary'>Code</th>
                                <th className='text-left py-2 text-text-primary'>Meaning</th>
                            </tr>
                        </thead>
                        <tbody className='text-text-secondary'>
                            <tr className='border-b border-border-primary'>
                                <td className='py-2'>
                                    <code className='bg-bg-primary px-2 py-1 rounded text-sm'>-32700</code>
                                </td>
                                <td className='py-2'>Parse error - Invalid JSON received</td>
                            </tr>
                            <tr className='border-b border-border-primary'>
                                <td className='py-2'>
                                    <code className='bg-bg-primary px-2 py-1 rounded text-sm'>-32600</code>
                                </td>
                                <td className='py-2'>Invalid request - Not a valid request object</td>
                            </tr>
                            <tr className='border-b border-border-primary'>
                                <td className='py-2'>
                                    <code className='bg-bg-primary px-2 py-1 rounded text-sm'>-32601</code>
                                </td>
                                <td className='py-2'>Method not found - The method does not exist</td>
                            </tr>
                            <tr className='border-b border-border-primary'>
                                <td className='py-2'>
                                    <code className='bg-bg-primary px-2 py-1 rounded text-sm'>-32602</code>
                                </td>
                                <td className='py-2'>Invalid params - Invalid method parameters</td>
                            </tr>
                            <tr>
                                <td className='py-2'>
                                    <code className='bg-bg-primary px-2 py-1 rounded text-sm'>-32603</code>
                                </td>
                                <td className='py-2'>Internal error - Internal JSON-RPC error</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </CardBoxWrapper>

            {/* Event Format (WebSocket only) */}
            <CardBoxWrapper title='Event Format (WebSocket Only)'>
                <div className='mb-4'>
                    <p className='text-text-secondary mb-4'>
                        When using WebSocket connections, the server can push events to subscribed clients. Events have
                        a different format from requests and responses:
                    </p>
                    <ul className='list-disc pl-6 space-y-2 text-text-secondary mb-6'>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>event</code>: The name of the
                            event (e.g., "data.created", "data.updated")
                        </li>
                        <li>
                            <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>data</code>: The event payload
                        </li>
                    </ul>
                    <p className='text-text-secondary mb-4'>
                        Note: Events do not have an <code className='bg-bg-tertiary px-2 py-1 rounded text-sm'>id</code>{" "}
                        field because they are server-initiated messages, not responses to client requests.
                    </p>
                </div>
                <CodeWrapper
                    code={eventExample}
                    label={{ text: "Event Example" }}
                    lang='json'
                />
            </CardBoxWrapper>
        </main>
    );
}
