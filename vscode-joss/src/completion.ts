import * as vscode from 'vscode';
import * as fs from 'fs';
import * as path from 'path';

export function getCompletionItemProvider() {
    return vscode.languages.registerCompletionItemProvider(
        'joss-html',
        {
            provideCompletionItems(document: vscode.TextDocument, position: vscode.Position, token: vscode.CancellationToken, context: vscode.CompletionContext) {
                const linePrefix = document.lineAt(position).text.substr(0, position.character);

                // --- View Variable Suggestions (Intelligent) ---
                if (document.fileName.endsWith('.joss.html') && linePrefix.trim().endsWith('$')) {
                    const viewName = getRelativeViewName(document.fileName);

                    if (viewName) {
                        const vars = scanControllersForViewVariables(viewName);
                        return vars.map(v => createVariable(v, `Passed from Controller: $${v}`));
                    }
                }

                if (linePrefix.endsWith('Router::')) {
                    return [
                        createMethod('get', 'get(path, handler)', 'Defines a GET route.'),
                        createMethod('post', 'post(path, handler)', 'Defines a POST route.'),
                        createMethod('put', 'put(path, handler)', 'Defines a PUT route.'),
                        createMethod('delete', 'delete(path, handler)', 'Defines a DELETE route.'),
                        createMethod('match', 'match(methods, path, handler)', 'Defines a route for multiple methods (e.g. "GET|POST").'),
                        createMethod('middleware', 'middleware(name)', 'Starts a middleware group (e.g. "auth", "guest").'),
                        createMethod('end', 'end()', 'Ends the current middleware group.'),
                        createMethod('api', 'api(path, handler)', 'Defines an API route (GET & POST).'),
                        createMethod('group', 'group(prefix)', 'Groups routes under a prefix.')
                    ];
                }

                if (linePrefix.endsWith('Auth::')) {
                    return [
                        createMethod('check', 'check()', 'Returns true if user is authenticated.'),
                        createMethod('guest', 'guest()', 'Returns true if user is NOT authenticated.'),
                        createMethod('user', 'user()', 'Returns the current user object.'),
                        createMethod('id', 'id()', 'Returns the current user ID.'),
                        createMethod('hasRole', 'hasRole(role)', 'Checks if user has a specific role.'),
                        createMethod('attempt', 'attempt(email, password)', 'Attempts login. Returns true on success.'),
                        createMethod('create', 'create([email, password, name])', 'Creates a new user.'),
                        createMethod('logout', 'logout()', 'Logs out the current user.'),
                        createMethod('verify', 'verify(token)', 'Verifies a user account.'),
                        createMethod('refresh', 'refresh(id)', 'Refreshes JWT token.'),
                        createMethod('update', 'update(id, data)', 'Updates user data.'),
                        createMethod('delete', 'delete(id)', 'Deletes a user.'),
                        createMethod('validateToken', 'validateToken(token)', 'Validates a Bearer token.')
                    ];
                }

                if (linePrefix.endsWith('Request::')) {
                    return [
                        createMethod('input', 'input(key, default)', 'Retrieves a value from request data.'),
                        createMethod('all', 'all()', 'Returns all request data.'),
                        createMethod('file', 'file(key)', 'Retrieves an uploaded file.'),
                        createMethod('get', 'get(key)', 'Retrieves GET parameter.'),
                        createMethod('post', 'post(key)', 'Retrieves POST parameter.'),
                        createMethod('except', 'except(keys)', 'Returns all except specified keys.'),
                        createMethod('cookie', 'cookie(key)', 'Retrieves a cookie value.')
                    ];
                }

                if (linePrefix.endsWith('Response::')) {
                    return [
                        createMethod('json', 'json(data)', 'Returns a JSON response.'),
                        createMethod('redirect', 'redirect(url)', 'Redirects to a URL.'),
                        createMethod('back', 'back()', 'Redirects back to the previous page.'),
                        createMethod('raw', 'raw(data, status, type, headers)', 'Returns a raw response.'),
                        createMethod('error', 'error(msg, code)', 'Returns an error response.')
                    ];
                }

                // Chainable methods for Response
                if (linePrefix.trim().match(/->$/)) {
                    return [
                        createMethod('withCookie', 'withCookie(name, value)', 'Adds a cookie to the response.'),
                        createMethod('withHeader', 'withHeader(key, value)', 'Adds a header to the response.'),
                        createMethod('status', 'status(code)', 'Sets the HTTP status code.')
                    ];
                }

                if (linePrefix.endsWith('View::')) {
                    return [
                        createMethod('render', 'render(viewName, data)', 'Renders an HTML view.')
                    ];
                }

                if (linePrefix.endsWith('System::')) {
                    return [
                        createMethod('Run', 'Run(command, args)', 'Executes a system command.'),
                        createMethod('env', 'env(key, default)', 'Retrieves an environment variable securely.')
                    ];
                }

                if (linePrefix.endsWith('Math::')) {
                    return [
                        createMethod('random', 'random(min, max)', 'Returns random integer.'),
                        createMethod('floor', 'floor(val)', 'Rounds down.'),
                        createMethod('ceil', 'ceil(val)', 'Rounds up.'),
                        createMethod('abs', 'abs(val)', 'Absolute value.')
                    ];
                }

                if (!linePrefix.includes('.') && !linePrefix.includes('::')) {
                    return [
                        createFunction('print', 'print(msg)', 'Prints to stdout.'),
                        createFunction('echo', 'echo(msg)', 'Alias for print.'),
                        createFunction('dd', 'dd(var)', 'Dump and Die.'),

                        // Snippets for converting fake directives to Ternaries
                        createSnippet('@if', '{{ (${1:condition}) ? { ${2} } : {} }}', 'Inserts a JOSS ternary conditional.'),
                        createSnippet('@else', ': { ${1} }', 'Inserts the else part of a ternary.'),
                        createSnippet('@elseif', ': (${1:condition}) ? { ${2} } : ', 'Inserts an else-if ternary chain.'),
                        createSnippet('@foreach', '@foreach(${1:$items} as ${2:$item})\n\t${3}\n@endforeach', 'Iterates over arrays/maps.'),
                        createSnippet('@endforeach', '@endforeach', 'Ends a foreach loop.'),
                        createSnippet('match', 'match (${1:\$var}) {\n\t${2:value} => ${3:result},\n\tdefault => ${4:fallback}\n}', 'Inserts a PHP-style match expression.'),

                        createKeyword('class', 'Defines a new class.'),
                        createKeyword('func', 'Defines a new function.'),
                        createKeyword('Init', 'Defines an initializer/constructor.'),
                        createKeyword('var', 'Defines a variable.'),
                        createKeyword('const', 'Defines a constant.'),
                        createKeyword('return', 'Returns a value.'),
                        createKeyword('if', 'Conditional statement.'),
                        createKeyword('else', 'Else statement.'),
                        createKeyword('for', 'Loop statement.'),
                        createKeyword('foreach', 'Iterates over arrays/maps.'),
                        createKeyword('match', 'PHP-style match expression for strict branching.'),
                        createKeyword('default', 'Fallback match arm.'),
                        createKeyword('async', 'Starts an asynchronous operation.'),
                        createKeyword('await', 'Waits for an asynchronous operation.'),
                        createKeyword('try', 'Try block.'),
                        createKeyword('catch', 'Catch block.'),
                        createKeyword('throw', 'Throw exception.'),

                        // New Classes
                        createClass('WebResponse'),
                        createClass('Queue'),
                        createClass('Task'),
                        createClass('Cron'),
                        createClass('Math'),
                        createClass('JSON'),
                        createClass('System')
                    ];
                }

                return [];
            }
        },
        '.', ':', '>', '$'
    );
}

