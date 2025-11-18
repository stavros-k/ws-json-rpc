import Link from "next/link";
import { MdOutlineBolt } from "react-icons/md";
import { TbWorld } from "react-icons/tb";
import { docs } from "@/data/api";

const methodCount = Object.keys(docs.methods).length;
const eventCount = Object.keys(docs.events).length;
const typeCount = Object.keys(docs.types).length;

export default function Home() {
    return (
        <div className='font-sans min-h-screen bg-bg-primary flex-1'>
            {/* Hero Section */}
            <div className='relative overflow-hidden bg-linear-to-br from-gradient-start via-gradient-mid to-gradient-end'>
                <div className='absolute inset-0 bg-black/30 dark:bg-black/40'></div>
                <div className='relative w-full px-6 py-24 sm:py-32 lg:px-8'>
                    <div className='text-center'>
                        <h1 className='text-5xl sm:text-6xl font-bold text-white mb-4 drop-shadow-lg'>
                            {docs.info.title}
                        </h1>
                        <div className='inline-block px-4 py-2 bg-white/90 dark:bg-bg-secondary/80 backdrop-blur-sm rounded-full mb-6 border-2 border-white/30 dark:border-border-primary shadow-lg'>
                            <span className='text-sm text-gray-800 dark:text-text-secondary font-bold'>
                                Version {docs.info.version}
                            </span>
                        </div>
                        <p className='text-xl sm:text-2xl text-white/95 max-w-3xl mx-auto drop-shadow-md font-medium'>
                            {docs.info.description}
                        </p>
                    </div>
                </div>
            </div>

            {/* Features Section */}
            <div className='w-full px-6 py-16 lg:px-8'>
                <h2 className='text-3xl font-bold text-text-primary text-center mb-12'>
                    Protocol Support
                </h2>
                <div className='grid md:grid-cols-2 gap-8 max-w-6xl mx-auto'>
                    <div className='bg-bg-secondary p-8 rounded-2xl border-2 border-success-border shadow-lg hover:shadow-2xl hover:scale-105 transition-all duration-300'>
                        <div className='w-14 h-14 bg-protocol-http rounded-xl flex items-center justify-center mb-5 shadow-md'>
                            <TbWorld className='w-8 h-8 text-text-primary' />
                        </div>
                        <h3 className='text-2xl font-bold text-text-primary mb-3'>
                            HTTP Support
                        </h3>
                        <p className='text-text-secondary text-base leading-relaxed'>
                            Make JSON-RPC calls over HTTP for simple
                            request-response patterns
                        </p>
                    </div>

                    <div className='bg-bg-secondary p-8 rounded-2xl border-2 border-protocol-ws shadow-lg hover:shadow-2xl hover:scale-105 transition-all duration-300'>
                        <div className='w-14 h-14 bg-protocol-ws rounded-xl flex items-center justify-center mb-5 shadow-md'>
                            <MdOutlineBolt className='w-8 h-8 text-text-primary' />
                        </div>
                        <h3 className='text-2xl font-bold text-text-primary mb-3'>
                            WebSocket Support
                        </h3>
                        <p className='text-text-secondary text-base leading-relaxed'>
                            Real-time bidirectional communication with event
                            subscriptions
                        </p>
                    </div>
                </div>
            </div>

            {/* Quick Stats */}
            <div className='w-full px-6 py-16 lg:px-8 mb-8'>
                <div className='w-full px-6 lg:px-8'>
                    <h2 className='text-3xl font-bold text-text-primary text-center mb-12'>
                        API Overview
                    </h2>
                    <div className='grid grid-cols-1 md:grid-cols-3 gap-8 max-w-6xl mx-auto'>
                        <Link
                            href='/api/methods'
                            className='group'>
                            <div className='bg-bg-secondary p-8 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-accent-blue cursor-pointer hover:scale-105'>
                                <div className='text-5xl font-bold text-accent-blue mb-3 group-hover:scale-110 transition-transform'>
                                    {methodCount}
                                </div>
                                <div className='text-xl font-bold text-text-primary group-hover:text-accent-blue transition-colors mb-2'>
                                    Methods
                                </div>
                                <p className='text-sm text-text-secondary'>
                                    Available RPC methods
                                </p>
                            </div>
                        </Link>

                        <Link
                            href='/api/events'
                            className='group'>
                            <div className='bg-bg-secondary p-8 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-info-border cursor-pointer hover:scale-105'>
                                <div className='text-5xl font-bold text-info-text mb-3 group-hover:scale-110 transition-transform'>
                                    {eventCount}
                                </div>
                                <div className='text-xl font-bold text-text-primary group-hover:text-info-text transition-colors mb-2'>
                                    Events
                                </div>
                                <p className='text-sm text-text-secondary'>
                                    Subscribable events
                                </p>
                            </div>
                        </Link>

                        <Link
                            href='/api/types'
                            className='group'>
                            <div className='bg-bg-secondary p-8 rounded-2xl shadow-lg hover:shadow-2xl transition-all duration-300 border-2 border-border-primary hover:border-accent-blue-light cursor-pointer hover:scale-105'>
                                <div className='text-5xl font-bold text-accent-blue-light mb-3 group-hover:scale-110 transition-transform'>
                                    {typeCount}
                                </div>
                                <div className='text-xl font-bold text-text-primary group-hover:text-accent-blue-light transition-colors mb-2'>
                                    Types
                                </div>
                                <p className='text-sm text-text-secondary'>
                                    Type definitions
                                </p>
                            </div>
                        </Link>
                    </div>
                </div>
            </div>
        </div>
    );
}
