'use client';

import { useConfig } from '@/src/context/ConfigContext';

export function RootInitForm() {
    const {
        language,
        rootMode, setRootMode,
        rootName, setRootName,
        rootPath, setRootPath,
        moduleName, setModuleName,
        gitInit, setGitInit
    } = useConfig();

    return (
        <article className="section section-animated">
            <div className="section-head">
                <h2>5. Root Initialization</h2>
                <span className="hint">Target directory setup</span>
            </div>
            <div className="field">
                <label>Root mode</label>
                <select value={rootMode} onChange={(e) => setRootMode(e.target.value)}>
                    <option value="new">Create new root folder</option>
                    <option value="existing">Use existing root</option>
                </select>
            </div>
            <div className="field">
                {rootMode === 'new' ? (
                    <input value={rootName} onChange={(e) => setRootName(e.target.value)} placeholder="new root folder name" />
                ) : (
                    <input value={rootPath} onChange={(e) => setRootPath(e.target.value)} placeholder="existing path" />
                )}
            </div>
            <label className="toggle git-toggle">
                <input type="checkbox" checked={gitInit} onChange={(e) => setGitInit(e.target.checked)} />
                <span>Initialize Git repository</span>
            </label>
            {language === 'go' && (
                <div className="field">
                    <input value={moduleName} onChange={(e) => setModuleName(e.target.value)} placeholder="go module name" />
                </div>
            )}
        </article>
    );
}
