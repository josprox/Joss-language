import { Location } from 'vscode-languageserver/node';

export type JossSymbolKind = 'class' | 'method' | 'function' | 'property';

export interface ParameterInfo {
    name: string;
    label: string;
    type?: string;
    optional: boolean;
    defaultValue?: string;
    documentation?: string;
}

export interface JossSymbol {
    name: string;
    qualifiedName: string;
    kind: JossSymbolKind;
    location: Location;
    signature?: string;
    parameters: ParameterInfo[];
    returnType?: string;
    docstring?: string;
    containerName?: string;
    bodyEnd?: number;
    origin?: 'project' | 'plugin';
    packageName?: string;
    packageVersion?: string;
}

export function parseParameters(source: string): ParameterInfo[] {
    return splitArguments(source).filter(Boolean).map(raw => {
        const label = raw.trim();
        const match = label.match(/^(?:(\??[A-Za-z_][\w|\[\]]*)\s+)?(\$[A-Za-z_]\w*)(?:\s*=\s*(.+))?$/);
        if (!match) {
            return { name: label.replace(/^\$/, ''), label, optional: label.includes('=') };
        }
        return {
            name: match[2].substring(1),
            label,
            type: match[1],
            optional: match[3] !== undefined,
            defaultValue: match[3]?.trim()
        };
    });
}

export function splitArguments(source: string): string[] {
    const parts: string[] = [];
    let start = 0;
    let depth = 0;
    let quote = '';
    for (let index = 0; index < source.length; index++) {
        const char = source[index];
        if (quote) {
            if (char === quote && source[index - 1] !== '\\') quote = '';
            continue;
        }
        if (char === '"' || char === "'") {
            quote = char;
        } else if ('([{'.includes(char)) {
            depth++;
        } else if (')]}'.includes(char)) {
            depth--;
        } else if (char === ',' && depth === 0) {
            parts.push(source.substring(start, index));
            start = index + 1;
        }
    }
    parts.push(source.substring(start));
    return parts;
}
