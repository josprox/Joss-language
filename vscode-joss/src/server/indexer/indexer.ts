import { Location, Position, Range } from 'vscode-languageserver/node';
import * as levenshtein from 'fast-levenshtein';
import * as fs from 'fs';
import * as path from 'path';
import { URI } from 'vscode-uri';
import { JossSymbol, parseParameters } from '../languageSymbols';
import { readZipEntry } from '../utils/zipReader';

export type Symbol = JossSymbol;

interface IndexedRoute {
    method: string;
    path: string;
    controller: string;
    action: string;
    location: Location;
}

interface ClassRange {
    name: string;
    start: number;
    end: number;
}

interface PluginSymbolIndex {
    schema: number;
    package: string;
    version: string;
    classes?: Array<{
        name: string;
        super_class?: string;
        methods?: Array<{ name: string; parameters?: Array<{ name: string; type?: string }> }>;
        properties?: string[];
    }>;
    functions?: Array<{ name: string; parameters?: Array<{ name: string; type?: string }> }>;
}

export class Indexer {
    private symbols: Map<string, Symbol[]> = new Map();
    private routes: IndexedRoute[] = [];
    private sources: Map<string, string> = new Map();
    private workspaceRoot = '';

    setWorkspaceRoot(root: string) {
        this.workspaceRoot = root;
    }

    getWorkspaceRoot(): string {
        return this.workspaceRoot;
    }

    getSources(): Array<{ uri: string; content: string }> {
        return Array.from(this.sources, ([uri, content]) => ({ uri, content }));
    }

    async indexWorkspace(): Promise<void> {
        if (!this.workspaceRoot) return;
        this.symbols.clear();
        this.routes = [];
        this.sources.clear();
        await this.scanDirectory(this.workspaceRoot);
    }

    async indexDocument(uri: string, content: string): Promise<void> {
        this.removeUri(uri);
        this.indexText(uri, content);
    }

    private async scanDirectory(dir: string): Promise<void> {
        try {
            const entries = fs.readdirSync(dir, { withFileTypes: true });
            for (const entry of entries) {
                const fullPath = path.join(dir, entry.name);
                if (entry.isDirectory()) {
                    if (!['node_modules', '.git', 'out', 'dist', '.joss-release-work'].includes(entry.name)) {
                        await this.scanDirectory(fullPath);
                    }
                } else if (entry.isFile() && entry.name.endsWith('.joss')) {
                    this.indexText(URI.file(fullPath).toString(), fs.readFileSync(fullPath, 'utf-8'));
                } else if (entry.isFile() && entry.name.endsWith('.jp')) {
                    this.indexPluginArchive(fullPath);
                }
            }
        } catch (error) {
            console.error(`Error scanning directory ${dir}:`, error);
        }
    }

