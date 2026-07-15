import { DocumentSymbolParams, DocumentSymbol, SymbolInformation, SymbolKind, Range } from 'vscode-languageserver/node';
import { connection, documents, indexer } from '../server';

export function setupDocumentSymbolProvider() {
    connection.onDocumentSymbol(async (params: DocumentSymbolParams): Promise<DocumentSymbol[]> => {
        const document = documents.get(params.textDocument.uri);
        if (!document) return [];

        const symbols: DocumentSymbol[] = [];
        const text = document.getText();
        const lines = text.split('\n');

        // Find classes
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];

            // Match class declarations
            const classMatch = line.match(/class\s+(\w+)/);
            if (classMatch) {
                const className = classMatch[1];
                const range: Range = {
                    start: { line: i, character: 0 },
                    end: { line: i, character: line.length }
                };

                symbols.push({
                    name: className,
                    kind: SymbolKind.Class,
                    range: range,
                    selectionRange: range,
                    children: []
                });
            }

            // `func` is canonical; `function` remains accepted for legacy files.
            const funcMatch = line.match(/(?:func|function)\s+(\w+)/);
            if (funcMatch) {
                const funcName = funcMatch[1];
                const range: Range = {
                    start: { line: i, character: 0 },
                    end: { line: i, character: line.length }
                };

                symbols.push({
                    name: funcName,
                    kind: SymbolKind.Function,
                    range: range,
                    selectionRange: range
                });
            }
        }

        return symbols;
    });

    connection.onWorkspaceSymbol(async params => {
        const query = params.query.toLowerCase();
        return (await indexer.getAllSymbols())
            .filter(symbol => !query || symbol.name.toLowerCase().includes(query) || symbol.qualifiedName.toLowerCase().includes(query))
            .map(symbol => ({
                name: symbol.qualifiedName,
                kind: symbol.kind === 'class' ? SymbolKind.Class
                    : symbol.kind === 'property' ? SymbolKind.Property
                        : symbol.kind === 'method' ? SymbolKind.Method : SymbolKind.Function,
                location: symbol.location,
                containerName: symbol.containerName
            } as SymbolInformation));
    });
}
