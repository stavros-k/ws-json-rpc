import Link from "next/link";
import type { TypeData, FieldMetadata } from "@/data/api";
import { docs } from "@/data/api";

interface TypeMetadataProps {
    data: TypeData;
    typeName: string;
}

// Helper function to check if a type name exists in the docs
const isTypeLink = (type: string) => {
    return type in docs.types;
};

function TypeKindBadge({ kind }: { kind: string }) {
    return (
        <div>
            <span className='inline-block px-3 py-1.5 rounded-md bg-info-bg text-info-text border border-info-border font-semibold text-sm'>
                {kind}
            </span>
        </div>
    );
}

function TypeDescription({ description }: { description?: string }) {
    if (!description) return null;

    return (
        <div>
            <h2 className='text-xl font-semibold mb-3 text-text-primary'>Description</h2>
            <p className='text-text-secondary'>{description}</p>
        </div>
    );
}

function UsedBySection({ usedBy }: { usedBy: Array<{ type: string; target: string; role: string }> | null }) {
    if (!usedBy || usedBy.length === 0) return null;

    return (
        <div>
            <h2 className='text-xl font-semibold mb-4 text-text-primary'>Used By</h2>
            <div className='flex flex-wrap gap-3'>
                {usedBy.map((usage) => {
                    const targetUrl = `/api/${usage.type}/${usage.target}` as
                        | "/api/method/[method]"
                        | "/api/event/[event]";
                    const roleLabel = usage.role === "param" ? "Parameter" : "Result";
                    const typeLabel = usage.type === "method" ? "Method" : "Event";
                    const key = `${usage.type}-${usage.target}-${usage.role}`;

                    return (
                        <div
                            key={key}
                            className='flex items-center gap-2'>
                            <Link
                                href={targetUrl}
                                className='px-3 py-1.5 rounded-md bg-purple-500/10 text-purple-300 hover:brightness-110 transition-colors font-mono text-sm border border-purple-500/30'>
                                {typeLabel} | {usage.target}
                            </Link>
                            <span className='text-sm text-text-tertiary'>({roleLabel})</span>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}

function EnumValuesSection({ enumValues }: { enumValues: string[] | null }) {
    if (!enumValues || enumValues.length === 0) return null;

    return (
        <div>
            <h2 className='text-xl font-semibold mb-4 text-text-primary'>Possible Values</h2>
            <div className='flex flex-wrap gap-2'>
                {enumValues.map((value) => (
                    <code
                        key={value}
                        className='px-3 py-1.5 rounded-md bg-type-enum-bg text-type-enum border border-type-enum-border font-mono text-sm'>
                        &quot;{value}&quot;
                    </code>
                ))}
            </div>
        </div>
    );
}

function FieldItem({ field }: { field: FieldMetadata }) {
    return (
        <div className='p-4 rounded-lg bg-bg-secondary border border-border-primary'>
            <div className='flex items-start justify-between mb-2'>
                <div className='flex items-center gap-2'>
                    <code className='text-lg font-semibold text-text-primary'>{field.name}</code>
                    {field.optional && (
                        <span className='text-xs px-2 py-0.5 rounded bg-yellow-500/20 text-yellow-400 border border-yellow-500/30'>
                            optional
                        </span>
                    )}
                </div>
                {isTypeLink(field.type) ? (
                    <Link
                        href={`/api/type/${field.type}`}
                        className='text-sm text-type-reference hover:text-type-reference-hover font-mono underline decoration-dotted'>
                        {field.type}
                    </Link>
                ) : (
                    <code className='text-sm text-type-primitive font-mono'>{field.type}</code>
                )}
            </div>

            {field.description && <p className='text-sm text-text-tertiary mb-2'>{field.description}</p>}

            {"enumValues" in field && field.enumValues && field.enumValues.length > 0 && (
                <div className='mt-3'>
                    <p className='text-xs text-text-tertiary mb-2'>Possible values:</p>
                    <div className='flex flex-wrap gap-2'>
                        {field.enumValues.map((value: string) => (
                            <code
                                key={value}
                                className='px-2 py-1 text-xs rounded bg-type-enum-bg text-type-enum border border-type-enum-border font-mono'>
                                {value}
                            </code>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}

function FieldsSection({ fields }: { fields: FieldMetadata[] | undefined }) {
    if (!fields || fields.length === 0) return null;

    return (
        <div>
            <h2 className='text-xl font-semibold mb-4 text-text-primary'>Fields</h2>
            <div className='space-y-4'>
                {fields.map((field) => (
                    <FieldItem
                        key={field.name}
                        field={field}
                    />
                ))}
            </div>
        </div>
    );
}

function ReferencesSection({ references, typeName }: { references: string[] | undefined; typeName: string }) {
    const hasReferences = references && references.length > 0;

    return (
        <div>
            <h2 className='text-xl font-semibold mb-4 text-text-primary'>References</h2>
            {hasReferences ? (
                <>
                    <p className='text-sm text-text-tertiary mb-3'>Types used by {typeName}:</p>
                    <div className='flex flex-wrap gap-2'>
                        {references.map((ref: string) => (
                            <Link
                                key={ref}
                                href={`/api/type/${ref}`}
                                className='px-3 py-1.5 rounded-md bg-type-object-bg text-type-object hover:brightness-110 transition-colors font-mono text-sm border border-type-object-border'>
                                {ref}
                            </Link>
                        ))}
                    </div>
                </>
            ) : (
                <p className='text-sm text-text-tertiary'>This type does not reference other types.</p>
            )}
        </div>
    );
}

function ReferencedBySection({ referencedBy, typeName }: { referencedBy: string[] | undefined; typeName: string }) {
    const hasReferencedBy = referencedBy && referencedBy.length > 0;

    return (
        <div>
            <h2 className='text-xl font-semibold mb-4 text-text-primary'>Referenced By</h2>
            {hasReferencedBy ? (
                <>
                    <p className='text-sm text-text-tertiary mb-3'>Types that use {typeName}:</p>
                    <div className='flex flex-wrap gap-2'>
                        {referencedBy.map((ref: string) => (
                            <Link
                                key={ref}
                                href={`/api/type/${ref}`}
                                className='px-3 py-1.5 rounded-md bg-type-enum-bg text-type-enum hover:brightness-110 transition-colors font-mono text-sm border border-type-enum-border'>
                                {ref}
                            </Link>
                        ))}
                    </div>
                </>
            ) : (
                <p className='text-sm text-text-tertiary'>No types reference this type.</p>
            )}
        </div>
    );
}

export function TypeMetadata({ data, typeName }: TypeMetadataProps) {
    const fields = "fields" in data ? data.fields : undefined;
    const isPrimitiveOrEnum = !fields || fields.length === 0;

    const enumValues =
        "enumValues" in data && Array.isArray(data.enumValues) && data.enumValues.length > 0 ? data.enumValues : null;
    const usedBy = "usedBy" in data && Array.isArray(data.usedBy) && data.usedBy.length > 0 ? data.usedBy : null;
    const references = "references" in data ? data.references : undefined;
    const referencedBy = "referencedBy" in data ? data.referencedBy : undefined;

    return (
        <div className='space-y-6'>
            <TypeKindBadge kind={data.kind} />
            <TypeDescription description={data.description} />

            {isPrimitiveOrEnum && (
                <>
                    <div className='border-t border-border-primary' />
                    <EnumValuesSection enumValues={enumValues} />
                </>
            )}

            {fields && fields.length > 0 && (
                <>
                    <div className='border-t border-border-primary' />
                    <FieldsSection fields={fields} />
                </>
            )}

            {usedBy && usedBy.length > 0 && (
                <>
                    <div className='border-t border-border-primary' />
                    <UsedBySection usedBy={usedBy} />
                </>
            )}

            {references && references.length > 0 && (
                <>
                    <div className='border-t border-border-primary' />
                    <ReferencesSection
                        references={references}
                        typeName={typeName}
                    />
                </>
            )}

            {referencedBy && referencedBy.length > 0 && (
                <>
                    <div className='border-t border-border-primary' />
                    <ReferencedBySection
                        referencedBy={referencedBy}
                        typeName={typeName}
                    />
                </>
            )}
        </div>
    );
}
