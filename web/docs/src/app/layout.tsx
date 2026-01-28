import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Sidebar } from "@/components/sidebar";
import { ThemeToggle } from "@/components/theme-toggle-client";
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
            <body className='bg-bg-primary flex'>
                <div className='fixed top-5 right-5 z-50'>
                    <ThemeToggle />
                </div>
                <Sidebar />
                <main className='flex-1 min-h-screen'>{children}</main>
            </body>
        </html>
    );
}
