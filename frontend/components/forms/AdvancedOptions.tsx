'use client';

import { useState } from 'react';
import { useConfig } from '@/src/context/ConfigContext';

export function AdvancedOptions({ children }: { children: React.ReactNode }) {
    const [isOpen, setIsOpen] = useState(false);
    const { features, fileToggles, customFolders, customFileEntries, removeFolders, removeFiles } = useConfig();

    // Calculate active advanced settings roughly
    const activeFeatures = Object.values(features).filter(Boolean).length;
    const activeToggles = Object.values(fileToggles).filter(Boolean).length;
    let activeCustom = 0;
    if (customFolders) activeCustom++;
    if (customFileEntries.some(f => f.path.trim() !== '')) activeCustom++;
    if (removeFolders) activeCustom++;
    if (removeFiles) activeCustom++;

    const totalActive = activeFeatures + activeToggles + activeCustom;

    return (
        <div className="advanced-accordion">
            <button
                className="advanced-header"
                onClick={() => setIsOpen(!isOpen)}
                aria-expanded={isOpen}
            >
                <div>
                    <strong>Advanced Options</strong>
                    <span className="hint" style={{ marginLeft: '10px' }}>Features, overrides, templates, presets</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    {totalActive > 0 && (
                        <span className="advanced-badge">{totalActive} active</span>
                    )}
                    <span className="chevron">{isOpen ? '▲' : '▼'}</span>
                </div>
            </button>

            {isOpen && (
                <div className="advanced-content">
                    {children}
                </div>
            )}
        </div>
    );
}
