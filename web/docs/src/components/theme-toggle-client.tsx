"use client";
import { useEffect, useState } from "react";

export type Theme = "light" | "dark";

export const ThemeToggle = () => {
    const [theme, setTheme] = useState<Theme>("dark");

    useEffect(() => {
        // Check for saved preference first
        const saved = localStorage.getItem("theme") as Theme | null;
        if (saved) {
            setTheme(saved);
            document.documentElement.dataset.theme = saved;
        } else {
            // Only use system preference if no saved preference exists
            const isDark = window.matchMedia(
                "(prefers-color-scheme: dark)"
            ).matches;
            const systemTheme = isDark ? "dark" : "light";
            setTheme(systemTheme);
            document.documentElement.dataset.theme = systemTheme;
        }
    }, []);

    const toggleTheme = () => {
        const newTheme = theme === "dark" ? "light" : "dark";
        setTheme(newTheme);
        localStorage.setItem("theme", newTheme);
        document.documentElement.dataset.theme = newTheme;
    };

    return (
        <button
            type='button'
            onClick={toggleTheme}
            className='w-12 h-12 flex items-center justify-center rounded-xl bg-bg-secondary hover:bg-bg-hover border-2 border-border-primary hover:border-accent-blue transition-all duration-200 shadow-md hover:shadow-lg hover:scale-110 text-xl'
            aria-label='Toggle theme'>
            {theme === "dark" ? "☀️" : "🌙"}
        </button>
    );
};
