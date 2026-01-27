import Link from "next/link";
import { TbApi, TbCode, TbDatabase, TbFileDescription, TbTopologyRing } from "react-icons/tb";
import { VerbBadge } from "@/components/verb-badge";
import { docs, getAllMQTTPublications, getAllMQTTSubscriptions, getAllOperations } from "@/data/api";

// Calculate total number of operations
const operations = getAllOperations();
const operationCount = operations.length;

// Calculate MQTT operations
const mqttPublications = getAllMQTTPublications();
const mqttSubscriptions = getAllMQTTSubscriptions();
const mqttPublicationCount = mqttPublications.length;
const mqttSubscriptionCount = mqttSubscriptions.length;

// Count types used by HTTP operations only
const httpTypeCount = Object.values(docs.types).filter((type) => {
    if (!("usedBy" in type) || !type.usedBy) return false;
    return type.usedBy.some((usage) => ["request", "response", "parameter"].includes(usage.role));
}).length;

// Count types used by MQTT operations only
const mqttTypeCount = Object.values(docs.types).filter((type) => {
    if (!("usedBy" in type) || !type.usedBy) return false;
    return type.usedBy.some((usage) => ["mqtt_publication", "mqtt_subscription"].includes(usage.role));
}).length;

const tableCount = docs.database.tableCount || 0;

// Get unique paths for route count
const uniquePaths = new Set(operations.map((op) => op.path));
const routeCount = uniquePaths.size;

// Calculate HTTP method distribution
const httpMethods = operations.reduce(
    (acc, op) => {
        acc[op.method] = (acc[op.method] || 0) + 1;
        return acc;
    },
    {} as Record<string, number>
);

