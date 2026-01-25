type Props = {
    title: string;
    children?: React.ReactNode;
};
export const CardBoxWrapper = ({ title, children }: Props) => {
    return (
        <div className='bg-bg-secondary rounded-xl p-6 mb-6 shadow-md border-2 border-border-primary hover:border-accent-blue transition-colors duration-200'>
            <h2 className='text-xl font-bold text-text-primary mb-4'>{title}</h2>
            {children}
        </div>
    );
};
