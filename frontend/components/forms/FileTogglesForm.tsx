'use client';

import { useConfig, fileToggleKeys } from '@/src/context/ConfigContext';

export function FileTogglesForm() {
    const { fileToggles, setFileToggles } = useConfig();

    return (
        <article className="section section-animated">
            <div className="section-head">
                <h2>4. File Toggles</h2>
                <span className="hint">Default generated files</span>
            </div>
            <div className="toggle-grid">
                {fileToggleKeys.map((item) => (
                    <label className="toggle" key={item.key}>
                        <input
                            type="checkbox"
                            checked={fileToggles[item.key] || false}
                            onChange={(e) => setFileToggles({ ...fileToggles, [item.key]: e.target.checked })}
                        />
                        <span>{item.label}</span>
                    </label>
                ))}
            </div>
        </article>
    );
}