export default function Home() {
    return (
        <div className='font-sans min-h-screen bg-bg-primary flex-1'>
            {/* Hero Section */}
            <div className='relative overflow-hidden bg-linear-to-br from-gradient-start via-gradient-mid to-gradient-end'>
                <div className='absolute inset-0 bg-black/30 dark:bg-black/40'></div>
                <div className='relative w-full px-6 py-16 sm:py-20 lg:px-8'>
                    <div className='text-center'>
                        <h1 className='text-4xl sm:text-5xl font-bold text-white mb-3 drop-shadow-lg'>
                            {docs.info.title}
                        </h1>
                        <div className='inline-block px-4 py-2 bg-white/90 dark:bg-bg-secondary/80 backdrop-blur-sm rounded-full mb-4 border-2 border-white/30 dark:border-border-primary shadow-lg'>
                            <span className='text-sm text-gray-800 dark:text-text-secondary font-bold'>
                                Version {docs.info.version}
                            </span>
                        </div>
                        <p className='text-lg sm:text-xl text-white/95 max-w-2xl mx-auto drop-shadow-md font-medium'>
                            {docs.info.description}
                        </p>
                    </div>
                </div>
            </div>

            {/* HTTP Section */}
            <div className='w-full px-6 py-12 lg:px-8 bg-bg-secondary/20'>
                <h2 className='text-3xl font-bold text-text-primary text-center mb-10'>HTTP API</h2>
                <div className='grid grid-cols-1 md:grid-cols-3 gap-6 max-w-6xl mx-auto mb-8'>
                    {/* HTTP Operations Card */}
                    <Link
                        href='/api/operations'
                        className='block'>
                        <div className='h-full bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-accent-blue cursor-pointer hover:scale-105'>
                            <div className='flex items-center gap-3 mb-4'>
                                <div className='w-12 h-12 bg-accent-blue/20 rounded-xl flex items-center justify-center shadow-md'>
                                    <TbApi className='w-7 h-7 text-accent-blue' />
                                </div>
                                <div className='text-lg font-bold text-text-primary'>Operations</div>
                            </div>
                            <div className='text-4xl font-bold text-accent-blue mb-2'>{operationCount}</div>
                            <p className='text-xs text-text-secondary'>
                                Across {routeCount} route{routeCount !== 1 ? "s" : ""}
                            </p>
                        </div>
                    </Link>

                    {/* HTTP Types Card */}
                    <Link
                        href='/api/types'
                        className='block'>
                        <div className='h-full bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-success-border cursor-pointer hover:scale-105'>
                            <div className='flex items-center gap-3 mb-4'>
                                <div className='w-12 h-12 bg-success-bg rounded-xl flex items-center justify-center shadow-md'>
                                    <TbCode className='w-7 h-7 text-success-text' />
                                </div>
                                <div className='text-lg font-bold text-text-primary'>Types</div>
                            </div>
                            <div className='text-4xl font-bold text-success-text mb-2'>{httpTypeCount}</div>
                            <p className='text-xs text-text-secondary'>HTTP types</p>
                        </div>
                    </Link>

                    {/* OpenAPI Specification Card */}
                    <Link
                        href='/api/openapi'
                        className='block'>
                        <div className='h-full bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-warning-border cursor-pointer hover:scale-105'>
                            <div className='flex items-center gap-3 mb-4'>
                                <div className='w-12 h-12 bg-warning-bg rounded-xl flex items-center justify-center shadow-md'>
                                    <TbFileDescription className='w-7 h-7 text-warning-text' />
                                </div>
                                <div className='text-lg font-bold text-text-primary'>OpenAPI</div>
                            </div>
                            <div className='text-4xl font-bold text-warning-text mb-2'>3.0</div>
                            <p className='text-xs text-text-secondary'>Specification</p>
                        </div>
                    </Link>
                </div>

                {/* HTTP Methods */}
                <div className='max-w-6xl mx-auto'>
                    <h3 className='text-xl font-bold text-text-primary text-center mb-6'>HTTP Methods</h3>
                    <div className='flex flex-wrap justify-center gap-4'>
                        {Object.entries(httpMethods)
                            .sort(([, a], [, b]) => b - a)
                            .map(([method, count]) => (
                                <div
                                    key={method}
                                    className='bg-bg-secondary p-4 rounded-xl shadow-lg hover:shadow-xl transition-all duration-300 border-2 border-border-primary hover:scale-105 w-32'>
                                    <div className='text-center'>
                                        <VerbBadge
                                            verb={method}
                                            size='sm'
                                        />
                                        <div className='text-2xl font-bold text-text-primary mt-3'>{count}</div>
                                    </div>
                                </div>
                            ))}
                    </div>
                </div>
            </div>

            {/* MQTT Section */}
            <div className='w-full px-6 py-12 lg:px-8'>
                <h2 className='text-3xl font-bold text-text-primary text-center mb-10'>MQTT API</h2>
                <div className='grid grid-cols-1 md:grid-cols-3 gap-6 max-w-6xl mx-auto'>
                    {/* MQTT Publications Card */}
                    <Link
                        href='/api/mqtt/publications'
                        className='block'>
                        <div className='h-full bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-accent-blue cursor-pointer hover:scale-105'>
                            <div className='flex items-center gap-3 mb-4'>
                                <div className='w-12 h-12 bg-accent-blue/20 rounded-xl flex items-center justify-center shadow-md'>
                                    <TbTopologyRing className='w-7 h-7 text-accent-blue' />
                                </div>
                                <div className='text-lg font-bold text-text-primary'>Publications</div>
                            </div>
                            <div className='text-4xl font-bold text-accent-blue mb-2'>{mqttPublicationCount}</div>
                            <p className='text-xs text-text-secondary'>Server publishes</p>
                        </div>
                    </Link>

                    {/* MQTT Subscriptions Card */}
                    <Link
                        href='/api/mqtt/subscriptions'
                        className='block'>
                        <div className='h-full bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-accent-green cursor-pointer hover:scale-105'>
                            <div className='flex items-center gap-3 mb-4'>
                                <div className='w-12 h-12 bg-accent-green/20 rounded-xl flex items-center justify-center shadow-md'>
                                    <TbTopologyRing className='w-7 h-7 text-accent-green' />
                                </div>
                                <div className='text-lg font-bold text-text-primary'>Subscriptions</div>
                            </div>
                            <div className='text-4xl font-bold text-accent-green mb-2'>{mqttSubscriptionCount}</div>
                            <p className='text-xs text-text-secondary'>Server subscribes</p>
                        </div>
                    </Link>

                    {/* MQTT Types Card */}
                    <Link
                        href='/api/types'
                        className='block'>
                        <div className='h-full bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-accent-purple cursor-pointer hover:scale-105'>
                            <div className='flex items-center gap-3 mb-4'>
                                <div className='w-12 h-12 bg-accent-purple/20 rounded-xl flex items-center justify-center shadow-md'>
                                    <TbCode className='w-7 h-7 text-accent-purple' />
                                </div>
                                <div className='text-lg font-bold text-text-primary'>Types</div>
                            </div>
                            <div className='text-4xl font-bold text-accent-purple mb-2'>{mqttTypeCount}</div>
                            <p className='text-xs text-text-secondary'>MQTT types</p>
                        </div>
                    </Link>
                </div>
            </div>

            {/* Other Resources Section */}
            <div className='w-full px-6 py-12 lg:px-8 bg-bg-secondary/20'>
                <h2 className='text-3xl font-bold text-text-primary text-center mb-10'>Resources</h2>
                <div className='grid grid-cols-1 md:grid-cols-2 gap-6 max-w-4xl mx-auto'>
                    {/* Types Card */}
                    <Link
                        href='/api/types'
                        className='block'>
                        <div className='h-full bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-success-border cursor-pointer hover:scale-105'>
                            <div className='flex items-center gap-3 mb-4'>
                                <div className='w-12 h-12 bg-success-bg rounded-xl flex items-center justify-center shadow-md'>
                                    <TbCode className='w-7 h-7 text-success-text' />
                                </div>
                                <div className='text-lg font-bold text-text-primary'>All Types</div>
                            </div>
                            <div className='text-4xl font-bold text-success-text mb-2'>
                                {Object.keys(docs.types).length}
                            </div>
                            <p className='text-xs text-text-secondary'>Type definitions</p>
                        </div>
                    </Link>

                    {/* Database Schema Card */}
                    <Link
                        href='/api/database/schema'
                        className='block'>
                        <div className='h-full bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-info-border cursor-pointer hover:scale-105'>
                            <div className='flex items-center gap-3 mb-4'>
                                <div className='w-12 h-12 bg-info-bg rounded-xl flex items-center justify-center shadow-md'>
                                    <TbDatabase className='w-7 h-7 text-info-text' />
                                </div>
                                <div className='text-lg font-bold text-text-primary'>Database</div>
                            </div>
                            <div className='text-4xl font-bold text-info-text mb-2'>{tableCount}</div>
                            <p className='text-xs text-text-secondary'>Table{tableCount !== 1 ? "s" : ""}</p>
                        </div>
                    </Link>
                </div>
            </div>

        </div>
    );
}
