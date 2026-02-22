'use client';

import { useConfig, SchemaField, SchemaModel } from '@/src/context/ConfigContext';

export function SchemaBuilder() {
    const { schemaModels, setSchemaModels } = useConfig();

    function addModel() {
        setSchemaModels((prev: SchemaModel[]) => [...prev, { name: '', fields: [{ name: 'name', type: 'string' }] }]);
    }

    function removeModel(index: number) {
        setSchemaModels((prev: SchemaModel[]) => {
            if (prev.length === 1) return [{ name: '', fields: [{ name: 'name', type: 'string' }] }];
            return prev.filter((_: SchemaModel, i: number) => i !== index);
        });
    }

    function updateModelName(index: number, name: string) {
        setSchemaModels((prev: SchemaModel[]) => prev.map((model: SchemaModel, i: number) => (i === index ? { ...model, name } : model)));
    }

    function addField(modelIndex: number) {
        setSchemaModels((prev: SchemaModel[]) =>
            prev.map((model: SchemaModel, i: number) => (
                i === modelIndex ? { ...model, fields: [...model.fields, { name: '', type: 'string' }] } : model
            ))
        );
    }

    function removeField(modelIndex: number, fieldIndex: number) {
        setSchemaModels((prev: SchemaModel[]) =>
            prev.map((model: SchemaModel, i: number) => {
                if (i !== modelIndex) return model;
                if (model.fields.length === 1) return { ...model, fields: [{ name: '', type: 'string' }] };
                return { ...model, fields: model.fields.filter((_: SchemaField, idx: number) => idx !== fieldIndex) };
            })
        );
    }

    function updateField(modelIndex: number, fieldIndex: number, patch: Partial<SchemaField>) {
        setSchemaModels((prev: SchemaModel[]) =>
            prev.map((model: SchemaModel, i: number) => (
                i === modelIndex
                    ? {
                        ...model,
                        fields: model.fields.map((field: SchemaField, idx: number) => (idx === fieldIndex ? { ...field, ...patch } : field))
                    }
                    : model
            ))
        );
    }

    return (
        <div className="schema-builder">
            <label>Schema Builder</label>
            {schemaModels.map((model: SchemaModel, modelIndex: number) => (
                <div className="schema-model" key={`model-${modelIndex}`}>
                    <div className="row">
                        <input
                            value={model.name}
                            onChange={(e) => updateModelName(modelIndex, e.target.value)}
                            placeholder="Model Name (e.g. User)"
                        />
                        <button type="button" className="ghost" onClick={() => removeModel(modelIndex)}>Remove Model</button>
                    </div>
                    {model.fields.map((field: SchemaField, fieldIndex: number) => (
                        <div className="row schema-field" key={`field-${modelIndex}-${fieldIndex}`}>
                            <input
                                value={field.name}
                                onChange={(e) => updateField(modelIndex, fieldIndex, { name: e.target.value })}
                                placeholder="field name"
                            />
                            <select
                                value={field.type}
                                onChange={(e) => updateField(modelIndex, fieldIndex, { type: e.target.value })}
                            >
                                <option value="string">string</option>
                                <option value="int">int</option>
                                <option value="float">float</option>
                                <option value="bool">bool</option>
                                <option value="datetime">datetime</option>
                            </select>
                            <button type="button" className="ghost" onClick={() => removeField(modelIndex, fieldIndex)}>Remove Field</button>
                        </div>
                    ))}
                    <button type="button" className="ghost" onClick={() => addField(modelIndex)}>Add Field</button>
                </div>
            ))}
            <button type="button" className="ghost" onClick={addModel}>Add Model</button>
        </div>
    );
}
