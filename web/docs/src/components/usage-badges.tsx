import { TbApi, TbTopologyRing } from "react-icons/tb";
import type { UsedByItem } from "@/data/api";

interface UsageBadgesProps {
    usedBy: UsedByItem[] | undefined;
}

export function UsageBadges({ usedBy }: UsageBadgesProps) {
    if (!usedBy || usedBy.length === 0) {
        return null;
    }

    const hasHTTP = usedBy.some((item) => ["request", "response", "parameter"].includes(item.role));
    const hasMQTT = usedBy.some((item) => item.role === "mqtt_publication" || item.role === "mqtt_subscription");

    if (!hasHTTP && !hasMQTT) {
        return null;
    }

    return (
        <div className='flex items-center gap-2'>
            {hasHTTP && (
                <div className='flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-green-bg text-accent-green-text border-2 border-accent-green-border font-semibold text-sm'>
                    <TbApi className='w-4 h-4' />
                    <span>HTTP</span>
                </div>
            )}
            {hasMQTT && (
                <div className='flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-purple-bg text-accent-purple-text border-2 border-accent-purple-border font-semibold text-sm'>
                    <TbTopologyRing className='w-4 h-4' />
                    <span>MQTT</span>
                </div>
            )}
        </div>
    );
}
