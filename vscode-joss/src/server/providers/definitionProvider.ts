import { DefinitionParams, Definition } from 'vscode-languageserver/node';
import { connection, documents, indexer } from '../server';
import { referenceAtPosition } from '../utils/callContext';

export function setupDefinitionProvider() {
    connection.onDefinition(async (params: DefinitionParams): Promise<Definition | null> => {
        const document = documents.get(params.textDocument.uri);
        if (!document) return null;

        const position = params.position;
        const line = document.getText({
            start: { line: position.line, character: 0 },
            end: { line: position.line, character: 1000 }
        });

        // Check if we're on a Router call like Router::get("/path", "Controller@method")
        const routerMatch = line.match(/Router::(get|post|put|delete|patch)\s*\(\s*["']([^"']+)["']\s*,\s*["'](\w+)@(\w+)["']/);
        if (routerMatch) {
            const [_, method, path, controller, action] = routerMatch;

            // Check if cursor is on controller name
            const controllerStart = line.indexOf(controller, routerMatch.index!);
            const controllerEnd = controllerStart + controller.length;

            if (position.character >= controllerStart && position.character <= controllerEnd) {
                const symbol = await indexer.findSymbol(controller);
                if (symbol) return symbol.location;
            }

            // Check if cursor is on method name
            const methodStart = line.indexOf(action, controllerStart);
            const methodEnd = methodStart + action.length;

            if (position.character >= methodStart && position.character <= methodEnd) {
                const symbol = await indexer.findMethod(controller, action);
                if (symbol) return symbol.location;
            }
        }

        // Fallback: try to find symbol at cursor position
        const word = referenceAtPosition(document, position);

        // Try direct symbol lookup
        let symbol = await indexer.findSymbol(word);
        if (symbol) return symbol.location;

        // Try as Controller.method
        if (word.includes('::') || word.includes('->')) {
            const [controller, method] = word.split(/::|->/);
            symbol = await indexer.findMethod(controller, method);
            if (symbol) return symbol.location;
        }

        const sameName = await indexer.findSymbolsBySimpleName(word.replace(/^\$/, ''));
        if (sameName.length) return sameName[0].location;

        return null;
    });
}
