type Props = {
    deprecated: string;
    itemType?: "type" | "operation";
};

export const Deprecation = ({ deprecated, itemType = "type" }: Props) => {
    if (!deprecated) return null;

    const itemLabel = itemType === "operation" ? "operation" : "type";

    return (
        <div className='bg-warning-bg border-2 border-warning-border px-4 py-3 rounded-lg mb-6 text-warning-text'>
            <div className='flex items-start gap-3'>
                <span className='text-xl'>⚠️</span>
                <div>
                    <p className='font-bold mb-1'>This {itemLabel} is deprecated</p>
                    <p className='text-sm'>{deprecated}</p>
                </div>
            </div>
        </div>
    );
};
