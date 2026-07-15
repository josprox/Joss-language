import { ParameterInfo } from './languageSymbols';

export interface NativeCallable {
    name: string;
    owner?: string;
    parameters: ParameterInfo[];
    returnType?: string;
    documentation: string;
    static?: boolean;
}

function parameter(label: string, documentation?: string): ParameterInfo {
    const optional = label.includes('=');
    const clean = label.split('=')[0].trim();
    const pieces = clean.split(/\s+/);
    const rawName = pieces[pieces.length - 1];
    return {
        name: rawName.replace(/^\$/, ''),
        label,
        type: pieces.length > 1 ? pieces.slice(0, -1).join(' ') : undefined,
        optional,
        defaultValue: optional ? label.substring(label.indexOf('=') + 1).trim() : undefined,
        documentation
    };
}

function callable(owner: string | undefined, name: string, params: string[], returnType: string, documentation: string, isStatic = true): NativeCallable {
    return { owner, name, parameters: params.map(value => parameter(value)), returnType, documentation, static: isStatic };
}

export const nativeCallables: NativeCallable[] = [
    callable(undefined, 'print', ['$value'], 'nil', 'Escribe un valor en la salida estándar.'),
    callable(undefined, 'echo', ['$value'], 'nil', 'Alias de print.'),
    callable(undefined, 'dd', ['$value'], 'nil', 'Muestra un valor y detiene la ejecución.'),
    callable(undefined, 'empty', ['$value'], 'bool', 'Indica si un valor está vacío.'),
    callable(undefined, 'isset', ['$value'], 'bool', 'Indica si un valor existe y no es nil.'),
    callable(undefined, 'count', ['$value'], 'int', 'Devuelve el número de elementos.'),
    callable(undefined, 'await', ['$future'], 'any', 'Espera el resultado de una operación async.'),

    callable('Router', 'get', ['string $path', '$handler'], 'nil', 'Registra una ruta GET.'),
    callable('Router', 'post', ['string $path', '$handler'], 'nil', 'Registra una ruta POST.'),
    callable('Router', 'put', ['string $path', '$handler'], 'nil', 'Registra una ruta PUT.'),
    callable('Router', 'delete', ['string $path', '$handler'], 'nil', 'Registra una ruta DELETE.'),
    callable('Router', 'match', ['string $methods', 'string $path', '$handler'], 'nil', 'Registra varios métodos HTTP para una ruta.'),
    callable('Router', 'ws', ['string $path', '$handler'], 'nil', 'Registra una ruta WebSocket estática.'),
    callable('Router', 'group', ['string $prefix', 'function $callback'], 'nil', 'Agrupa rutas bajo un prefijo.'),
    callable('Router', 'middleware', ['string $name', 'function $callback = nil'], 'nil', 'Aplica middleware a un grupo.'),
    callable('Router', 'api', [], 'nil', 'Activa configuración para rutas API.'),
    callable('Router', 'end', [], 'nil', 'Finaliza el grupo activo.'),

    callable('Auth', 'check', [], 'bool', 'Indica si existe un usuario autenticado.'),
    callable('Auth', 'guest', [], 'bool', 'Indica si la petición es anónima.'),
    callable('Auth', 'user', [], 'User', 'Devuelve la instancia del usuario autenticado.'),
    callable('Auth', 'id', [], 'int|nil', 'Devuelve el identificador del usuario autenticado.'),
    callable('Auth', 'attempt', ['string $email', 'string $password'], 'bool|AuthLoginResult', 'Intenta autenticar credenciales.'),
    callable('Auth', 'hasRole', ['string $role'], 'bool', 'Comprueba el rol del usuario.'),
    callable('Auth', 'logout', [], 'bool', 'Limpia la autenticación del runtime.'),
    callable('Auth', 'validateToken', ['string $token'], 'bool', 'Valida un JWT y restaura la sesión.'),

    callable('Request', 'input', ['string $key', '$default = nil'], 'any', 'Lee un valor de la petición.'),
    callable('Request', 'all', [], 'map', 'Devuelve todos los datos de la petición.'),
    callable('Request', 'except', ['array $keys'], 'map', 'Devuelve los datos excepto las claves indicadas.'),
    callable('Request', 'file', ['string $key'], 'map|nil', 'Obtiene un archivo subido.'),
    callable('Request', 'get', ['string $key', '$default = nil'], 'any', 'Lee un parámetro GET.'),
    callable('Request', 'post', ['string $key', '$default = nil'], 'any', 'Lee un parámetro POST.'),
    callable('Request', 'cookie', ['string $key', '$default = nil'], 'string|nil', 'Lee una cookie.'),

    callable('Response', 'json', ['$data', 'int $status = 200'], 'WebResponse', 'Crea una respuesta JSON.'),
    callable('Response', 'redirect', ['string $url', 'int $status = 302'], 'WebResponse', 'Crea una redirección HTTP.'),
    callable('Response', 'error', ['string $message', 'int $status = 400'], 'WebResponse', 'Crea una respuesta de error.'),
    callable('Response', 'raw', ['$content', 'int $status', 'string $mime', 'map $headers = {}'], 'WebResponse', 'Crea una respuesta sin transformación.'),
    callable('Response', 'stream', ['function $callback', 'string $mime = "text/event-stream"'], 'WebResponse', 'Crea una respuesta transmitida.'),
    callable('Redirect', 'to', ['string $url', 'int $status = 302'], 'WebResponse', 'Alias de redirección con estado explícito.'),
    callable('WebResponse', 'withCookie', ['string $name', 'string $value'], 'WebResponse', 'Añade una cookie.', false),
    callable('WebResponse', 'withHeader', ['string $name', 'string $value'], 'WebResponse', 'Añade un encabezado.', false),
    callable('WebResponse', 'status', ['int $status'], 'WebResponse', 'Cambia el código HTTP.', false),
    callable('WebResponse', 'with', ['string $key', '$value'], 'WebResponse', 'Añade datos a la respuesta.', false),

    callable('View', 'render', ['string $view', 'map $data = {}'], 'WebResponse|string', 'Renderiza una vista Joss.'),
    callable('JSON', 'parse', ['string $json'], 'any', 'Convierte JSON a un valor Joss.'),
    callable('JSON', 'decode', ['string $json'], 'any', 'Alias de parse.'),
    callable('JSON', 'encode', ['$value'], 'string', 'Codifica un valor como JSON.'),
    callable('JSON', 'stringify', ['$value'], 'string', 'Alias de encode.'),
    callable('Math', 'random', ['int $min', 'int $max'], 'int', 'Genera un entero aleatorio.'),
    callable('Math', 'floor', ['number $value'], 'number', 'Redondea hacia abajo.'),
    callable('Math', 'ceil', ['number $value'], 'number', 'Redondea hacia arriba.'),
    callable('Math', 'abs', ['number $value'], 'number', 'Devuelve el valor absoluto.'),
    callable('Str', 'length', ['string $value'], 'int', 'Devuelve la longitud.'),
    callable('Str', 'random', ['int $length = 16'], 'string', 'Genera texto aleatorio.'),
    callable('Str', 'startsWith', ['string $value', 'string $prefix'], 'bool', 'Comprueba el prefijo.'),
    callable('Str', 'substring', ['string $value', 'int $start', 'int $length = nil'], 'string', 'Extrae una sección de texto.'),
    callable('Str', 'indexOf', ['string $value', 'string $search'], 'int', 'Busca texto.'),
    callable('Str', 'contains', ['string $value', 'string $search'], 'bool', 'Comprueba si contiene texto.'),
    callable('Str', 'trim', ['string $value'], 'string', 'Elimina espacios externos.'),

    callable('System', 'env', ['string $key', '$default = nil'], 'any', 'Lee una variable de env.joss.'),
    callable('System', 'Run', ['string $command', 'array $args = []'], 'string|map', 'Ejecuta un proceso del sistema.'),
    callable('System', 'log', ['$value'], 'nil', 'Escribe un diagnóstico.'),
    callable('System', 'sleep', ['number $seconds'], 'nil', 'Pausa la ejecución.'),
    callable('System', 'now', [], 'int', 'Devuelve el tiempo actual.'),
    callable('UUID', 'v4', [], 'string', 'Genera un UUID v4.'),
    callable('UUID', 'generate', [], 'string', 'Genera un UUID.'),

    callable('Plugin', 'call', ['string $plugin', 'string $method', 'array $args = []'], 'any', 'Invoca un payload nativo JP v2.'),
    callable('Plugin', 'stream', ['string $plugin', 'string $method', 'array $args', 'function $callback'], 'any', 'Invoca un payload JP v2 con streaming.'),
    callable('Plugin', 'path', ['string $plugin', 'string $relativePath'], 'string', 'Materializa un recurso del plugin.'),
    callable('Plugin', 'platform', [], 'string', 'Devuelve el target os-arch actual.'),
    callable('Cron', 'schedule', ['string $name', 'string $expression', 'function $callback'], 'nil', 'Programa una tarea cron.'),
    callable('Task', 'on_request', ['string $name', 'function $callback'], 'nil', 'Registra una tarea por petición.'),
    callable('Server', 'spawn', ['string $name', 'string $command', 'int $port'], 'Process', 'Inicia un servicio administrado.'),
    callable('Server', 'start', [], 'nil', 'Inicia los servicios registrados.'),
    callable('Session', 'get', ['string $key', '$default = nil'], 'any', 'Lee una variable de sesión.'),
    callable('Session', 'put', ['string $key', '$value'], 'nil', 'Guarda una variable de sesión.'),
    callable('Session', 'has', ['string $key'], 'bool', 'Comprueba una variable de sesión.'),
    callable('Session', 'forget', ['string $key'], 'nil', 'Elimina una variable de sesión.'),
    callable('Session', 'all', [], 'map', 'Devuelve toda la sesión.'),
    callable('Cache', 'put', ['string $key', '$value', 'int $seconds = 0'], 'nil', 'Guarda un valor en caché.'),
    callable('Cache', 'get', ['string $key', '$default = nil'], 'any', 'Lee un valor de caché.'),
    callable('Cache', 'has', ['string $key'], 'bool', 'Comprueba una clave de caché.'),
    callable('Cache', 'forget', ['string $key'], 'nil', 'Elimina una clave de caché.'),
    callable('GranDB', 'table', ['string $table'], 'GranDB', 'Selecciona la tabla para la consulta.'),
    callable('GranDB', 'select', ['array $columns'], 'GranDB', 'Selecciona las columnas devueltas.'),
    callable('GranDB', 'where', ['string $column', '$operatorOrValue', '$value = nil'], 'GranDB', 'Añade una condición AND.'),
    callable('GranDB', 'orWhere', ['string $column', '$operatorOrValue', '$value = nil'], 'GranDB', 'Añade una condición OR.'),
    callable('GranDB', 'whereIn', ['string $column', 'array $values'], 'GranDB', 'Filtra por una lista de valores.'),
    callable('GranDB', 'whereNull', ['string $column'], 'GranDB', 'Filtra valores nulos.'),
    callable('GranDB', 'whereBetween', ['string $column', '$min', '$max'], 'GranDB', 'Filtra por un intervalo.'),
    callable('GranDB', 'join', ['string $table', 'string $left', 'string $operator', 'string $right'], 'GranDB', 'Añade un JOIN.'),
    callable('GranDB', 'leftJoin', ['string $table', 'string $left', 'string $operator', 'string $right'], 'GranDB', 'Añade un LEFT JOIN.'),
    callable('GranDB', 'get', [], 'array', 'Ejecuta la consulta y devuelve una lista nativa.'),
    callable('GranDB', 'first', [], 'map|nil', 'Devuelve el primer resultado.'),
    callable('GranDB', 'find', ['$id'], 'map|nil', 'Busca un registro por ID.'),
    callable('GranDB', 'value', ['string $column'], 'any', 'Devuelve el valor de una columna.'),
    callable('GranDB', 'pluck', ['string $column', 'string $key = nil'], 'array|map', 'Extrae una columna.'),
    callable('GranDB', 'exists', [], 'bool', 'Indica si la consulta tiene resultados.'),
    callable('GranDB', 'count', ['string $column = "*"'], 'int', 'Cuenta resultados.'),
    callable('GranDB', 'sum', ['string $column'], 'number', 'Suma una columna.'),
    callable('GranDB', 'avg', ['string $column'], 'number', 'Calcula el promedio.'),
    callable('GranDB', 'insert', ['map $data'], 'bool', 'Inserta un registro.'),
    callable('GranDB', 'insertGetId', ['map $data'], 'int', 'Inserta y devuelve el ID.'),
    callable('GranDB', 'update', ['map $data'], 'int', 'Actualiza los registros seleccionados.'),
    callable('GranDB', 'delete', [], 'int', 'Elimina los registros seleccionados.'),
    callable('GranDB', 'orderBy', ['string $column', 'string $direction = "asc"'], 'GranDB', 'Ordena la consulta.'),
    callable('GranDB', 'limit', ['int $count'], 'GranDB', 'Limita resultados.'),
    callable('GranDB', 'offset', ['int $count'], 'GranDB', 'Desplaza resultados.'),
    callable('Schema', 'create', ['string $table', 'function $callback'], 'bool', 'Crea una tabla con Blueprint.'),
    callable('Schema', 'table', ['string $table', 'function $callback'], 'bool', 'Modifica una tabla con Blueprint.'),
    callable('SQLite', 'open', ['string $path'], 'SQLite', 'Abre una base SQLite.'),
    callable('SQLite', 'query', ['string $sql', 'array $params = []'], 'array|bool', 'Ejecuta SQL parametrizado.'),
    callable('SQLite', 'close', [], 'nil', 'Cierra la base SQLite.'),
    callable('WebSocket', 'send', ['$content'], 'bool', 'Envía contenido al cliente.', false),
    callable('WebSocket', 'broadcast', ['$content'], 'bool', 'Difunde contenido a conexiones.', false),
    callable('WebSocket', 'onMessage', ['function $callback'], 'WebSocket', 'Registra el callback de mensajes.', false),
    callable('WebSocket', 'close', [], 'nil', 'Cierra la conexión.', false),
    callable('Stream', 'send', ['$content'], 'bool', 'Envía un fragmento del stream.', false),
    callable('Stream', 'close', [], 'nil', 'Finaliza el stream.', false),
    callable('MFA', 'verify', ['string $code'], 'bool', 'Verifica un desafío MFA.'),
    callable('MFA', 'generateTOTP', ['string $label = nil'], 'map', 'Genera configuración TOTP.'),
    callable('MFA', 'verifyTOTP', ['string $secret', 'string $code'], 'bool', 'Verifica un código TOTP.'),
    callable('MFA', 'generateRecoveryCodes', ['int $count = 8'], 'array', 'Genera códigos de recuperación.'),
    callable('MFA', 'verifyRecoveryCode', ['string $code'], 'bool', 'Verifica un código de recuperación.'),
    callable('UserStorage', 'put', ['string $token', 'string $path', '$content'], 'bool', 'Guarda contenido en almacenamiento de usuario.'),
    callable('UserStorage', 'get', ['string $token', 'string $path'], 'any', 'Lee contenido del almacenamiento de usuario.'),
    callable('UserStorage', 'delete', ['string $token', 'string $path'], 'bool', 'Elimina contenido del almacenamiento de usuario.'),
    callable('Zip', 'extract', ['string $zipPath', 'string $destination'], 'bool', 'Extrae un ZIP.'),
    callable('Markdown', 'toHtml', ['string $markdown'], 'string', 'Convierte Markdown a HTML.'),
    callable('Markdown', 'readFile', ['string $path'], 'string', 'Lee Markdown desde un archivo.'),
    callable('Lang', 'get', ['string $key', 'map $replace = {}'], 'string', 'Obtiene una traducción.'),
    callable('Lang', 'set', ['string $locale'], 'nil', 'Cambia el idioma activo.'),
    callable('Lang', 'locale', [], 'string', 'Devuelve el idioma activo.'),
    callable('Lang', 'locales', [], 'array', 'Lista los idiomas disponibles.')
];

