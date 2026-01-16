type PageHeaderProps = {
    title: string;
    description: string;
};

export const PageHeader = ({ title, description }: PageHeaderProps) => {
    return (
        <div className='mb-8'>
            <h1 className='text-4xl font-bold mb-3 text-text-primary'>{title}</h1>
            <p className='text-text-secondary text-lg'>{description}</p>
        </div>
    );
};
