import type { ErrorData } from "@/data/api";
import { CardBoxWrapper } from "./card-box-wrapper";

type Props = {
    children?: React.ReactNode;
};

const HeaderRow = ({ children }: Props) => {
    return <th className='text-left p-3 text-sm font-semibold text-text-secondary'>{children}</th>;
};

const DataRow = ({ children }: Props) => {
    return (
        <td className='p-3 border-b border-border-primary text-sm text-text-tertiary'>
            {children}
        </td>
    );
};

type ErrorsProps = {
    errors?: ErrorData[];
};

export const Errors = ({ errors }: ErrorsProps) => {
    if (!errors?.length) return null;

    return (
        <CardBoxWrapper title='Errors'>
            <table className='w-full border-collapse'>
                <thead>
                    <tr className='bg-bg-sidebar border-b border-border-primary'>
                        <HeaderRow>Title</HeaderRow>
                        <HeaderRow>Code</HeaderRow>
                        <HeaderRow>Message</HeaderRow>
                        <HeaderRow>Description</HeaderRow>
                    </tr>
                </thead>
                <tbody>
                    {errors.map((error) => (
                        <tr key={error.code}>
                            <DataRow>{error.title}</DataRow>
                            <DataRow>
                                <span className='text-error-accent font-mono font-semibold'>
                                    {error.code}
                                </span>
                            </DataRow>
                            <DataRow>{error.message}</DataRow>
                            <DataRow>{error.description}</DataRow>
                        </tr>
                    ))}
                </tbody>
            </table>
        </CardBoxWrapper>
    );
};
