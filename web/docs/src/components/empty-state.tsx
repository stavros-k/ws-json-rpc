type EmptyStateProps = {
    title: string;
    description: string;
    icon?: string;
};

export function EmptyState({ title, description, icon = "🔍" }: EmptyStateProps) {
    return (
        <div className='flex flex-col items-center justify-center py-16 px-6 bg-bg-secondary rounded-xl border-2 border-border-primary'>
            <div className='text-6xl mb-4'>{icon}</div>
            <h3 className='text-xl font-bold text-text-primary mb-2'>{title}</h3>
            <p className='text-sm text-text-secondary text-center max-w-md'>{description}</p>
        </div>
    );
}
