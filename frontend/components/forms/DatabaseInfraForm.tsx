'use client';

import { useConfig, infraKeys } from '@/src/context/ConfigContext';
import { SchemaBuilder } from './SchemaBuilder';

export function DatabaseInfraForm() {
    const { db, setDb, useORM, setUseORM, infra, setInfra } = useConfig();

    return (
        <article className="section section-animated">
            <div className="section-head">
                <h2>2. Database and Infra</h2>
                <span className="hint">Runtime dependencies</span>
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

            <div className="toggle-grid">
                {infraKeys.map((item) => (
                    <label className="toggle" key={item.key}>
                        <input
                            type="checkbox"
                            checked={infra[item.key] || false}
                            onChange={(e) => setInfra({ ...infra, [item.key]: e.target.checked })}
                        />
                        <span>{item.label}</span>
                    </label>
                ))}
            </div>
        </article>
    );
}
