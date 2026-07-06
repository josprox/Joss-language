const vscode = require('vscode');

function getHoverProvider() {
    return vscode.languages.registerHoverProvider('joss', {
        provideHover(document, position, token) {
            const range = document.getWordRangeAtPosition(position);
            const word = document.getText(range);

            // Core Classes
            if (word === 'Router') return new vscode.Hover('**Router Class**\n\nHandles HTTP routing, middleware, and groups.');
            if (word === 'Auth') return new vscode.Hover('**Auth Class**\n\nManages user authentication, sessions, and roles.');
            if (word === 'Request') return new vscode.Hover('**Request Class**\n\nProvides access to HTTP request data (inputs, files).');
            if (word === 'Response') return new vscode.Hover('**Response Class**\n\nHandles HTTP responses (JSON, redirects).');
            if (word === 'View') return new vscode.Hover('**View Class**\n\nRenders HTML templates with inheritance support.');
            if (word === 'System') return new vscode.Hover('**System Class**\n\nProvides access to system commands and environment variables.');
            if (word === 'GranDB') return new vscode.Hover('**GranDB Class**\n\nActive Record ORM for database interactions.');

            // Keywords
            if (word === 'async') return new vscode.Hover('**async**\n\nMarks a function as asynchronous, allowing use of `await`.');
            if (word === 'await') return new vscode.Hover('**await**\n\nPauses execution until the Promise is resolved.');

            return null;
        }
    });
}

module.exports = {
    getHoverProvider
};
