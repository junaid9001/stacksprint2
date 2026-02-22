'use client';

import { useMemo, useState } from 'react';
import { useConfig, Service } from '@/src/context/ConfigContext';

export function LanguageArchitectureForm() {
    const {
        language, setLanguage,
        framework, setFramework,
        architecture, setArchitecture,
        services, setServices,
        serviceCommunication, setServiceCommunication
    } = useConfig();

    // Advanced mode: when false (MVP), hide gRPC in service communication
    const [advancedMode, setAdvancedMode] = useState(false);
    const isMvp = architecture === 'mvp';

    const frameworkChoices = useMemo(() => {
        if (language === 'go') return ['gin', 'fiber'];
        if (language === 'node') return ['express', 'fastify'];
        return ['fastapi', 'django'];
    }, [language]);

    const archHint = useMemo(() => {
        if (services.length > 1 && architecture !== 'microservices') {
            return '💡 Tip: You have multiple services defined. Select "Microservices".';
        }
        if (language === 'go' && architecture === 'mvp') {
            return '💡 Tip: "Clean Architecture" is the industry standard for Go.';
        }
        if (language === 'python' && architecture === 'microservices') {
            return '💡 Tip: Python backends are often easier to maintain as a "Modular Monolith".';
        }
        if (language === 'node' && (architecture === 'mvp' || architecture === 'clean')) {
            return '💡 Tip: Node.js ecosystems thrive with "Hexagonal" or "Modular Monolith".';
        }
        return null;
    }, [language, architecture, services.length]);

    // When architecture changes away from MVP, no need to reset advanced mode
    // When switching TO mvp, collapse advanced options
    const handleArchChange = (val: string) => {
        setArchitecture(val);
        if (val === 'mvp') setAdvancedMode(false);
    };

    return (
        <article className="section section-animated">
            <div className="section-head">
                <h2>Stack</h2>
                <span className="hint">Core language and architecture</span>
            </div>
            <div className="row">
                <div className="field">
                    <label>Language</label>
                    <select
                        value={language}
                        onChange={(e) => {
                            const next = e.target.value;
                            setLanguage(next);
                            setFramework(next === 'go' ? 'fiber' : next === 'node' ? 'express' : 'fastapi');
                        }}
                    >
                        <option value="go">Go</option>
                        <option value="node">Node</option>
                        <option value="python">Python</option>
                    </select>
                </div>
                <div className="field">
                    <label>Framework</label>
                    <select value={framework} onChange={(e) => setFramework(e.target.value)}>
                        {frameworkChoices.map((f) => <option key={f} value={f}>{f}</option>)}
                    </select>
                </div>
            </div>
            <div className="field">
                <label>Architecture</label>
                <select value={architecture} onChange={(e) => handleArchChange(e.target.value)}>
                    <option value="mvp">MVP</option>
                    <option value="clean">Clean Architecture</option>
                    <option value="hexagonal">Hexagonal</option>
                    <option value="modular-monolith">Modular Monolith</option>
                    <option value="microservices">Microservices (2-5)</option>
                </select>
                {archHint && <div className="hint" style={{ marginTop: '6px', color: '#fbbf24' }}>{archHint}</div>}
            </div>

            {/* Advanced Mode toggle — only shown for MVP */}
            {isMvp && (
                <label className="toggle advanced-mode-toggle">
                    <input
                        type="checkbox"
                        checked={advancedMode}
                        onChange={(e) => setAdvancedMode(e.target.checked)}
                    />
                    <span>Advanced Mode <span className="hint">(shows gRPC and messaging options)</span></span>
                </label>
            )}

            <div className={`microservices-panel ${architecture === 'microservices' ? 'open' : ''}`}>
                <div className="stack">
                    {services.map((s: Service, i: number) => (
                        <div className="row service-row" key={`${s.name}-${i}`}>
                            <input
                                value={s.name}
                                onChange={(e) => setServices(services.map((x: Service, idx: number) => idx === i ? { ...x, name: e.target.value } : x))}
                                placeholder="service name"
                            />
                            <input
                                type="number"
                                value={s.port}
                                onChange={(e) => setServices(services.map((x: Service, idx: number) => idx === i ? { ...x, port: Number(e.target.value) } : x))}
                                placeholder="port"
                            />
                        </div>
                    ))}
                    <button
                        type="button"
                        className="ghost"
                        onClick={() => services.length < 5 && setServices([...services, { name: `service-${services.length + 1}`, port: 8080 + services.length + 1 }])}
                    >
                        Add Service
                    </button>
                    <div className="hint">Keep service count between 2 and 5.</div>
                </div>
            </div>

            {/* Service Communication — hide gRPC for MVP unless advanced mode */}
            <div className="field">
                <label>Service communication</label>
                <select value={serviceCommunication} onChange={(e) => setServiceCommunication(e.target.value)}>
                    <option value="none">None</option>
                    <option value="http">HTTP</option>
                    {(!isMvp || advancedMode) && (
                        <option value="grpc">gRPC (+ shared proto)</option>
                    )}
                </select>
            </div>
        </article>
    );
}
