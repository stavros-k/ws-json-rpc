type Props = {
    group: string;
    tags?: string[];
};
export const GroupAndTags = ({ group, tags }: Props) => {
    if (!group && !tags?.length) return null;

    return (
        <div className='flex gap-2 mt-3'>
            <span className='bg-tag-blue-bg text-tag-blue-text px-3 py-1 rounded-xl text-xs font-medium border border-tag-blue-border'>
                {group}
            </span>
            {tags?.map((tag) => (
                <span
                    key={tag}
                    className='bg-info-bg text-info-text px-3 py-1 rounded-xl text-xs font-medium border border-info-border'>
                    {tag}
                </span>
            ))}
        </div>
    );
};
