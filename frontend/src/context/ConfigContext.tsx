'use client';

import { createContext, useContext, useState, useMemo, ReactNode } from 'react';

export type Service = { name: string; port: number };
export type ToggleItem = { key: string; label: string };
export type CustomFileEntry = { path: string; content: string };
export type SchemaField = { name: string; type: string };
export type SchemaModel = { name: string; fields: SchemaField[] };
export type SavedPreset = { name: string; config: Record<string, unknown> };

export const PRESET_STORAGE_KEY = 'stacksprint_presets_v1';

export const infraKeys: ToggleItem[] = [
    { key: 'redis', label: 'Redis' },
    { key: 'kafka', label: 'Kafka' },
    { key: 'nats', label: 'NATS' }
];

export const featureKeys: ToggleItem[] = [
    { key: 'jwt_auth', label: 'JWT Auth' },
    { key: 'swagger', label: 'Swagger / OpenAPI' },
    { key: 'github_actions_ci', label: 'GitHub Actions CI' },
    { key: 'makefile', label: 'Makefile' },
    { key: 'logger', label: 'Logger Setup' },
    { key: 'global_error_handler', label: 'Global Error Handler' },
    { key: 'health_endpoint', label: 'Health Endpoint' },
    { key: 'sample_test', label: 'Sample Test File' }
];

export const fileToggleKeys: ToggleItem[] = [
    { key: 'env', label: '.env File' },
    { key: 'gitignore', label: '.gitignore File' },
    { key: 'dockerfile', label: 'Dockerfile' },
    { key: 'docker_compose', label: 'docker-compose.yaml' },
    { key: 'readme', label: 'README.md' },
    { key: 'config_loader', label: 'Config Loader' },
    { key: 'logger', label: 'Logger Setup' },
    { key: 'base_route', label: 'Root Routes' },
    { key: 'example_crud', label: 'Demo CRUD Endpoint' },
    { key: 'health_check', label: 'Health Check' }
];

type ConfigState = {
    language: string;
    framework: string;
    architecture: string;
    db: string;
    useORM: boolean;
    serviceCommunication: string;
    services: Service[];
    infra: Record<string, boolean>;
    features: Record<string, boolean>;
    fileToggles: Record<string, boolean>;
    rootMode: string;
    rootName: string;
    rootPath: string;
    moduleName: string;
    gitInit: boolean;
    customFolders: string;
    schemaModels: SchemaModel[];
    customFileEntries: CustomFileEntry[];
    removeFolders: string;
    removeFiles: string;
};

type ConfigContextType = ConfigState & {
    setLanguage: (val: string) => void;
    setFramework: (val: string) => void;
    setArchitecture: (val: string) => void;
    setDb: (val: string) => void;
    setUseORM: (val: boolean) => void;
    setServiceCommunication: (val: string) => void;
    setServices: (val: Service[]) => void;
    setInfra: (val: Record<string, boolean>) => void;
    setFeatures: (val: Record<string, boolean>) => void;
    setFileToggles: (val: Record<string, boolean>) => void;
    setRootMode: (val: string) => void;
    setRootName: (val: string) => void;
    setRootPath: (val: string) => void;
    setModuleName: (val: string) => void;
    setGitInit: (val: boolean) => void;
    setCustomFolders: (val: string) => void;
    setSchemaModels: (val: SchemaModel[] | ((prev: SchemaModel[]) => SchemaModel[])) => void;
    setCustomFileEntries: (val: CustomFileEntry[] | ((prev: CustomFileEntry[]) => CustomFileEntry[])) => void;
    setRemoveFolders: (val: string) => void;
    setRemoveFiles: (val: string) => void;
    payload: Record<string, any>;
    applyPreset: (config: Record<string, any>) => void;
};

const ConfigContext = createContext<ConfigContextType | undefined>(undefined);

