import {
    ParameterInformation,
    SignatureHelp,
    SignatureHelpParams,
    SignatureInformation
} from 'vscode-languageserver/node';
import { connection, documents, indexer } from '../server';
import { findNativeCallable, nativeSignature } from '../nativeCatalog';
import { JossSymbol } from '../languageSymbols';
import { findActiveCall, inferReceiverClass } from '../utils/callContext';

export function setupSignatureHelpProvider() {
    connection.onSignatureHelp(async (params: SignatureHelpParams): Promise<SignatureHelp | null> => {
        const document = documents.get(params.textDocument.uri);
        if (!document) return null;
        const text = document.getText();
        const offset = document.offsetAt(params.position);
        const call = findActiveCall(text, offset);
        if (!call) return null;

        let reference = call.callee;
        if (reference.includes('->')) {
            const [receiver, method] = reference.split('->');
            let className = inferReceiverClass(text, call.openOffset, receiver);
            if (!className && receiver === '$this') {
                className = await indexer.getClassAtPosition(params.textDocument.uri, params.position);
            }
            if (className) reference = `${className}::${method}`;
        }

        const native = findNativeCallable(reference);
        if (native) {
            return result({
                label: nativeSignature(native),
                documentation: native.documentation,
                parameters: native.parameters.map(parameter => ParameterInformation.create(parameter.label, parameter.documentation))
            }, call.activeParameter);
        }

        let symbol = await indexer.findSymbol(reference);
        if (!symbol && !reference.includes('::')) {
            symbol = (await indexer.findSymbolsBySimpleName(reference.replace(/^\$/, ''))).find(item => item.kind !== 'class') || null;
        }
        return symbol ? result(symbolSignature(symbol), call.activeParameter) : null;
    });
}

function symbolSignature(symbol: JossSymbol): SignatureInformation {
    return {
        label: symbol.signature || `${symbol.qualifiedName}()`,
        documentation: symbol.docstring,
        parameters: symbol.parameters.map(parameter => ParameterInformation.create(parameter.label, parameter.documentation))
    };
}

function result(signature: SignatureInformation, activeParameter: number): SignatureHelp {
    return {
        signatures: [signature],
        activeSignature: 0,
        activeParameter: Math.min(activeParameter, Math.max(0, (signature.parameters?.length || 1) - 1))
    };
}
