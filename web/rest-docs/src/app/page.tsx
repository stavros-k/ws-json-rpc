import Link from "next/link";
import { TbApi, TbCode, TbDatabase, TbWorld } from "react-icons/tb";
import { VerbBadge } from "@/components/verb-badge";
import { docs } from "@/data/api";

// Calculate total number of operations across all routes
const routeCount = Object.keys(docs.routes).length;
const operationCount = Object.values(docs.routes).reduce((total, route) => {
    return total + Object.keys(route.verbs).length;
}, 0);
const typeCount = Object.keys(docs.types).length;

// Calculate HTTP method distribution
const httpMethods = Object.values(docs.routes).reduce(
    (acc, route) => {
        Object.keys(route.verbs).forEach((verb) => {
            acc[verb] = (acc[verb] || 0) + 1;
        });
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

            {/* API Overview */}
            <div className='w-full px-6 py-12 lg:px-8'>
                <h2 className='text-3xl font-bold text-text-primary text-center mb-10'>API Overview</h2>
                <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 max-w-7xl mx-auto'>
                    {/* Operations Card */}
                    <div className='bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-accent-blue hover:scale-105'>
                        <div className='w-12 h-12 bg-accent-blue/20 rounded-xl flex items-center justify-center mb-4 shadow-md'>
                            <TbApi className='w-7 h-7 text-accent-blue' />
                        </div>
                        <div className='text-4xl font-bold text-accent-blue mb-2'>{operationCount}</div>
                        <div className='text-lg font-bold text-text-primary mb-1'>Operations</div>
                        <p className='text-xs text-text-secondary'>
                            Across {routeCount} route{routeCount !== 1 ? "s" : ""}
                        </p>
                    </div>

                    {/* Types Card */}
                    <Link
                        href='/api/types'
                        className='block'>
                        <div className='h-full bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-success-border cursor-pointer hover:scale-105'>
                            <div className='w-12 h-12 bg-success-bg rounded-xl flex items-center justify-center mb-4 shadow-md'>
                                <TbCode className='w-7 h-7 text-success-text' />
                            </div>
                            <div className='text-4xl font-bold text-success-text mb-2'>{typeCount}</div>
                            <div className='text-lg font-bold text-text-primary mb-1'>Types</div>
                            <p className='text-xs text-text-secondary'>Type definitions</p>
                        </div>
                    </Link>

                    {/* HTTP Methods Card */}
                    <div className='bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:scale-105'>
                        <div className='w-12 h-12 bg-purple-500/20 rounded-xl flex items-center justify-center mb-4 shadow-md'>
                            <TbWorld className='w-7 h-7 text-purple-500' />
                        </div>
                        <div className='text-lg font-bold text-text-primary mb-2'>HTTP Methods</div>
                        <div className='flex flex-col gap-2'>
                            {Object.entries(httpMethods)
                                .sort(([, a], [, b]) => b - a)
                                .map(([method, count]) => (
                                    <div
                                        key={method}
                                        className='flex items-center justify-between'>
                                        <VerbBadge
                                            verb={method}
                                            size='sm'
                                        />
                                        <span className='text-sm font-bold text-text-secondary'>{count}</span>
                                    </div>
                                ))}
                        </div>
                    </div>

                    {/* Database Schema Card */}
                    <Link
                        href='/api/database/schema'
                        className='block'>
                        <div className='h-full bg-bg-secondary p-6 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-info-border cursor-pointer hover:scale-105'>
                            <div className='w-12 h-12 bg-info-bg rounded-xl flex items-center justify-center mb-4 shadow-md'>
                                <TbDatabase className='w-7 h-7 text-info-text' />
                            </div>
                            <div className='text-lg font-bold text-text-primary mb-1 mt-[22px]'>Database</div>
                            <p className='text-xs text-text-secondary'>Schema & tables</p>
                        </div>
                    </Link>
                </div>
            </div>

            {/* Available Servers - Moved to bottom */}
            {docs.info.servers && docs.info.servers.length > 0 && (
                <div className='w-full px-6 py-12 lg:px-8 bg-bg-secondary/20 mb-8'>
                    <h2 className='text-3xl font-bold text-text-primary text-center mb-10'>Available Servers</h2>
                    <div className='grid md:grid-cols-2 gap-6 max-w-4xl mx-auto'>
                        {docs.info.servers.map((server) => (
                            <div
                                key={server.URL}
                                className='bg-bg-secondary p-6 rounded-2xl border-2 border-border-primary shadow-lg hover:shadow-2xl hover:scale-105 transition-all duration-300'>
                                <div className='w-12 h-12 bg-accent-blue/20 rounded-xl flex items-center justify-center mb-4 shadow-md'>
                                    <TbWorld className='w-7 h-7 text-accent-blue' />
                                </div>
                                <h3 className='text-xl font-bold text-text-primary mb-2'>{server.Description}</h3>
                                <code className='text-sm text-accent-blue bg-bg-tertiary px-3 py-1.5 rounded-lg border border-border-primary inline-block break-all'>
                                    {server.URL}
                                </code>
                            </div>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}
