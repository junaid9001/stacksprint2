'use client';

import { useState } from 'react';
import { useConfig, infraKeys } from '@/src/context/ConfigContext';
import { SchemaBuilder } from './SchemaBuilder';

export function DatabaseInfraForm() {
    const { db, setDb, useORM, setUseORM, infra, setInfra, architecture } = useConfig();
    const [infraOpen, setInfraOpen] = useState(false);

    // If any infra toggle is on, auto-expand the section
    const anyInfraActive = infraKeys.some((item) => infra[item.key]);

    // For MVP: hide advanced infra toggles unless Advanced Mode is enabled (managed via LanguageArchitectureForm)
    const isMvp = architecture === 'mvp';

    const showInfra = infraOpen || anyInfraActive;

    return (
        <article className="section section-animated">
            <div className="section-head">
                <h2>Database & Infrastructure</h2>
                <span className="hint">Runtime data and messaging</span>
            </div>
            <div className="field">
                <label>Database</label>
                <select value={db} onChange={(e) => setDb(e.target.value)}>
                    <option value="postgresql">PostgreSQL</option>
                    <option value="mysql">MySQL</option>
                    <option value="mongodb">MongoDB</option>
                    <option value="none">None</option>
                </select>
            </div>
            <label className="toggle orm-toggle">
                <input type="checkbox" checked={useORM} onChange={(e) => setUseORM(e.target.checked)} />
                <span>Use ORM (GORM / Prisma / SQLAlchemy)</span>
            </label>

            <SchemaBuilder />

            {/* Collapsible Infrastructure Section */}
            <div className="infra-section">
                <button
                    type="button"
                    className="infra-toggle-header"
                    onClick={() => setInfraOpen(!showInfra)}
                    aria-expanded={showInfra}
                >
                    <span>Infrastructure</span>
                    <span className="infra-toggle-icons">
                        {anyInfraActive && (
                            <span className="infra-active-dot" title="Services active" />
                        )}
                        <span className="chevron">{showInfra ? '▲' : '▼'}</span>
                    </span>
                </button>

                {showInfra && (
                    <div className="toggle-grid infra-options">
                        {infraKeys
                            .filter((item) => {
                                // For MVP, hide advanced messaging — unless they're already active
                                if (isMvp && ['kafka', 'nats'].includes(item.key) && !infra[item.key]) {
                                    return false;
                                }
                                return true;
                            })
                            .map((item) => (
                                <label className="toggle" key={item.key}>
                                    <input
                                        type="checkbox"
                                        checked={infra[item.key] || false}
                                        onChange={(e) => setInfra({ ...infra, [item.key]: e.target.checked })}
                                    />
                                    <span>{item.label}</span>
                                </label>
                            ))}
                        {isMvp && (
                            <div className="hint mvp-infra-hint">
                                Kafka and NATS are hidden in MVP mode. Switch to a different architecture to enable them.
                            </div>
                        )}
                    </div>
                )}
            </div>
        </article>
    );
}
