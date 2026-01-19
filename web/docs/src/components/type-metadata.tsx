import Link from "next/link";
import { TbCircle, TbCircleCheckFilled } from "react-icons/tb";
import type { TypeData } from "@/data/api";

interface TypeMetadataProps {
    typeName: string;
    data: TypeData;
}

// Format descriptions for common string formats
const FORMAT_DESCRIPTIONS: Record<string, Array<string>> = {
    uuid: [
        "UUID (Universally Unique Identifier) - Version 4",
        "Typically formatted as 36 characters (e.g., 550e8400-e29b-41d4-a716-446655440000)",
    ],
    "date-time": [
        "Date-Time - A string representing date and time in a specific format.",
        "RFC 3339 format (e.g., 2025-01-23T15:30:00Z).",
        "TypeScript: Use 'new Date(...)' to parse.",
        "C#: Automatically handled by DateTime type deserialization.",
    ],
};

export function TypeMetadata({ typeName, data }: TypeMetadataProps) {
    const hasFields = "fields" in data && data.fields && data.fields.length > 0;
    const hasReferences = "references" in data && data.references && data.references.length > 0;
    const hasReferencedBy = "referencedBy" in data && data.referencedBy && data.referencedBy.length > 0;
    const hasJsonSchema = "jsonSchema" in data && data.jsonSchema;

    return (
        <div className='space-y-6'>
            {/* JSON Schema Section */}
            {hasJsonSchema && (
                <div>
                    <h2 className='text-xl font-semibold mb-4 text-text-primary'>JSON Schema</h2>
                    <pre className='p-4 rounded-lg bg-background-secondary border border-border-primary overflow-x-auto'>
                        <code className='text-sm text-text-secondary'>{JSON.stringify(JSON.parse(data.jsonSchema), null, 2)}</code>
                    </pre>
                </div>
            )}

            {/* Fields Section (for objects) */}
            {hasFields && (
                <div>
                    <h2 className='text-xl font-semibold mb-4 text-text-primary'>Fields</h2>
                    <div className='space-y-3'>
                        {data.fields.map((field) => (
                            <div
                                key={field.name}
                                className='p-4 rounded-lg bg-background-secondary border border-border-primary'>
                                <div className='flex items-start justify-between mb-2'>
                                    <div className='flex items-center gap-2'>
                                        <code className='text-base font-mono font-semibold text-type-field-name'>
                                            {field.name}
                                        </code>
                                        <div className='flex items-center gap-2'>
                                            {field.optional ? (
                                                <span className='flex items-center gap-1 text-xs px-2 py-0.5 rounded bg-gray-500/10 text-gray-500'>
                                                    <TbCircle className='w-3 h-3' />
                                                    Optional
                                                </span>
                                            ) : (
                                                <span className='flex items-center gap-1 text-xs px-2 py-0.5 rounded bg-red-500/10 text-red-500'>
                                                    <TbCircleCheckFilled className='w-3 h-3' />
                                                    Required
                                                </span>
                                            )}
                                            {field.nullable && (
                                                <span className='text-xs px-2 py-0.5 rounded bg-blue-500/10 text-blue-500'>
                                                    Nullable
                                                </span>
                                            )}
                                        </div>
                                    </div>
                                    <div className='flex items-center gap-2'>
                                        {field.isRef ? (
                                            <Link
                                                href={`/api/type/${field.type}`}
                                                className='font-mono text-base font-semibold text-type-reference hover:text-type-reference-hover hover:underline'>
                                                {field.type}
                                            </Link>
                                        ) : (
                                            <code className='font-mono text-base font-semibold text-type-primitive'>
                                                {field.type}
                                            </code>
                                        )}
                                        {"format" in field && field.format && (
                                            <span className='text-xs px-2 py-0.5 rounded bg-purple-500/10 text-purple-400 border border-purple-500/20 font-mono'>
                                                {field.format}
                                            </span>
                                        )}
                                    </div>
                                </div>
                                {field.description && (
                                    <p className='text-sm text-text-tertiary mt-2'>{field.description}</p>
                                )}
                                {"format" in field && field.format && FORMAT_DESCRIPTIONS[field.format] && (
                                    <div className='mt-2 p-2 rounded bg-purple-500/5 border border-purple-500/10'>
                                        <p className='text-xs text-purple-300'>
                                            {FORMAT_DESCRIPTIONS[field.format].map((line) => (
                                                <span key={line}>
                                                    {line}
                                                    <br />
                                                </span>
                                            ))}
                                        </p>
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* References Section */}
            {hasReferences && (
                <div>
                    <h2 className='text-xl font-semibold mb-4 text-text-primary'>References</h2>
                    <p className='text-sm text-text-tertiary mb-3'>Types used by {typeName}:</p>
                    <div className='flex flex-wrap gap-2'>
                        {data.references.map((ref) => (
                            <Link
                                key={ref}
                                href={`/api/type/${ref}`}
                                className='px-3 py-1.5 rounded-md bg-type-object-bg text-type-object hover:brightness-110 transition-colors font-mono text-sm border border-type-object-border'>
                                {ref}
                            </Link>
                        ))}
                    </div>
                </div>
            )}

            {/* Referenced By Section (Back-references) */}
            {hasReferencedBy && (
                <div>
                    <h2 className='text-xl font-semibold mb-4 text-text-primary'>Referenced By</h2>
                    <p className='text-sm text-text-tertiary mb-3'>Types that use {typeName}:</p>
                    <div className='flex flex-wrap gap-2'>
                        {data.referencedBy.map((ref) => (
                            <Link
                                key={ref}
                                href={`/api/type/${ref}`}
                                className='px-3 py-1.5 rounded-md bg-type-enum-bg text-type-enum hover:brightness-110 transition-colors font-mono text-sm border border-type-enum-border'>
                                {ref}
                            </Link>
                        ))}
                    </div>
                </div>
            )}

            {/* Empty State */}
            {!hasJsonSchema && !hasFields && !hasReferences && !hasReferencedBy && (
                <div className='text-center py-8 text-text-tertiary'>
                    <p>No additional metadata available for this type.</p>
                </div>
            )}
        </div>
    );
}
