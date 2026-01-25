type DeprecationInfo = {
    message: string;
} | null;

type Props = {
    deprecated: DeprecationInfo;
};

export const Deprecation = ({ deprecated }: Props) => {
    if (!deprecated) return null;

    return (
        <div className='bg-warning-bg border-2 border-warning-border px-4 py-3 rounded-lg mb-6 text-warning-text'>
            <div className='flex items-start gap-3'>
                <span className='text-xl'>⚠️</span>
                <div>
                    <p className='font-bold mb-1'>This type is deprecated</p>
                    {deprecated.message && <p className='text-sm'>{deprecated.message}</p>}
                </div>
            </div>
        </div>
    );
};
