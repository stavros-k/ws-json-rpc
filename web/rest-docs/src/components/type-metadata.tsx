import type { Route } from "next";
import Link from "next/link";
import { BiLinkExternal } from "react-icons/bi";
import type { FieldMetadata, TypeData } from "@/data/api";
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

type UsedByItem = {
    operationID: string;
    role: string;
};

function UsedBySection({ usedBy }: { usedBy: UsedByItem[] | null }) {
    if (!usedBy || usedBy.length === 0) return null;

    // Group by operationID and collect roles
    const grouped = usedBy.reduce(
        (acc, item) => {
            if (!acc[item.operationID]) {
                acc[item.operationID] = new Set();
            }
            acc[item.operationID].add(item.role);
            return acc;
        },
        {} as Record<string, Set<string>>
    );

    return (
        <div>
            <h2 className='text-xl font-semibold mb-4 text-text-primary'>Used By Operations</h2>
            <p className='text-sm text-text-tertiary mb-4'>This type is used by the following API operations:</p>
            <div className='grid grid-cols-1 md:grid-cols-2 gap-3'>
                {Object.entries(grouped).map(([operationID, roles]) => (
                    <Link
                        key={operationID}
                        href={`/api/operation/${operationID}` as Route}
                        className='block p-4 rounded-lg bg-bg-tertiary border-2 border-border-primary hover:border-accent-blue transition-all duration-200 hover:shadow-md'>
                        <div className='flex items-start justify-between gap-3'>
                            <code className='font-mono text-sm font-semibold text-text-primary break-all hover:text-accent-blue transition-colors'>
                                {operationID}
                            </code>
                            <div className='flex flex-wrap gap-1 shrink-0'>
                                {Array.from(roles).map((role) => (
                                    <span
                                        key={role}
                                        className={`text-xs px-2 py-0.5 rounded font-medium ${
                                            role === "request"
                                                ? "bg-blue-500/20 text-blue-400 border border-blue-500/30"
                                                : "bg-green-500/20 text-green-400 border border-green-500/30"
                                        }`}>
                                        {role}
                                    </span>
                                ))}
                            </div>
                        </div>
                    </Link>
                ))}
            </div>
        </div>
    );
}

type EnumValue = {
    value: string;
    description: string;
    deprecated: string | null;
};

function EnumValuesSection({ enumValues }: { enumValues: EnumValue[] | null }) {
    if (!enumValues || enumValues.length === 0) return null;

    return (
        <div>
            <h2 className='text-xl font-semibold mb-4 text-text-primary'>Possible Values</h2>
            <div className='flex flex-col gap-4'>
                {enumValues.map((enumValue) => (
                    <div
                        key={enumValue.value}
                        className='p-4 rounded-lg bg-bg-secondary border-2 border-border-primary hover:border-accent-blue/50 transition-all duration-200'>
                        <div className='flex items-center gap-2 mb-2'>
                            <code className='px-3 py-1.5 rounded-lg bg-type-enum-bg text-type-enum border-2 border-type-enum-border font-mono text-sm font-semibold'>
                                &quot;{enumValue.value}&quot;
                            </code>
                            {enumValue.deprecated && (
                                <span className='text-xs px-2 py-0.5 rounded bg-warning-bg text-warning-text border border-warning-border font-semibold'>
                                    DEPRECATED
                                </span>
                            )}
                        </div>
                        {enumValue.description && <p className='text-sm text-text-tertiary'>{enumValue.description}</p>}
                        {enumValue.deprecated && (
                            <p className='text-sm text-warning-text mt-2 italic'>{enumValue.deprecated}</p>
                        )}
                    </div>
                ))}
            </div>
        </div>
    );
}

function FieldItem({ field }: { field: FieldMetadata }) {
    const displayType = "displayType" in field ? field.displayType : "";
    // Use typeInfo.type for actual type reference, fallback to displayType
    const actualType = "typeInfo" in field && field.typeInfo ? field.typeInfo.type : displayType;
    const isClickableType = isTypeLink(actualType);

    return (
        <div className='p-4 rounded-lg bg-bg-secondary border-2 border-border-primary hover:border-accent-blue/50 transition-all duration-200'>
            <div className='flex items-start justify-between mb-2 gap-4'>
                <div className='flex items-center gap-2 flex-wrap'>
                    <code className='text-lg font-semibold text-text-primary'>{field.name}</code>
                    {"typeInfo" in field && field.typeInfo && field.typeInfo.required && (
                        <span className='text-xs px-2 py-0.5 rounded bg-red-500/20 text-red-400 border border-red-500/30 font-semibold'>
                            required
                        </span>
                    )}
                    {"typeInfo" in field && field.typeInfo && !field.typeInfo.required && (
                        <span className='text-xs px-2 py-0.5 rounded bg-yellow-500/20 text-yellow-400 border border-yellow-500/30'>
                            optional
                        </span>
                    )}
                    {"typeInfo" in field && field.typeInfo && field.typeInfo.nullable && (
                        <span className='text-xs px-2 py-0.5 rounded bg-blue-500/20 text-blue-400 border border-blue-500/30'>
                            nullable
                        </span>
                    )}
                </div>
                <div className='shrink-0'>
                    {isClickableType ? (
                        <Link
                            href={`/api/type/${actualType}`}
                            className='inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-type-reference/10 text-type-reference hover:bg-type-reference/20 border-2 border-type-reference/30 hover:border-type-reference font-mono text-sm font-semibold transition-all duration-200 hover:scale-105'>
                            {displayType}
                            <BiLinkExternal className='w-3.5 h-3.5' />
                        </Link>
                    ) : (
                        <code className='px-3 py-1.5 rounded-lg bg-type-primitive/10 text-type-primitive border-2 border-type-primitive/30 font-mono text-sm font-semibold'>
                            {displayType}
                        </code>
                    )}
                </div>
            </div>

            {"description" in field && field.description && (
                <p className='text-sm text-text-tertiary mt-2'>{String(field.description)}</p>
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
    const usedBy =
        "usedBy" in data && Array.isArray(data.usedBy) && data.usedBy.length > 0 ? (data.usedBy as UsedByItem[]) : null;
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