export function ConfigProvider({ children }: { children: ReactNode }) {
    const [language, setLanguage] = useState('go');
    const [framework, setFramework] = useState('fiber');
    const [architecture, setArchitecture] = useState('mvp');
    const [db, setDb] = useState('postgresql');
    const [useORM, setUseORM] = useState(true);
    const [serviceCommunication, setServiceCommunication] = useState('none');
    const [services, setServices] = useState<Service[]>([
        { name: 'users', port: 8081 },
        { name: 'orders', port: 8082 }
    ]);
    const [infra, setInfra] = useState<Record<string, boolean>>({ redis: false, kafka: false, nats: false });
    const [features, setFeatures] = useState<Record<string, boolean>>({
        jwt_auth: false,
        swagger: true,
        github_actions_ci: true,
        makefile: true,
        logger: true,
        global_error_handler: true,
        health_endpoint: true,
        sample_test: true
    });
    const [fileToggles, setFileToggles] = useState<Record<string, boolean>>({
        env: true,
        gitignore: true,
        dockerfile: true,
        docker_compose: true,
        readme: true,
        config_loader: true,
        logger: true,
        base_route: true,
        example_crud: true,
        health_check: true
    });
    const [rootMode, setRootMode] = useState('new');
    const [rootName, setRootName] = useState('my-stacksprint-app');
    const [rootPath, setRootPath] = useState('.');
    const [moduleName, setModuleName] = useState('github.com/example/my-stacksprint-app');
    const [gitInit, setGitInit] = useState(true);
    const [customFolders, setCustomFolders] = useState('');
    const [schemaModels, setSchemaModels] = useState<SchemaModel[]>([
        { name: 'Item', fields: [{ name: 'id', type: 'int' }, { name: 'name', type: 'string' }] }
    ]);
    const [customFileEntries, setCustomFileEntries] = useState<CustomFileEntry[]>([{ path: '', content: '' }]);
    const [removeFolders, setRemoveFolders] = useState('');
    const [removeFiles, setRemoveFiles] = useState('');

    const parseCsv = (v: string): string[] => v.split(',').map((s) => s.trim()).filter(Boolean);

    const payload = useMemo(() => ({
        language,
        framework,
        architecture,
        services: architecture === 'microservices' ? services : [],
        db,
        use_orm: useORM,
        service_communication: serviceCommunication,
        infra,
        features,
        file_toggles: fileToggles,
        custom: {
            add_folders: parseCsv(customFolders),
            models: schemaModels
                .filter((model) => model.name.trim() !== '')
                .map((model) => ({
                    name: model.name.trim(),
                    fields: model.fields.filter((field) => field.name.trim() !== '')
                })),
            add_files: customFileEntries
                .filter((item) => item.path.trim() !== '')
                .map((item) => ({ path: item.path.trim(), content: item.content })),
            add_service_names: services.map((s) => s.name),
            remove_files: parseCsv(removeFiles)
        },
        root: {
            mode: rootMode,
            name: rootName,
            path: rootPath,
            git_init: gitInit,
            module: moduleName
        }
    }), [
        language, framework, architecture, services, db, useORM, serviceCommunication,
        infra, features, fileToggles, customFolders, schemaModels, customFileEntries,
        removeFolders, removeFiles, rootMode, rootName, rootPath, gitInit, moduleName
    ]);

    const applyPreset = (config: Record<string, any>) => {
        setLanguage(config.language || 'go');
        setFramework(config.framework || 'fiber');
        setArchitecture(config.architecture || 'mvp');
        setDb(config.db || 'postgresql');
        setUseORM(Boolean(config.use_orm));
        setServiceCommunication(config.service_communication || 'none');

        const cfgServices = config.services || [];
        setServices(cfgServices.length > 0 ? cfgServices : [{ name: 'users', port: 8081 }, { name: 'orders', port: 8082 }]);

        setInfra(config.infra || { redis: false, kafka: false, nats: false });
        setFeatures(config.features || features);
        setFileToggles(config.file_toggles || fileToggles);

        const custom = config.custom || {};
        setCustomFolders(Array.isArray(custom.add_folders) ? custom.add_folders.join(', ') : '');
        const models = Array.isArray(custom.models) ? custom.models : [];
        setSchemaModels(models.length > 0 ? models : [{ name: 'Item', fields: [{ name: 'id', type: 'int' }, { name: 'name', type: 'string' }] }]);
        const addFiles = Array.isArray(custom.add_files) ? custom.add_files : [];
        setCustomFileEntries(addFiles.length > 0 ? addFiles : [{ path: '', content: '' }]);
        setRemoveFolders(Array.isArray(custom.remove_folders) ? custom.remove_folders.join(', ') : '');
        setRemoveFiles(Array.isArray(custom.remove_files) ? custom.remove_files.join(', ') : '');

        const root = config.root || {};
        setRootMode(root.mode || 'new');
        setRootName(root.name || 'my-stacksprint-app');
        setRootPath(root.path || '.');
        setModuleName(root.module || 'github.com/example/my-stacksprint-app');
    };

    return (
        <ConfigContext.Provider value={{
            language, setLanguage,
            framework, setFramework,
            architecture, setArchitecture,
            db, setDb,
            useORM, setUseORM,
            serviceCommunication, setServiceCommunication,
            services, setServices,
            infra, setInfra,
            features, setFeatures,
            fileToggles, setFileToggles,
            rootMode, setRootMode,
            rootName, setRootName,
            rootPath, setRootPath,
            moduleName, setModuleName,
            gitInit, setGitInit,
            customFolders, setCustomFolders,
            schemaModels, setSchemaModels,
            customFileEntries, setCustomFileEntries,
            removeFolders, setRemoveFolders,
            removeFiles, setRemoveFiles,
            payload,
            applyPreset
        }}>
            {children}
        </ConfigContext.Provider>
    );
}

export function useConfig() {
    const context = useContext(ConfigContext);
    if (context === undefined) {
        throw new Error('useConfig must be used within a ConfigProvider');
    }
    return context;
}
