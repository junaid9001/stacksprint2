'use client';

import { useState, useEffect, useRef } from 'react';
import { useConfig } from '@/src/context/ConfigContext';
import { useToast } from '@/components/ui/ToastContainer';

export function SidebarPreview() {
    const { payload } = useConfig();
    const { addToast } = useToast();

    const [bashScript, setBashScript] = useState('');
    const [powerShellScript, setPowerShellScript] = useState('');
    const [filePaths, setFilePaths] = useState<string[]>([]);
    const [warnings, setWarnings] = useState<string[]>([]);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [previewLoading, setPreviewLoading] = useState(false);
    const [activeScriptTab, setActiveScriptTab] = useState<'bash' | 'powershell'>('bash');

    const payloadRef = useRef(payload);

    useEffect(() => {
        payloadRef.current = payload;
    }, [payload]);

    const fetchScripts = async (mode: 'manual' | 'preview', signal?: AbortSignal) => {
        if (mode === 'manual') setLoading(true);
        else setPreviewLoading(true);

        setError('');

        try {
            const api = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
            const res = await fetch(`${api}/generate`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payloadRef.current),
                signal
            });
            const body = await res.json();
            if (!res.ok) {
                setError(body.error || 'Generation failed');
                if (mode === 'manual') addToast(body.error || 'Generation failed', 'error');
                return;
            }
            setBashScript(body.bash_script || '');
            setPowerShellScript(body.powershell_script || '');
            setFilePaths(Array.isArray(body.file_paths) ? body.file_paths : []);
            setWarnings(Array.isArray(body.warnings) ? body.warnings : []);

            if (mode === 'manual') {
                addToast('Scripts generated successfully!', 'success');
            }
        } catch (e) {
            if ((e as Error).name !== 'AbortError') {
                setError(String(e));
                if (mode === 'manual') addToast('Generation aborted or failed.', 'error');
            }
        } finally {
            if (mode === 'manual') setLoading(false);
            else setPreviewLoading(false);
        }
    };

    useEffect(() => {
        const controller = new AbortController();
        const timer = setTimeout(() => {
            fetchScripts('preview', controller.signal);
        }, 500);

        return () => {
            clearTimeout(timer);
            controller.abort();
        };
    }, [payload]);

    const download = (name: string, content: string) => {
        const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = name;
        a.click();
        URL.revokeObjectURL(url);
        addToast(`Downloaded ${name}`, 'info');
    };

    const copyToClipboard = () => {
        const text = activeScriptTab === 'bash' ? bashScript : powerShellScript;
        navigator.clipboard.writeText(text);
        addToast('Copied to clipboard!', 'success');
    };

    return (
        <aside className="panel sticky">
            <article className="section section-animated">
                <div className="section-head">
                    <h2>Generated Output</h2>
                    <span className="hint">Download script and run locally</span>
                </div>
                <p className="hint">After running your script, execute `docker compose up --build` in the generated project.</p>
                <div className="preview-status">{previewLoading ? 'Updating live preview...' : 'Live preview synced'}</div>

                {warnings.length > 0 && (
                    <div className="warning-box">
                        <strong>Configuration Warnings</strong>
                        {warnings.map((warning) => <div key={warning} className="warning-item">- {warning}</div>)}
                    </div>
                )}

                <div className="actions" style={{ marginBottom: '10px' }}>
                    <button className="primary" disabled={loading} onClick={() => fetchScripts('manual')}>
                        {loading ? 'Generating...' : 'Generate Scripts Manually'}
                    </button>
                    {error && <div className="error">{error}</div>}
                </div>

                <div className="download-row">
                    <button className="primary" disabled={!bashScript} onClick={() => download('stacksprint-init.sh', bashScript)}>Download Bash</button>
                    <button className="ghost" disabled={!powerShellScript} onClick={() => download('stacksprint-init.ps1', powerShellScript)}>Download PowerShell</button>
                </div>

                <div className="script-tabs" style={{ display: 'flex', gap: '8px', marginBottom: '8px', alignItems: 'center' }}>
                    <button className={`ghost ${activeScriptTab === 'bash' ? 'active' : ''}`} onClick={() => setActiveScriptTab('bash')}>Bash</button>
                    <button className={`ghost ${activeScriptTab === 'powershell' ? 'active' : ''}`} onClick={() => setActiveScriptTab('powershell')}>PowerShell</button>
                    <div style={{ flex: 1 }} />
                    <button className="ghost copy-btn" onClick={copyToClipboard}>Copy script</button>
                </div>
                <textarea className="script-preview" readOnly value={activeScriptTab === 'bash' ? bashScript : powerShellScript} placeholder="Script preview..." />

                <label>Project Explorer</label>
                <div className="file-tree">
                    {filePaths.length === 0 && <div className="file-tree-empty">No generated paths yet.</div>}
                    {filePaths.map((item) => {
                        const depth = item.split('/').filter(Boolean).length - 1;
                        const isDir = !item.includes('.');
                        return (
                            <div key={item} className="file-tree-row" style={{ paddingLeft: `${depth * 14 + 8}px` }}>
                                <span className="file-tree-icon">{isDir ? 'd' : 'f'}</span>
                                <span>{item}</span>
                            </div>
                        );
                    })}
                </div>
            </article>
        </aside>
    );
}
