export function getFileExplanation(path: string, arch: string): string | null {
    const p = path.toLowerCase();

    // ── Hexagonal Architecture ──────────────────────────────────────────────
    if (arch === 'hexagonal') {
        if (p.includes('/adapters/') || p.includes('/adapter/')) return 'Concrete implementations for external communication (HTTP, Databases, Message Brokers).';
        if (p.includes('/ports/') || p.includes('/port/')) return 'Interfaces defining how the core domain expects to communicate with the outside world.';
        if (p.includes('/core/') || p.includes('/domain/')) return 'Enterprise business rules and entities, completely isolated from external dependencies.';
        if (p.includes('/services/')) return 'Application specific business logic coordinating the domain entities and ports.';
    }

    // ── Clean Architecture ──────────────────────────────────────────────────
    if (arch === 'clean') {
        if (p.includes('/usecases/') || p.includes('/usecase/')) return 'Application specific business rules. Orchestrates the flow of data to and from entities.';
        if (p.includes('/repositories/') || p.includes('/repository/')) return 'Data access layer bridging domain entities strictly to storage engines.';
        if (p.includes('/entities/') || p.includes('/entity/') || p.includes('/domain/')) return 'Core enterprise business objects and crucial rules. The most stable layer.';
        if (p.includes('/delivery/') || p.includes('/controllers/') || p.includes('/handlers/')) return 'Interface adapters. Converts data from the format most convenient for the use cases to the format for external agency (e.g. Web).';
    }

    // ── Modular Monolith ────────────────────────────────────────────────────
    if (arch === 'modular-monolith') {
        if (p.includes('/modules/') || p.match(/\/internal\/[a-z0-9_]+\//)) {
            if (p.includes('/handlers/')) return 'Module-specific HTTP controllers and request parsers.';
            if (p.includes('/service.go') || p.includes('/service.ts') || p.includes('/service.py')) return 'Module-specific business logic boundary.';
            if (p.includes('/repository')) return 'Module-specific data access layer.';
        }
        if (p.includes('/shared/') || p.includes('/pkg/')) return 'Shared utilities and cross-cutting concerns (logging, errors) utilized by all modules.';
    }

    // ── Microservices ───────────────────────────────────────────────────────
    if (arch === 'microservices') {
        if (p.includes('/pb/') || p.includes('.proto')) return 'Protocol Buffers definition. The RPC contract describing how services communicate.';
        if (p.includes('/config/')) return 'Service-specific configuration loading (handling environment vars globally).';
        if (p.includes('/clients/')) return 'gRPC or HTTP client stubs used to talk to other internal microservices.';
        // General matching for service structure
        if (p.includes('/handlers/')) return 'Entry point controllers processing transport representations (HTTP/gRPC/Kafka).';
        if (p.includes('/repository/')) return 'The persistence layer owned exclusively by this microservice. No DB sharing.';
    }

    // ── MVP ─────────────────────────────────────────────────────────────────
    if (arch === 'mvp') {
        if (p.includes('/handlers/') || p.includes('/controllers/')) return 'Route handlers managing incoming requests and dispatching responsibilities.';
        if (p.includes('/models/')) return 'Database schema representations and basic data validation forms.';
        if (p.includes('/routes/')) return 'Application routing declarations binding URLs to handlers.';
    }

    // ── General / Universal matches ─────────────────────────────────────────
    if (p.includes('docker-compose.yaml')) return 'Container orchestration configuration to spin up your databases and dependencies locally.';
    if (p.includes('makefile')) return 'Convenience scripts and aliases for building, testing, and running the project.';
    if (p.includes('go.mod') || p.includes('package.json') || p.includes('requirements.txt')) return 'Dependency tracking and package manager descriptor.';
    if (p.includes('.env.example')) return 'Template for environment variables required by the application runtime.';
    if (p.includes('/models/')) return 'Data representations and database schema definitions.';
    if (p.includes('.gitkeep')) return 'A placeholder file ensuring the empty directory structure is committed to version control.';
    if (p.includes('database.go') || p.includes('database.ts') || p.includes('database.py') || p.includes('/db/')) return 'Database connection pooling and initialization logic.';
    if (p.includes('schema.prisma')) return 'Prisma ORM schema definition declaring your database shapes and relations.';

    return null;
}
