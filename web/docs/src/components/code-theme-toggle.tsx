"use client";
import { useEffect, useState } from "react";
import { type CodeTheme, THEMES } from "@/utils/code-theme";
import type { Theme } from "./theme-toggle-client";

// Resolve code theme for current UI theme
function resolveCodeTheme(theme: Theme): CodeTheme {
    const storageKey = `codeTheme-${theme}`;
    const saved = localStorage.getItem(storageKey) as CodeTheme | null;
    const themeIDs = Object.keys(THEMES);

    // Validate saved theme matches the UI theme category
    if (saved && themeIDs.includes(saved) && THEMES[saved].category === theme) {
        document.documentElement.dataset.codeTheme = saved;
        return saved;
    }

    // Default based on UI theme
    const defaultTheme = (theme === "light" ? "light" : "dark") as CodeTheme;
    document.documentElement.dataset.codeTheme = defaultTheme;
    return defaultTheme;
}

export const CodeThemeToggle = () => {
    const [codeTheme, setCodeTheme] = useState<CodeTheme>("dark");
    const [uiTheme, setUiTheme] = useState<Theme>("dark");

    useEffect(() => {
        // Track UI theme changes
        const updateUiTheme = () => {
            const theme = document.documentElement.dataset.theme as Theme | undefined;
            const newTheme = theme || "dark";
            setUiTheme(newTheme);
            // Resolve appropriate code theme when UI theme changes
            const resolvedTheme = resolveCodeTheme(newTheme);
            setCodeTheme(resolvedTheme);
        };

        // Initial check
        updateUiTheme();

        // Watch for theme changes
        const observer = new MutationObserver(updateUiTheme);
        observer.observe(document.documentElement, {
            attributes: true,
            attributeFilter: ["data-theme"],
        });

        return () => observer.disconnect();
    }, []);

    const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
        const newTheme = e.target.value as CodeTheme;
        setCodeTheme(newTheme);
        // Save to UI-theme-specific storage key
        const storageKey = `codeTheme-${uiTheme}`;
        localStorage.setItem(storageKey, newTheme);
        document.documentElement.dataset.codeTheme = newTheme;
    };

    // Filter themes based on UI theme
    const filteredThemes = Object.entries(THEMES).filter(([_, theme]) => theme.category === uiTheme);

    return (
        <div className='px-3 py-2 rounded-lg border border-border-primary bg-bg-secondary'>
            <div className='flex items-center justify-between gap-3'>
                <label
                    htmlFor='code-theme-select'
                    className='text-sm font-medium text-text-primary'>
                    Code Theme
                </label>
                <select
                    id='code-theme-select'
                    value={codeTheme}
                    onChange={handleChange}
                    className='px-2 py-1 text-xs border border-border-primary rounded bg-bg-primary text-text-primary focus:outline-none focus:ring-1 focus:ring-accent-blue'>
                    {filteredThemes.map(([id, theme]) => (
                        <option
                            key={id}
                            value={id}>
                            {theme.label}
                        </option>
                    ))}
                </select>
            </div>
        </div>
    );
};
