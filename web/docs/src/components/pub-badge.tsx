type PubBadgeProps = {
    border?: boolean;
};
export function PubBadge({ border = true }: PubBadgeProps) {
    return (
        <span
            className={`px-1.5 py-0.5 rounded text-[10px] font-bold bg-accent-blue-bg ${border ? "border border-accent-blue-border" : ""} text-accent-blue-text`}>
            PUB
        </span>
    );
}