const registeredNativeMethods: Record<string, string[]> = {
    Stack: [], Queue: [],
    GranDB: ['table', 'select', 'where', 'orWhere', 'whereIn', 'orWhereIn', 'whereNotIn', 'whereNull', 'whereNotNull', 'whereBetween', 'whereNotBetween', 'join', 'innerJoin', 'leftJoin', 'rightJoin', 'get', 'first', 'find', 'value', 'pluck', 'exists', 'doesntExist', 'count', 'sum', 'avg', 'min', 'max', 'insert', 'insertGetId', 'update', 'delete', 'deleteAll', 'truncate', 'orderBy', 'latest', 'oldest', 'inRandomOrder', 'limit', 'offset'],
    Auth: ['user', 'check', 'guest', 'id', 'logout', 'attempt', 'create', 'hasRole', 'verify', 'refresh', 'delete', 'login', 'complete2FA', 'validateToken'],
    AuthLoginResult: ['require2FA', 'onSuccess', 'onChallenge', 'onFail', 'response'],
    MFA: ['verify', 'policy', 'challenge', 'generateTOTP', 'verifyTOTP', 'generateRecoveryCodes', 'verifyRecoveryCode'],
    TwoFactor: ['verify', 'required', 'challenge'],
    System: ['env', 'Run', 'load_driver', 'log', 'sleep', 'now'],
    Plugin: ['call', 'stream', 'path', 'platform'], Cron: ['schedule'], Task: ['on_request'], View: ['render'],
    Router: ['get', 'post', 'put', 'delete', 'match', 'api', 'group', 'middleware', 'end', 'ws'],
    Redirect: ['to'], Request: ['input', 'post', 'all', 'except', 'get', 'file', 'cookie'],
    Response: ['json', 'redirect', 'error', 'raw', 'stream'], WebResponse: ['with', 'withCookie', 'withHeader', 'status'],
    WebSocket: ['broadcast', 'send', 'onMessage', 'close'], Schema: ['create', 'table'], Blueprint: [], Redis: [], Migration: [], Middleware: [],
    Math: ['random', 'floor', 'ceil', 'abs'], Session: ['get', 'put', 'has', 'forget', 'all'], UUID: ['generate', 'v4'],
    Str: ['length', 'random', 'startsWith', 'substring', 'indexOf', 'contains', 'trim'],
    UserStorage: ['put', 'get', 'getToFile', 'update', 'path', 'exists', 'delete'], SQLite: ['open', 'query', 'close'],
    Zip: ['extract'], JSON: ['parse', 'stringify', 'decode', 'encode'], Markdown: ['toHtml', 'readFile'],
    Cache: ['put', 'get', 'has', 'forget'], Stream: ['send', 'close'],
    Process: ['constructor', 'start', 'wait', 'kill', 'pid', 'stdin', 'stdout_chan', 'stderr_chan'],
    Server: ['start', 'spawn'], Lang: ['get', 'set', 'locale', 'locales'],
    SEO: ['title', 'description', 'keywords', 'og', 'canonical', 'meta', 'render'], Sitemap: ['add', 'generate']
};

