'use client';

import { useState, useEffect } from 'react';
import { useConfig, PRESET_STORAGE_KEY, SavedPreset } from '@/src/context/ConfigContext';
import { useToast } from '@/components/ui/ToastContainer';

export function PresetManager() {
    const [presetName, setPresetName] = useState('');
    const [presets, setPresets] = useState<SavedPreset[]>([]);
    const { payload, applyPreset } = useConfig();
    const { addToast } = useToast();

    useEffect(() => {
        try {
            const raw = localStorage.getItem(PRESET_STORAGE_KEY);
            if (!raw) return;
            const parsed = JSON.parse(raw) as SavedPreset[];
            if (Array.isArray(parsed)) {
                setPresets(parsed);
            }
        } catch {
            setPresets([]);
        }
    }, []);

    function persistPresets(next: SavedPreset[]) {
        setPresets(next);
        localStorage.setItem(PRESET_STORAGE_KEY, JSON.stringify(next));
    }

    function savePreset() {
        const name = presetName.trim();
        if (!name) {
            addToast('Preset name is required.', 'error');
            return;
        }
        const next = [{ name, config: payload }, ...presets.filter((p) => p.name !== name)];
        persistPresets(next);
        setPresetName('');
        addToast(`Saved preset "${name}"`, 'success');
    }

    function loadPreset(name: string) {
        const match = presets.find((p) => p.name === name);
        if (!match) return;
        applyPreset(match.config);
        addToast(`Loaded preset "${name}"`, 'info');
    }

    function deletePreset(name: string) {
        persistPresets(presets.filter((p) => p.name !== name));
        addToast(`Deleted preset "${name}"`, 'info');
    }

    return (
        <article className="section section-animated">
            <div className="section-head">
                <h2>Preset Library</h2>
                <span className="hint">Save and reuse stack configurations</span>
            </div>
            <div className="row">
                <input
                    value={presetName}
                    onChange={(e) => setPresetName(e.target.value)}
                    placeholder="Preset name (e.g. go-clean-pg)"
                />
                <button type="button" className="ghost" onClick={savePreset}>Save Preset</button>
            </div>
            <div className="preset-list">
                {presets.length === 0 && <div className="hint">No presets saved yet.</div>}
                {presets.map((preset) => (
                    <div key={preset.name} className="preset-item">
                        <span>{preset.name}</span>
                        <div className="preset-actions">
                            <button type="button" className="ghost" onClick={() => loadPreset(preset.name)}>Load</button>
                            <button type="button" className="ghost" onClick={() => deletePreset(preset.name)}>Delete</button>
                        </div>
                    </div>
                ))}
            </div>
        </article>
    );
}
