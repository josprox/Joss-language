import * as fs from 'fs';
import * as path from 'path';
import { URI } from 'vscode-uri';
import { connection, indexer, securityAnalyzer } from '../server';

export function setupCustomRequests() {
    // Manual workspace indexing
    connection.onRequest('workspace/executeCommand', async (params: any) => {
        if (params.command === 'joss.indexWorkspace') {
            await indexer.indexWorkspace();
            return { success: true };
        }
        return { success: false };
    });

    connection.onRequest('joss/getRoutes', async () => {
        return await indexer.getAllRoutes();
    });

    connection.onRequest('joss/resolveRoute', async (route: any) => {
        const controller = await indexer.findController(route.controller);
        return controller;
    });

    connection.onRequest('joss/createController', async (params: { name: string }) => {
        const root = indexer.getWorkspaceRoot();
        if (!root) return { success: false, error: 'No hay un workspace Joss abierto' };

        const controllersDir = path.join(root, 'app', 'controllers');
        const controllerPath = path.join(controllersDir, `${params.name}.joss`);
        if (fs.existsSync(controllerPath)) {
            return { success: false, error: `Ya existe ${path.relative(root, controllerPath)}` };
        }

        fs.mkdirSync(controllersDir, { recursive: true });
        fs.writeFileSync(controllerPath, `class ${params.name} {\n    public func index() {\n        return Response::json({"ok": true})\n    }\n}\n`, 'utf8');
        await indexer.indexDocument(URI.file(controllerPath).toString(), fs.readFileSync(controllerPath, 'utf8'));
        return { success: true, path: controllerPath };
    });

    connection.onRequest('joss/securityCheck', async () => {
        const issues = [];
        for (const source of indexer.getSources()) {
            const sourceIssues = await securityAnalyzer.analyze(source.content);
            issues.push(...sourceIssues.map(issue => ({ ...issue, uri: source.uri })));
        }
        return { issues };
    });
}