for (const [owner, methods] of Object.entries(registeredNativeMethods)) {
    for (const name of methods) {
        if (!nativeCallables.some(item => item.owner === owner && item.name === name)) {
            nativeCallables.push(callable(owner, name, [], 'any', `Método nativo ${owner}::${name}.`));
        }
    }
}

const registeredNativeClasses = Object.keys(registeredNativeMethods);

export const nativeClasses = Array.from(new Set([...registeredNativeClasses, ...nativeCallables.map(item => item.owner).filter((owner): owner is string => !!owner)])).sort();

export function nativeSignature(item: NativeCallable): string {
    const owner = item.owner ? `${item.owner}${item.static === false ? '->' : '::'}` : '';
    const result = `${owner}${item.name}(${item.parameters.map(param => param.label).join(', ')})`;
    return item.returnType ? `${result}: ${item.returnType}` : result;
}

export function findNativeCallable(reference: string): NativeCallable | undefined {
    const normalized = reference.replace('->', '::');
    const separator = normalized.lastIndexOf('::');
    if (separator >= 0) {
        const owner = normalized.substring(0, separator).replace(/^\$/, '');
        const name = normalized.substring(separator + 2);
        return nativeCallables.find(item => item.owner === owner && item.name === name);
    }
    return nativeCallables.find(item => !item.owner && item.name === reference.replace(/^\$/, ''));
}
