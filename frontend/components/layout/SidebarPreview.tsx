'use client';

import { useState, useEffect, useRef } from 'react';
import { useConfig } from '@/src/context/ConfigContext';
import { useToast } from '@/components/ui/ToastContainer';
import { ComplexityCard } from '@/components/ui/ComplexityCard';
import { getFileExplanation } from '@/src/utils/xray';

interface ComplexityReport {
    score: number;
    architecture_weight: number;
    infra_weight: number;
    service_weight: number;
    model_weight: number;
    risk_level: 'low' | 'moderate' | 'high';
    notes: string[];
}

interface Warning {
    code: string;
    severity: string;
    message: string;
    reason: string;
}

export function SidebarPreview() {
    const { payload, services } = useConfig();
    const { addToast } = useToast();

    const [bashScript, setBashScript] = useState('');
    const [filePaths, setFilePaths] = useState<string[]>([]);
    const [prevFileCount, setPrevFileCount] = useState<number | null>(null);
    const [warnings, setWarnings] = useState<Warning[]>([]);
    const [complexityReport, setComplexityReport] = useState<ComplexityReport | null>(null);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [previewLoading, setPreviewLoading] = useState(false);

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
            setFilePaths((prev) => {
                // only set diff if there was a previous successful generation
                if (prev.length > 0) setPrevFileCount(prev.length);
                return Array.isArray(body.file_paths) ? body.file_paths : [];
            });
            setWarnings(Array.isArray(body.warnings) ? body.warnings : []);
            setComplexityReport(body.complexity_report ?? null);

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
        navigator.clipboard.writeText(bashScript);
        addToast('Copied to clipboard!', 'success');
    };

    const copyAll = () => {
        navigator.clipboard.writeText(bashScript);
        addToast('Script copied!', 'success');
    };

    const fileCount = filePaths.length;
    const serviceCount = services?.length ?? 0;

    return (
        <aside className="panel sticky">
            <article className="section section-animated">
                <div className="section-head">
                    <h2>Generated Output</h2>
                    <span className="hint">Download script and run locally</span>
                </div>
                <p className="hint">After running your script, execute `docker compose up --build` in the generated project.</p>
                <div className="preview-status">{previewLoading ? 'Updating live preview...' : 'Live preview synced'}</div>

                {/* Complexity Card — above warnings */}
                <ComplexityCard report={complexityReport} />

                {/* Warnings */}
                {warnings.length > 0 && (
                    <div className="warning-box">
                        <strong>Configuration Warnings</strong>
                        {warnings.map((w) => (
                            <div key={w.code} className="warning-item">
                                {w.severity === 'warn' ? '⚠' : 'ℹ'} {w.message}
                                {w.reason && <span className="warning-reason"> — {w.reason}</span>}
                            </div>
                        ))}
                    </div>
                )}

                <div className="actions" style={{ marginBottom: '10px' }}>
                    <button className="primary" disabled={loading} onClick={() => fetchScripts('manual')}>
                        {loading ? 'Generating...' : 'Generate Scripts Manually'}
                    </button>
                    {error && <div className="error">{error}</div>}
                </div>

                {/* Stats row */}
                {fileCount > 0 && (
                    <div className="output-stats" style={{ display: 'flex', gap: '12px', alignItems: 'center', marginBottom: '12px' }}>
                        <span style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                            📄 {fileCount} file{fileCount !== 1 ? 's' : ''}
                            {prevFileCount !== null && prevFileCount !== fileCount && (
                                <span style={{
                                    color: fileCount > prevFileCount ? 'var(--success)' : 'var(--danger)',
                                    fontSize: '0.75rem',
                                    fontWeight: 600,
                                    background: 'rgba(255,255,255,0.05)',
                                    padding: '2px 6px',
                                    borderRadius: '4px'
                                }}>
                                    {fileCount > prevFileCount ? `+${fileCount - prevFileCount}` : `${fileCount - prevFileCount}`} files
                                </span>
                            )}
                        </span>
                        {serviceCount > 0 && <span>🔲 {serviceCount} service{serviceCount !== 1 ? 's' : ''}</span>}
                    </div>
                )}

                <div className="download-row">
                    <button className="primary" disabled={!bashScript} onClick={() => download('stacksprint-init.sh', bashScript)}>Download Script</button>
                </div>

                <div className="script-tabs" style={{ display: 'flex', gap: '8px', marginBottom: '8px', alignItems: 'center' }}>
                    <span style={{ fontSize: '0.82rem', color: 'var(--muted)', fontWeight: 600 }}>Bash Script <span style={{ fontWeight: 400, opacity: 0.7 }}>(Linux / macOS / WSL / Git Bash)</span></span>
                    <div style={{ flex: 1 }} />
                    <button className="ghost copy-btn" onClick={copyToClipboard}>Copy</button>
                    <button className="ghost copy-btn" onClick={copyAll}>Copy All</button>
                </div>
                <textarea className="script-preview" readOnly value={bashScript} placeholder="Script preview..." />

                <label>Project Explorer</label>
                <div className="file-tree">
                    {filePaths.length === 0 && <div className="file-tree-empty">No generated paths yet.</div>}
                    {filePaths.map((item) => {
                        const depth = item.split('/').filter(Boolean).length - 1;
                        const isDir = !item.includes('.');
                        const explanation = !isDir || item.includes('.gitkeep') || item.includes('docker-compose') || item.includes('Makefile') || item.includes('.env.example') || item.includes('package.json') || item.includes('go.mod') || item.includes('requirements.txt') || item.includes('schema.prisma')
                            ? getFileExplanation(item, payload.architecture)
                            : getFileExplanation(item, payload.architecture);

                        return (
                            <div
                                key={item}
                                className="file-tree-row"
                                style={{ paddingLeft: `${depth * 14 + 8}px` }}
                                title={explanation || undefined}
                            >
                                <span className="file-tree-icon">{isDir ? 'd' : 'f'}</span>
                                <span className={explanation ? 'xray-indicator' : ''}>{item}</span>
                            </div>
                        );
                    })}
                </div>
            </article>
        </aside>
    );
}
