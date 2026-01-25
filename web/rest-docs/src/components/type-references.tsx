import Link from "next/link";
import type { TypeData } from "@/data/api";

interface TypeReferencesProps {
    typeName: string;
    data: TypeData;
}

export function TypeReferences({ typeName, data }: TypeReferencesProps) {
    const references = "references" in data ? data.references : undefined;
    const referencedBy = "referencedBy" in data ? data.referencedBy : undefined;

    const hasReferences = references && references.length > 0;
    const hasReferencedBy = referencedBy && referencedBy.length > 0;

    return (
        <div className='space-y-6'>
            {/* Types this type references */}
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

            {/* Types that reference this type */}
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
        </div>
    );
}
