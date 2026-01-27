type SubBadgeProps = {
    border?: boolean;
};
export function SubBadge({ border = true }: SubBadgeProps) {
    return (
        <span
            className={`px-1.5 py-0.5 rounded text-[10px] font-bold bg-accent-green-bg ${border ? "border border-accent-green-border" : ""} text-accent-green-text`}>
            SUB
        </span>
    );
}
