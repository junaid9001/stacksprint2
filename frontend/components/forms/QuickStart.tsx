'use client';

import { useConfig } from '@/src/context/ConfigContext';

export function QuickStart() {
    const { applyPreset } = useConfig();

    const presets = [
        {
            name: 'API Service',
            desc: 'Go · Fiber · PostgreSQL',
            icon: '⚡',
            config: {
                language: 'go', framework: 'fiber', architecture: 'clean', database: 'postgresql',
                useORM: true, rootMode: 'new', rootName: 'api-service', moduleName: 'github.com/user/api-service',
                infra: { redis: false, kafka: false, nats: false }, features: { docker: true, makefile: true, k8s: false },
                services: [], serviceCommunication: 'none'
            }
        },
        {
            name: 'Web Backend',
            desc: 'Node · Express · Modular',
            icon: '🌐',
            config: {
                language: 'node', framework: 'express', architecture: 'modular-monolith', database: 'postgresql',
                useORM: true, rootMode: 'new', rootName: 'web-backend', moduleName: '',
                infra: { redis: true, kafka: false, nats: false }, features: { docker: true, makefile: false, k8s: false },
                services: [], serviceCommunication: 'none'
            }
        },
        {
            name: 'Python API',
            desc: 'FastAPI · Clean · PG',
            icon: '🐍',
            config: {
                language: 'python', framework: 'fastapi', architecture: 'clean', database: 'postgresql',
                useORM: true, rootMode: 'new', rootName: 'python-api', moduleName: '',
                infra: { redis: false, kafka: false, nats: false }, features: { docker: true, makefile: true, k8s: false },
                services: [], serviceCommunication: 'none'
            }
        },
        {
            name: 'Microservices',
            desc: 'Go · Kafka · gRPC',
            icon: '📦',
            config: {
                language: 'go', framework: 'fiber', architecture: 'microservices', database: 'postgresql',
                useORM: false, rootMode: 'new', rootName: 'platform', moduleName: 'github.com/user/platform',
                infra: { redis: true, kafka: true, nats: false }, features: { docker: true, makefile: true, k8s: true },
                services: [
                    { name: 'auth-svc', port: 8081 },
                    { name: 'user-svc', port: 8082 }
                ],
                serviceCommunication: 'grpc'
            }
        }
    ];

    return (
        <div className="quickstart-section">
            <div className="quickstart-header">
                <h2>Quick Start</h2>
                <span className="hint">1-click presets to fill the form</span>
            </div>
            <div className="quickstart-grid">
                {presets.map((p) => (
                    <button
                        key={p.name}
                        type="button"
                        className="quickstart-tile"
                        onClick={() => applyPreset(p.config as any)}
                    >
                        <span className="quickstart-icon">{p.icon}</span>
                        <div className="quickstart-text">
                            <strong>{p.name}</strong>
                            <span>{p.desc}</span>
                        </div>
                    </button>
                ))}
            </div>
        </div>
    );
}
