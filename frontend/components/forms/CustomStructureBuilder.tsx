'use client';

import { useConfig, CustomFileEntry } from '@/src/context/ConfigContext';

export function CustomStructureBuilder() {
    const {
        customFolders, setCustomFolders,
        customFileEntries, setCustomFileEntries,
        removeFolders, setRemoveFolders,
        removeFiles, setRemoveFiles
    } = useConfig();

    function updateCustomFile(index: number, patch: Partial<CustomFileEntry>) {
        setCustomFileEntries((prev) => prev.map((entry, i) => (i === index ? { ...entry, ...patch } : entry)));
    }

    function addCustomFileRow() {
        setCustomFileEntries((prev) => [...prev, { path: '', content: '' }]);
    }

    function removeCustomFileRow(index: number) {
        setCustomFileEntries((prev) => {
            if (prev.length === 1) return [{ path: '', content: '' }];
            return prev.filter((_, i) => i !== index);
        });
    }

    return (
        <article className="section section-animated">
            <div className="section-head">
                <h2>6. Custom Structure Builder</h2>
                <span className="hint">Add or remove paths dynamically</span>
            </div>
            <div className="field">
                <label>Add folders (comma-separated)</label>
                <input value={customFolders} onChange={(e) => setCustomFolders(e.target.value)} placeholder="internal/payments, scripts/dev" />
            </div>
            <div className="file-builder">
                <label>Custom files</label>
                {customFileEntries.map((entry, idx) => (
                    <div className="file-item" key={`custom-file-${idx}`}>
                        <input
                            value={entry.path}
                            onChange={(e) => updateCustomFile(idx, { path: e.target.value })}
                            placeholder="File path (e.g. docs/NOTES.md)"
                        />
                        <textarea
                            rows={4}
                            value={entry.content}
                            onChange={(e) => updateCustomFile(idx, { content: e.target.value })}
                            placeholder="File content"
                        />
                        <button type="button" className="ghost" onClick={() => removeCustomFileRow(idx)}>Remove File</button>
                    </div>
                ))}
                <button type="button" className="ghost" onClick={addCustomFileRow}>Add Another File</button>
            </div>
            <div className="field">
                <label>Remove folders (comma-separated)</label>
                <input value={removeFolders} onChange={(e) => setRemoveFolders(e.target.value)} placeholder="internal/logger" />
            </div>
            <div className="field">
                <label>Remove files (comma-separated)</label>
                <input value={removeFiles} onChange={(e) => setRemoveFiles(e.target.value)} placeholder="README.md, .env" />
            </div>
        </article>
    );
}
