type Props = {
    group: string;
};
export const Group = ({ group }: Props) => {
    if (!group) return null;

    return (
        <div className='flex gap-2 mt-3'>
            <span className='bg-tag-blue-bg text-tag-blue-text px-3 py-1 rounded-xl text-xs font-medium border border-tag-blue-border'>
                {group}
            </span>
        </div>
    );
};