// --- Helper Functions for Intelligent Scanning ---

function getRelativeViewName(filePath: string): string | null {
    const viewsDir = 'app' + path.sep + 'views';
    const idx = filePath.indexOf(viewsDir);
    if (idx === -1) return null;

    let relative = filePath.substring(idx + viewsDir.length + 1);
    relative = relative.replace('.joss.html', '').replace('.joss', '');
    return relative.replace(/\\/g, '/');
}

function scanControllersForViewVariables(viewName: string): string[] {
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (!workspaceFolders) return [];

    const rootPath = workspaceFolders[0].uri.fsPath;
    const controllersPath = path.join(rootPath, 'app', 'controllers');

    if (!fs.existsSync(controllersPath)) return [];

    const variables = new Set<string>();
    const files = getAllFiles(controllersPath);

    files.forEach(file => {
        const content = fs.readFileSync(file, 'utf-8');
        if (content.includes(`"${viewName}"`) || content.includes(`'${viewName}'`)) {
            const mapRegex = /"([^"]+)"\s*:/g;
            let match;
            while ((match = mapRegex.exec(content)) !== null) {
                variables.add(match[1]);
            }
        }
    });

    return Array.from(variables);
}

function getAllFiles(dirPath: string, arrayOfFiles: string[] = []): string[] {
    const files = fs.readdirSync(dirPath);
    files.forEach(function (file) {
        if (fs.statSync(dirPath + "/" + file).isDirectory()) {
            arrayOfFiles = getAllFiles(dirPath + "/" + file, arrayOfFiles);
        } else {
            if (file.endsWith('.joss')) {
                arrayOfFiles.push(path.join(dirPath, "/", file));
            }
        }
    });
    return arrayOfFiles;
}

// --- Completion Helpers ---

function createMethod(label: string, detail: string, doc: string) {
    const item = new vscode.CompletionItem(label, vscode.CompletionItemKind.Method);
    item.detail = detail;
    item.documentation = new vscode.MarkdownString(doc);
    return item;
}

function createFunction(label: string, detail: string, doc: string) {
    const item = new vscode.CompletionItem(label, vscode.CompletionItemKind.Function);
    item.detail = detail;
    item.documentation = new vscode.MarkdownString(doc);
    return item;
}

function createKeyword(label: string, doc: string) {
    const item = new vscode.CompletionItem(label, vscode.CompletionItemKind.Keyword);
    item.documentation = new vscode.MarkdownString(doc);
    return item;
}

function createClass(label: string) {
    const item = new vscode.CompletionItem(label, vscode.CompletionItemKind.Class);
    return item;
}

function createVariable(label: string, doc: string) {
    const item = new vscode.CompletionItem(label, vscode.CompletionItemKind.Variable);
    item.documentation = new vscode.MarkdownString(doc);
    item.insertText = label;
    return item;
}

function createSnippet(label: string, snippetText: string, doc: string) {
    const item = new vscode.CompletionItem(label, vscode.CompletionItemKind.Snippet);
    item.insertText = new vscode.SnippetString(snippetText);
    item.documentation = new vscode.MarkdownString(doc);
    item.detail = "JOSS Ternary Snippet";
    return item;
}
