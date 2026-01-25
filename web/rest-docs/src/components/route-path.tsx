type RoutePathProps = {
    path: string;
    className?: string;
};

export function RoutePath({ path, className = "" }: RoutePathProps) {
    // Split the path into segments and identify parameters (anything between {})
    const segments = path.split(/(\{[^}]+\})/g).filter(Boolean);

    return (
        <span className={className}>
            {segments.map((segment, index) => {
                const isParam = segment.startsWith("{") && segment.endsWith("}");
                if (isParam) {
                    return (
                        <span
                            key={index}
                            className='text-accent-blue font-bold'>
                            {segment}
                        </span>
                    );
                }
                return (
                    <span
                        key={index}
                        className='text-text-secondary'>
                        {segment}
                    </span>
                );
            })}
        </span>
    );
}
