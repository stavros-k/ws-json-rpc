type Props = {
    group: string;
    size?: "sm" | "md" | "lg";
};
export const Group = ({ group, size = "md" }: Props) => {
    if (!group) return null;

    const sizeClasses = {
        sm: "text-xs px-2 py-1",
        md: "text-sm px-3 py-1.5",
        lg: "text-base px-4 py-2",
    }[size];

    return (
        <span className={`${sizeClasses} bg-tag-blue-bg text-tag-blue-text rounded-lg font-bold border-2 border-tag-blue-border shadow-sm inline-block`}>
            {group}
        </span>
    );
};
