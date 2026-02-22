'use client';

import { useConfig, featureKeys } from '@/src/context/ConfigContext';

export function OptionalFeaturesForm() {
    const { features, setFeatures } = useConfig();

    return (
        <article className="section section-animated">
            <div className="section-head">
                <h2>Optional Features</h2>
                <span className="hint">Boilerplate extras</span>
            </div>
            <div className="toggle-grid">
                {featureKeys.map((item) => (
                    <label className="toggle" key={item.key}>
                        <input
                            type="checkbox"
                            checked={features[item.key] || false}
                            onChange={(e) => setFeatures({ ...features, [item.key]: e.target.checked })}
                        />
                        <span>{item.label}</span>
                    </label>
                ))}
            </div>
        </article>
    );
}
