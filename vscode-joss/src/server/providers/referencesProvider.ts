import { Location, ReferenceParams } from 'vscode-languageserver/node';
import { connection, documents, indexer } from '../server';
import { referenceAtPosition } from '../utils/callContext';

export function setupReferencesProvider() {
    connection.onReferences(async (params: ReferenceParams): Promise<Location[]> => {
        const document = documents.get(params.textDocument.uri);
        if (!document) return [];
        const reference = referenceAtPosition(document, params.position);
        if (!reference) return [];
        return indexer.findReferences(reference);
    });
}
