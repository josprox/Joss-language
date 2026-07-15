import { Position } from 'vscode-languageserver/node';
import { TextDocument } from 'vscode-languageserver-textdocument';

export interface ActiveCall {
    callee: string;
    activeParameter: number;
    openOffset: number;
}

export function findActiveCall(text: string, offset: number): ActiveCall | undefined {
    let nested = 0;
    let quote = '';
    let openOffset = -1;
    for (let index = offset - 1; index >= 0; index--) {
        const char = text[index];
        if (quote) {
            if (char === quote && text[index - 1] !== '\\') quote = '';
            continue;
        }
        if (char === '"' || char === "'") { quote = char; continue; }
        if (char === ')') nested++;
        else if (char === '(') {
            if (nested === 0) { openOffset = index; break; }
            nested--;
        }
    }
    if (openOffset < 0) return undefined;

    const prefix = text.substring(0, openOffset).match(/([A-Za-z_$][\w$]*(?:(?:::|->)[A-Za-z_]\w*)?)\s*$/);
    if (!prefix) return undefined;
    return {
        callee: prefix[1],
        activeParameter: countActiveParameter(text.substring(openOffset + 1, offset)),
        openOffset
    };
}

export function referenceAtPosition(document: TextDocument, position: Position): string {
    const text = document.getText();
    const offset = document.offsetAt(position);
    let start = offset;
    let end = offset;
    while (start > 0 && /[A-Za-z0-9_$:>-]/.test(text[start - 1])) start--;
    while (end < text.length && /[A-Za-z0-9_$:>-]/.test(text[end])) end++;
    return text.substring(start, end).replace(/:+$/, '').replace(/->$/, '');
}

export function inferReceiverClass(text: string, beforeOffset: number, receiver: string): string | undefined {
    if (receiver === '$this') return undefined;
    const escaped = receiver.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const regex = new RegExp(`${escaped}\\s*=\\s*new\\s+([A-Za-z_]\\w*)`, 'g');
    let result: string | undefined;
    let match: RegExpExecArray | null;
    const source = text.substring(0, beforeOffset);
    while ((match = regex.exec(source)) !== null) result = match[1];
    return result;
}

function countActiveParameter(source: string): number {
    let active = 0;
    let depth = 0;
    let quote = '';
    for (let index = 0; index < source.length; index++) {
        const char = source[index];
        if (quote) {
            if (char === quote && source[index - 1] !== '\\') quote = '';
            continue;
        }
        if (char === '"' || char === "'") quote = char;
        else if ('([{'.includes(char)) depth++;
        else if (')]}'.includes(char)) depth--;
        else if (char === ',' && depth === 0) active++;
    }
    return active;
}
