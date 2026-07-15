import {
    CompletionItem,
    CompletionItemKind,
    InsertTextFormat,
    TextDocumentPositionParams
} from 'vscode-languageserver/node';
import { connection, documents, indexer } from '../server';
import { nativeCallables, nativeClasses, nativeSignature, NativeCallable } from '../nativeCatalog';
import { JossSymbol } from '../languageSymbols';
import { inferReceiverClass } from '../utils/callContext';

const keywords = ['class', 'function', 'return', 'foreach', 'async', 'await', 'try', 'catch', 'throw', 'new', 'let', 'int', 'float', 'string', 'bool', 'array', 'map', 'nil'];

export function setupCompletionProvider() {
    connection.onCompletion(async (params: TextDocumentPositionParams): Promise<CompletionItem[]> => {
        const document = documents.get(params.textDocument.uri);
        if (!document) return [];
        const text = document.getText();
        const offset = document.offsetAt(params.position);
        const prefix = text.substring(Math.max(0, offset - 300), offset);

        const staticMatch = prefix.match(/([A-Za-z_]\w*)::$/);
        if (staticMatch) {
            return deduplicate([
                ...nativeCallables.filter(item => item.owner === staticMatch[1] && item.static !== false).map(nativeCompletion),
                ...(await indexer.getClassMembers(staticMatch[1])).filter(item => item.kind === 'method').map(symbolCompletion)
            ]);
        }

        const instanceMatch = prefix.match(/(\$[A-Za-z_]\w*|\$this)->$/);
        if (instanceMatch) {
            let className = inferReceiverClass(text, offset, instanceMatch[1]);
            if (!className && instanceMatch[1] === '$this') {
                className = await indexer.getClassAtPosition(params.textDocument.uri, params.position);
            }
            if (className) {
                return deduplicate([
                    ...nativeCallables.filter(item => item.owner === className && item.static === false).map(nativeCompletion),
                    ...(await indexer.getClassMembers(className)).map(symbolCompletion)
                ]);
            }
            return deduplicate([
                ...nativeCallables.filter(item => item.static === false).map(nativeCompletion),
                ...(await indexer.getAllSymbols()).filter(item => item.kind === 'method' || item.kind === 'property').map(symbolCompletion)
            ]);
        }

        const workspaceSymbols = await indexer.getAllSymbols();
        return deduplicate([
            ...nativeClasses.map(name => ({ label: name, kind: CompletionItemKind.Class, detail: 'Clase nativa de Joss' })),
            ...nativeCallables.filter(item => !item.owner).map(nativeCompletion),
            ...workspaceSymbols.filter(item => item.kind === 'class' || item.kind === 'function').map(symbolCompletion),
            ...keywords.map(label => ({ label, kind: CompletionItemKind.Keyword, detail: 'Palabra reservada de Joss' }))
        ]);
    });

    connection.onCompletionResolve((item: CompletionItem): CompletionItem => item);
}

function nativeCompletion(item: NativeCallable): CompletionItem {
    return {
        label: item.name,
        kind: item.owner ? CompletionItemKind.Method : CompletionItemKind.Function,
        detail: nativeSignature(item),
        documentation: item.documentation,
        insertTextFormat: InsertTextFormat.Snippet,
        insertText: snippet(item.name, item.parameters.map(parameter => parameter.name))
    };
}

function symbolCompletion(symbol: JossSymbol): CompletionItem {
    const kind = symbol.kind === 'class' ? CompletionItemKind.Class
        : symbol.kind === 'property' ? CompletionItemKind.Property
            : symbol.kind === 'function' ? CompletionItemKind.Function : CompletionItemKind.Method;
    return {
        label: symbol.name,
        kind,
        detail: symbol.signature,
        documentation: symbol.docstring || (symbol.origin === 'plugin'
            ? `Exportado por ${symbol.packageName} ${symbol.packageVersion}.`
            : `${symbol.kind} de ${symbol.containerName || 'este proyecto'}`),
        insertTextFormat: symbol.kind === 'method' || symbol.kind === 'function' ? InsertTextFormat.Snippet : InsertTextFormat.PlainText,
        insertText: symbol.kind === 'method' || symbol.kind === 'function'
            ? snippet(symbol.name, symbol.parameters.map(parameter => parameter.name))
            : symbol.name
    };
}

function snippet(name: string, parameters: string[]): string {
    if (!parameters.length) return `${name}()`;
    return `${name}(${parameters.map((parameter, index) => `\${${index + 1}:\$${parameter}}`).join(', ')})`;
}

function deduplicate(items: CompletionItem[]): CompletionItem[] {
    const seen = new Set<string>();
    return items.filter(item => {
        const key = `${item.kind}:${item.label}`;
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
    });
}
