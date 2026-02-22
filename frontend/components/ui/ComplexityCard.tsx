'use client';

import { useEffect, useRef } from 'react';

interface ComplexityReport {
    score: number;
    architecture_weight: number;
    infra_weight: number;
    service_weight: number;
    model_weight: number;
    risk_level: 'low' | 'moderate' | 'high';
    notes: string[];
}

interface Props {
    report: ComplexityReport | null;
}

const RISK_COLORS: Record<string, string> = {
    low: '#22c55e',
    moderate: '#f59e0b',
    high: '#ef4444',
};

const RISK_LABELS: Record<string, string> = {
    low: 'Low',
    moderate: 'Moderate',
    high: 'High',
};

export function ComplexityCard({ report }: Props) {
    const barRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (barRef.current && report) {
            barRef.current.style.width = `${report.score}%`;
        }
    }, [report?.score]);

    if (!report) return null;

    const riskColor = RISK_COLORS[report.risk_level] ?? '#94a3b8';
    const riskLabel = RISK_LABELS[report.risk_level] ?? report.risk_level;

    return (
        <div className="complexity-card">
            <div className="complexity-header">
                <span className="complexity-title">Project Complexity</span>
                <span className="risk-badge" style={{ background: riskColor }}>
                    {riskLabel} Risk
                </span>
            </div>

            {/* Score bar */}
            <div className="complexity-bar-track">
                <div
                    ref={barRef}
                    className="complexity-bar-fill"
                    style={{
                        width: `${report.score}%`,
                        background: riskColor,
                    }}
                />
            </div>
            <div className="complexity-score-row">
                <span className="complexity-label">Score</span>
                <span className="complexity-score">{report.score} / 100</span>
            </div>

            {/* Weight breakdown */}
            <div className="complexity-weights">
                <WeightRow label="Architecture" value={report.architecture_weight} />
                <WeightRow label="Infrastructure" value={report.infra_weight} />
                {report.service_weight > 0 && (
                    <WeightRow label="Services" value={report.service_weight} />
                )}
                {report.model_weight > 0 && (
                    <WeightRow label="Models" value={report.model_weight} />
                )}
            </div>

            {/* Advisory notes */}
            {report.notes && report.notes.length > 0 && (
                <div className="complexity-suggestions">
                    <span className="suggestions-label">Suggestions</span>
                    {report.notes.map((note, i) => (
                        <div key={i} className="suggestion-item">
                            ⚠ {note}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}

function WeightRow({ label, value }: { label: string; value: number }) {
    return (
        <div className="weight-row">
            <span className="weight-label">{label}</span>
            <span className="weight-value">+{value}</span>
        </div>
    );
}
