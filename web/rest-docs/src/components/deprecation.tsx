import type { ItemType } from "@/data/api";

type Props = {
    type: ItemType;
    deprecated: boolean;
};

export const Deprecation = ({ type, deprecated }: Props) => {
    if (!deprecated) return null;

    return (
        <div className='bg-warning-bg border border-warning-border px-4 py-3 rounded-lg mb-4 text-warning-text'>
            ⚠️ This {type} is deprecated and may be removed in a future version.
        </div>
    );
};