    private indexText(uri: string, content: string): void {
        this.sources.set(uri, content);
        const classRanges: ClassRange[] = [];
        const classRegex = /\bclass\s+([A-Za-z_]\w*)(?:\s+extends\s+([A-Za-z_]\w*))?\s*\{/g;
        let match: RegExpExecArray | null;
        while ((match = classRegex.exec(content)) !== null) {
            const openBrace = content.indexOf('{', match.index);
            const end = findMatchingBrace(content, openBrace);
            const className = match[1];
            classRanges.push({ name: className, start: match.index, end });
            this.addSymbol({
                name: className,
                qualifiedName: className,
                kind: 'class',
                location: locationAt(uri, content, match.index, match[0].length, end),
                parameters: [],
                signature: match[2] ? `class ${className} extends ${match[2]}` : `class ${className}`,
                docstring: extractDocstring(content, match.index),
                bodyEnd: end
            });

            const body = content.substring(openBrace + 1, end);
            const bodyOffset = openBrace + 1;
            this.indexCallables(uri, content, body, bodyOffset, className);
            this.indexProperties(uri, content, body, bodyOffset, className);
        }

        const functionRegex = callableRegex();
        while ((match = functionRegex.exec(content)) !== null) {
            if (classRanges.some(range => match!.index >= range.start && match!.index < range.end)) continue;
            this.addCallable(uri, content, match, undefined, 0);
        }

        const routeRegex = /Router(?:::|\.)(get|post|put|delete|patch|ws)\s*\(\s*["']([^"']+)["']\s*,\s*["'](\w+)@(\w+)["']/g;
        while ((match = routeRegex.exec(content)) !== null) {
            this.routes.push({
                method: match[1].toUpperCase(),
                path: match[2],
                controller: match[3],
                action: match[4],
                location: locationAt(uri, content, match.index, match[0].length)
            });
        }
    }

    private indexPluginArchive(filePath: string): void {
        const uri = URI.file(filePath).toString();
        try {
            const data = readZipEntry(filePath, 'META-INF/joss-symbols.json');
            if (!data) return;
            const index = JSON.parse(data.toString('utf8')) as PluginSymbolIndex;
            if (index.schema !== 1 || !index.package || !index.version) return;
            const location: Location = { uri, range: Range.create(0, 0, 0, 0) };
            for (const pluginClass of index.classes || []) {
                this.addSymbol({
                    name: pluginClass.name,
                    qualifiedName: pluginClass.name,
                    kind: 'class',
                    location,
                    parameters: [],
                    signature: pluginClass.super_class ? `class ${pluginClass.name} extends ${pluginClass.super_class}` : `class ${pluginClass.name}`,
                    origin: 'plugin', packageName: index.package, packageVersion: index.version
                });
                for (const method of pluginClass.methods || []) {
                    this.addPluginCallable(location, index, pluginClass.name, method);
                }
                for (const property of pluginClass.properties || []) {
                    this.addSymbol({
                        name: property,
                        qualifiedName: `${pluginClass.name}->${property}`,
                        kind: 'property', containerName: pluginClass.name, location, parameters: [],
                        signature: `${pluginClass.name}->${property}`,
                        origin: 'plugin', packageName: index.package, packageVersion: index.version
                    });
                }
            }
            for (const fn of index.functions || []) this.addPluginCallable(location, index, undefined, fn);
        } catch (error) {
            console.error(`Error indexing plugin ${filePath}:`, error);
        }
    }

    private addPluginCallable(location: Location, index: PluginSymbolIndex, className: string | undefined, callable: { name: string; parameters?: Array<{ name: string; type?: string }> }): void {
        const parameters = (callable.parameters || []).map(parameter => ({
            name: parameter.name,
            label: `${parameter.type ? `${parameter.type} ` : ''}$${parameter.name}`,
            type: parameter.type,
            optional: false
        }));
        const qualifiedName = className ? `${className}::${callable.name}` : callable.name;
        this.addSymbol({
            name: callable.name, qualifiedName,
            kind: className ? 'method' : 'function', containerName: className,
            location, parameters,
            signature: `${qualifiedName}(${parameters.map(parameter => parameter.label).join(', ')})`,
            docstring: `Exportado por ${index.package} ${index.version}.`,
            origin: 'plugin', packageName: index.package, packageVersion: index.version
        });
    }

    private indexCallables(uri: string, fullContent: string, body: string, bodyOffset: number, className: string): void {
        const regex = callableRegex();
        let match: RegExpExecArray | null;
        while ((match = regex.exec(body)) !== null) {
            this.addCallable(uri, fullContent, match, className, bodyOffset);
        }
    }

    private addCallable(uri: string, content: string, match: RegExpExecArray, className?: string, offset = 0): void {
        const name = match[1];
        const parameters = parseParameters(match[2]);
        const returnType = match[3] || undefined;
        const index = offset + match.index;
        const qualifiedName = className ? `${className}::${name}` : name;
        const signature = `${qualifiedName}(${parameters.map(param => param.label).join(', ')})${returnType ? `: ${returnType}` : ''}`;
        this.addSymbol({
            name,
            qualifiedName,
            kind: className ? 'method' : 'function',
            containerName: className,
            location: locationAt(uri, content, index, match[0].length),
            parameters,
            returnType,
            signature,
            docstring: extractDocstring(content, index)
        });
    }

    private indexProperties(uri: string, content: string, body: string, bodyOffset: number, className: string): void {
        const seen = new Set<string>();
        const regex = /\$this->([A-Za-z_]\w*)\s*=/g;
        let match: RegExpExecArray | null;
        while ((match = regex.exec(body)) !== null) {
            if (seen.has(match[1])) continue;
            seen.add(match[1]);
            const index = bodyOffset + match.index + '$this->'.length;
            this.addSymbol({
                name: match[1],
                qualifiedName: `${className}->${match[1]}`,
                kind: 'property',
                containerName: className,
                location: locationAt(uri, content, index, match[1].length),
                parameters: [],
                signature: `${className}->${match[1]}`
            });
        }
    }

    private removeUri(uri: string): void {
        for (const [key, values] of this.symbols) {
            const remaining = values.filter(symbol => symbol.location.uri !== uri);
            if (remaining.length) this.symbols.set(key, remaining);
            else this.symbols.delete(key);
        }
        this.routes = this.routes.filter(route => route.location.uri !== uri);
        this.sources.delete(uri);
    }

    async findSymbol(name: string): Promise<Symbol | null> {
        const normalized = name.replace('->', '::');
        return this.symbols.get(normalized)?.[0] || this.symbols.get(name)?.[0] || null;
    }

    async findController(name: string): Promise<Location | null> {
        const symbol = await this.findSymbol(name);
        return symbol?.kind === 'class' ? symbol.location : null;
    }

    async findMethod(controller: string, method: string): Promise<Symbol | null> {
        return this.findSymbol(`${controller}::${method}`);
    }

    async getClassMembers(className: string): Promise<Symbol[]> {
        return (await this.getAllSymbols()).filter(symbol => symbol.containerName === className);
    }

    async findSymbolsBySimpleName(name: string): Promise<Symbol[]> {
        return (await this.getAllSymbols()).filter(symbol => symbol.name === name);
    }

    async getClassAtPosition(uri: string, position: Position): Promise<string | undefined> {
        const classes = (await this.getAllSymbols()).filter(symbol => symbol.kind === 'class' && symbol.location.uri === uri);
        return classes.find(symbol => containsPosition(symbol.location.range, position))?.name;
    }

    async fuzzyFindMethod(controller: string, method: string): Promise<string[]> {
        const allMethods = (await this.getClassMembers(controller)).filter(symbol => symbol.kind === 'method').map(symbol => symbol.name);
        return allMethods.map(name => ({ name, distance: levenshtein.get(method, name) }))
            .filter(item => item.distance <= 3).sort((a, b) => a.distance - b.distance).map(item => item.name);
    }

    async getAllSymbols(): Promise<Symbol[]> {
        const unique = new Map<string, Symbol>();
        for (const symbol of Array.from(this.symbols.values()).flat()) {
            const key = `${symbol.qualifiedName}|${symbol.location.uri}|${symbol.location.range.start.line}|${symbol.location.range.start.character}`;
            unique.set(key, symbol);
        }
        return Array.from(unique.values());
    }

    async getAllRoutes(): Promise<IndexedRoute[]> {
        return this.routes;
    }

    async findReferences(reference: string): Promise<Location[]> {
        const needle = reference.includes('::') || reference.includes('->')
            ? reference.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
            : `\\b${reference.replace(/^\$/, '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\b`;
        const regex = new RegExp(needle, 'g');
        const locations: Location[] = [];
        for (const [uri, content] of this.sources) {
            regex.lastIndex = 0;
            let match: RegExpExecArray | null;
            while ((match = regex.exec(content)) !== null) {
                locations.push(locationAt(uri, content, match.index, match[0].length));
            }
        }
        return locations;
    }

    addSymbol(symbol: Symbol) {
        for (const key of new Set([symbol.qualifiedName, symbol.name])) {
            const current = this.symbols.get(key) || [];
            if (!current.some(existing => existing.qualifiedName === symbol.qualifiedName && existing.location.uri === symbol.location.uri && existing.location.range.start.line === symbol.location.range.start.line)) {
                current.push(symbol);
                this.symbols.set(key, current);
            }
        }
    }
}

function callableRegex(): RegExp {
    return /(?:\b(?:public|private|protected|static)\s+)*\b(?:function|func)\s+([A-Za-z_]\w*)\s*\(([^)]*)\)(?:\s*(?::|->)\s*([A-Za-z_][\w|\[\]]*))?/g;
}

function locationAt(uri: string, content: string, index: number, length: number, endOffset?: number): Location {
    const start = positionAt(content, index);
    const end = positionAt(content, endOffset ?? index + length);
    return { uri, range: { start, end } };
}

function positionAt(content: string, offset: number): Position {
    const before = content.substring(0, Math.max(0, offset));
    const lines = before.split('\n');
    return { line: lines.length - 1, character: lines[lines.length - 1].replace(/\r$/, '').length };
}

function containsPosition(range: Range, position: Position): boolean {
    const afterStart = position.line > range.start.line || (position.line === range.start.line && position.character >= range.start.character);
    const beforeEnd = position.line < range.end.line || (position.line === range.end.line && position.character <= range.end.character);
    return afterStart && beforeEnd;
}

function findMatchingBrace(content: string, openBrace: number): number {
    let depth = 0;
    let quote = '';
    let lineComment = false;
    let blockComment = false;
    for (let index = openBrace; index < content.length; index++) {
        const char = content[index];
        const next = content[index + 1];
        if (lineComment) {
            if (char === '\n') lineComment = false;
            continue;
        }
        if (blockComment) {
            if (char === '*' && next === '/') { blockComment = false; index++; }
            continue;
        }
        if (quote) {
            if (char === quote && content[index - 1] !== '\\') quote = '';
            continue;
        }
        if (char === '/' && next === '/') { lineComment = true; index++; continue; }
        if (char === '/' && next === '*') { blockComment = true; index++; continue; }
        if (char === '"' || char === "'") { quote = char; continue; }
        if (char === '{') depth++;
        if (char === '}' && --depth === 0) return index + 1;
    }
    return content.length;
}

function extractDocstring(content: string, declarationIndex: number): string | undefined {
    const before = content.substring(0, declarationIndex).replace(/\s+$/, '');
    const block = before.match(/\/\*\*([\s\S]*?)\*\/\s*$/);
    if (block) {
        return block[1].split('\n').map(line => line.replace(/^\s*\*\s?/, '').trimEnd()).join('\n').trim();
    }
    const lines = before.split('\n');
    const docs: string[] = [];
    for (let index = lines.length - 1; index >= 0; index--) {
        const match = lines[index].match(/^\s*\/\/\/?\s?(.*)$/);
        if (!match) break;
        docs.unshift(match[1]);
    }
    return docs.length ? docs.join('\n').trim() : undefined;
}
