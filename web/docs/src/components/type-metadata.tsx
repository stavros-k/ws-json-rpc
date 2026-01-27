import type { Route } from "next";
import Link from "next/link";
import { BiLinkExternal } from "react-icons/bi";
import type { EnumValue, FieldMetadata, TypeData, UsedByItem } from "@/data/api";
import { docs, getAllMQTTPublications, getAllMQTTSubscriptions, getAllOperations } from "@/data/api";
import { RoutePath } from "./route-path";
import { VerbBadge } from "./verb-badge";

interface TypeMetadataProps {
    data: TypeData;
    typeName: string;
}

// Helper function to check if a type name exists in the docs
const isTypeLink = (type: string) => {
    return type in docs.types;
};

function TypeDescription({ description }: { description?: string }) {
    if (!description) return null;

    return (
        <div>
            <h2 className='text-xl font-semibold mb-3 text-text-primary'>Description</h2>
            <p className='text-text-secondary'>{description}</p>
        </div>
    );
}

function UsedBySection({ usedBy }: { usedBy: UsedByItem[] | null }) {
    if (!usedBy || usedBy.length === 0) return null;

    const allOperations = getAllOperations();
    const allPublications = getAllMQTTPublications();
    const allSubscriptions = getAllMQTTSubscriptions();

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
            <p className='text-sm text-text-tertiary mb-4'>This type is used by the following operations:</p>
            <div className='grid grid-cols-1 gap-3'>
                {Object.entries(grouped).map(([operationID, roles]) => {
                    // Check roles to determine operation type
                    const isHTTP = Array.from(roles).some((role) =>
                        ["request", "response", "parameter"].includes(role)
                    );
                    const isMQTTPublication = roles.has("mqtt_publication");
                    const isMQTTSubscription = roles.has("mqtt_subscription");

                    // HTTP Operation
                    if (isHTTP) {
                        const operation = allOperations.find((op) => op.operationID === operationID);
                        return (
                            <Link
                                key={operationID}
                                href={`/api/operation/${operationID}` as Route}
                                className='block p-4 rounded-lg bg-bg-tertiary border-2 border-border-primary hover:border-accent-green-light transition-all duration-200 hover:shadow-md'>
                                <div className='flex flex-col gap-2'>
                                    <div className='flex items-center justify-between gap-3'>
                                        <div className='flex items-center gap-2 min-w-0'>
                                            {operation && (
                                                <>
                                                    <VerbBadge
                                                        verb={operation.method}
                                                        size='xs'
                                                    />
                                                    <span className='text-sm font-mono font-semibold truncate'>
                                                        <RoutePath path={operation.path} />
                                                    </span>
                                                </>
                                            )}
                                        </div>
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
                                    <code className='text-xs text-text-muted font-mono'>{operationID}</code>
                                </div>
                            </Link>
                        );
                    }

                    // MQTT Publication
                    if (isMQTTPublication) {
                        const publication = allPublications.find((pub) => pub.operationID === operationID);
                        return (
                            <Link
                                key={operationID}
                                href={`/api/mqtt/publication/${operationID}` as Route}
                                className='block p-4 rounded-lg bg-bg-tertiary border-2 border-border-primary hover:border-accent-blue-light transition-all duration-200 hover:shadow-md'>
                                <div className='flex flex-col gap-2'>
                                    <div className='flex items-center justify-between gap-3'>
                                        <div className='flex items-center gap-2 min-w-0'>
                                            {publication && (
                                                <>
                                                    <span className='px-2 py-0.5 rounded text-xs font-bold bg-accent-blue-bg text-accent-blue-text border border-accent-blue-border'>
                                                        PUB
                                                    </span>
                                                    <span className='text-sm font-mono font-semibold truncate'>
                                                        <RoutePath path={publication.topic} />
                                                    </span>
                                                </>
                                            )}
                                        </div>
                                        <div className='flex flex-wrap gap-1 shrink-0'>
                                            {Array.from(roles).map((role) => (
                                                <span
                                                    key={role}
                                                    className='text-xs px-2 py-0.5 rounded font-medium bg-blue-500/20 text-blue-400 border border-blue-500/30'>
                                                    {role}
                                                </span>
                                            ))}
                                        </div>
                                    </div>
                                    <code className='text-xs text-text-muted font-mono'>{operationID}</code>
                                </div>
                            </Link>
                        );
                    }

                    // MQTT Subscription
                    if (isMQTTSubscription) {
                        const subscription = allSubscriptions.find((sub) => sub.operationID === operationID);
                        return (
                            <Link
                                key={operationID}
                                href={`/api/mqtt/subscription/${operationID}` as Route}
                                className='block p-4 rounded-lg bg-bg-tertiary border-2 border-border-primary hover:border-accent-green-light transition-all duration-200 hover:shadow-md'>
                                <div className='flex flex-col gap-2'>
                                    <div className='flex items-center justify-between gap-3'>
                                        <div className='flex items-center gap-2 min-w-0'>
                                            {subscription && (
                                                <>
                                                    <span className='px-2 py-0.5 rounded text-xs font-bold bg-accent-green-bg text-accent-green-text border border-accent-green-border'>
                                                        SUB
                                                    </span>
                                                    <span className='text-sm font-mono font-semibold truncate'>
                                                        <RoutePath path={subscription.topic} />
                                                    </span>
                                                </>
                                            )}
                                        </div>
                                        <div className='flex flex-wrap gap-1 shrink-0'>
                                            {Array.from(roles).map((role) => (
                                                <span
                                                    key={role}
                                                    className='text-xs px-2 py-0.5 rounded font-medium bg-green-500/20 text-green-400 border border-green-500/30'>
                                                    {role}
                                                </span>
                                            ))}
                                        </div>
                                    </div>
                                    <code className='text-xs text-text-muted font-mono'>{operationID}</code>
                                </div>
                            </Link>
                        );
                    }

                    return null;
                })}
            </div>
        </div>
    );
}

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

    // Check if this field has additionalProperties (map/dictionary type)
    const additionalProps = field.typeInfo?.additionalProperties;

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

            {additionalProps && (
                <div className='mt-3 p-3 rounded-lg bg-bg-tertiary border border-border-primary'>
                    <div className='flex items-center gap-2 mb-1'>
                        <span className='text-xs font-semibold text-text-secondary uppercase tracking-wide'>
                            Map/Dictionary Type
                        </span>
                    </div>
                    <div className='flex items-center gap-2 text-sm'>
                        <span className='text-text-tertiary'>Values:</span>
                        {isTypeLink(additionalProps.type) ? (
                            <Link
                                href={`/api/type/${additionalProps.type}`}
                                className='inline-flex items-center gap-1 px-2 py-0.5 rounded bg-type-reference/10 text-type-reference hover:bg-type-reference/20 border border-type-reference/30 hover:border-type-reference font-mono text-xs font-semibold transition-all duration-200'>
                                {additionalProps.type}
                                <BiLinkExternal className='w-3 h-3' />
                            </Link>
                        ) : (
                            <code className='px-2 py-0.5 rounded bg-type-primitive/10 text-type-primitive border border-type-primitive/30 font-mono text-xs font-semibold'>
                                {additionalProps.type}
                            </code>
                        )}
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
