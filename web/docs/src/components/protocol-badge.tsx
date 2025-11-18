type Props = {
    title: string;
    supported: boolean;
};
export const ProtocolBadge = ({ title, supported }: Props) => {
    return (
        <span
            className={`px-3 py-1.5 rounded-lg text-xs font-bold shadow-sm ${
                supported
                    ? "bg-success-bg text-success-text border-2 border-success-border"
                    : "bg-error-bg text-error-text border-2 border-error-border"
            }`}>
            <span className='mr-1.5'>{supported ? "✅" : "❌"}</span>
            {title}
        </span>
    );
};
