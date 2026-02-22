'use client';

import { useMemo } from 'react';
import { useConfig, Service } from '@/src/context/ConfigContext';

export function LanguageArchitectureForm() {
    const {
        language, setLanguage,
        framework, setFramework,
        architecture, setArchitecture,
        services, setServices,
        serviceCommunication, setServiceCommunication
    } = useConfig();

    const frameworkChoices = useMemo(() => {
        if (language === 'go') return ['gin', 'fiber'];
        if (language === 'node') return ['express', 'fastify'];
        return ['fastapi', 'django'];
    }, [language]);

    return (
        <article className="section section-animated">
            <div className="section-head">
                <h2>1. Language and Architecture</h2>
                <span className="hint">Core stack selection</span>
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
                <select value={architecture} onChange={(e) => setArchitecture(e.target.value)}>
                    <option value="mvp">MVP</option>
                    <option value="clean">Clean Architecture</option>
                    <option value="hexagonal">Hexagonal</option>
                    <option value="modular-monolith">Modular Monolith</option>
                    <option value="microservices">Microservices (2-5)</option>
                </select>
            </div>

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

            <div className="field">
                <label>Service communication</label>
                <select value={serviceCommunication} onChange={(e) => setServiceCommunication(e.target.value)}>
                    <option value="none">None</option>
                    <option value="http">HTTP</option>
                    <option value="grpc">gRPC (+ shared proto)</option>
                </select>
            </div>
        </article>
    );
}
