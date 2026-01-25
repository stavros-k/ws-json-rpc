import type { getItemData } from "./sidebar";

type Props = {
    item: ReturnType<typeof getItemData>;
};

export const Protocols = ({ item }: Props) => {
    if (item.type !== "method" && item.type !== "event") return null;
    if (!item.data.protocols) return null;

    return (
        <div className='flex gap-1'>
            {item.data.protocols.http && (
                <span className='text-[10px] px-1 py-0.5 rounded-sm bg-protocol-http text-white'>HTTP</span>
            )}
            {item.data.protocols.ws && (
                <span className='text-[10px] px-1 py-0.5 rounded-sm bg-protocol-ws text-white'>WS</span>
            )}
        </div>
    );
};
