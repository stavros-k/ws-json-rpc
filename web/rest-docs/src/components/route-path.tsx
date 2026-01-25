type RoutePathProps = {
    path: string;
    className?: string;
};

export function RoutePath({ path, className = "" }: RoutePathProps) {
    // Split the path into segments and identify parameters (anything between {})
    const segments = path.split(/(\{[^}]+\})/g).filter(Boolean);

    return (
        <span className={className}>
            {segments.map((segment, idx) => {
                const isParam = segment.startsWith("{") && segment.endsWith("}");
                if (isParam) {
                    return (
                        <span
                            key={`param-${segment}-${idx}`}
                            className='text-accent-blue font-bold'>
                            {segment}
                        </span>
                    );
                }
                return (
                    <span
                        key={`segment-${segment}-${idx}`}
                        className='text-text-secondary'>
                        {segment}
                    </span>
                );
            })}
        </span>
    );
}
