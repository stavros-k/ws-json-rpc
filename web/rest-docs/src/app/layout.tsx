import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Sidebar } from "@/components/sidebar";
import { ThemeToggle } from "@/components/theme-toggle-client";
import { AutoSubscribeProvider } from "@/contexts/auto-subscribe-context";
import { MaxResultsProvider } from "@/contexts/max-results-context";
import { WebSocketProvider } from "@/contexts/websocket-context";
import { docs } from "@/data/api";

export const metadata: Metadata = {
    title: docs.info.title,
};

export const viewport: Viewport = {
    width: "device-width",
    initialScale: 1,
};

export default function RootLayout({ children }: LayoutProps<"/">) {
    return (
        <html
            lang='en'
            className='m-0 p-0 box-border'>
            <body className='bg-bg-primary'>
                <div className='fixed top-5 right-5 z-50'>
                    <ThemeToggle />
                </div>
                <WebSocketProvider>
                    <AutoSubscribeProvider>
                        <MaxResultsProvider>
                            <div className='flex min-h-screen'>
                                <Sidebar />
                                {children}
                            </div>
                        </MaxResultsProvider>
                    </AutoSubscribeProvider>
                </WebSocketProvider>
            </body>
        </html>
    );
}
