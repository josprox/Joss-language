<?php

declare(strict_types=1);

const JOSS_PLUGIN_PROTOCOL = 'joss-rpc-v1';

/**
 * Runs a JP v2 native sidecar over stdin/stdout.
 *
 * A handler receives positional arguments. A Generator emits streaming chunks
 * and its return value becomes the final result.
 *
 * @param array<string, callable> $methods
 */
function runJossPlugin(array $methods): void
{
    $requestId = '';

    try {
        $raw = stream_get_contents(STDIN);
        if ($raw === false || trim($raw) === '') {
            throw new InvalidArgumentException('empty request');
        }

        $request = json_decode($raw, true, 512, JSON_THROW_ON_ERROR);
        if (!is_array($request)) {
            throw new InvalidArgumentException('request must be a JSON object');
        }

        $requestId = isset($request['id']) ? (string) $request['id'] : '';
        if (($request['protocol'] ?? null) !== JOSS_PLUGIN_PROTOCOL) {
            throw new InvalidArgumentException('unsupported protocol');
        }

        $method = isset($request['method']) ? (string) $request['method'] : '';
        if ($method === '' || !isset($methods[$method]) || !is_callable($methods[$method])) {
            throw new BadMethodCallException('unknown method: ' . $method);
        }

        $args = $request['args'] ?? [];
        if (!is_array($args) || !array_is_list($args)) {
            throw new InvalidArgumentException('args must be a JSON array');
        }

        $result = $methods[$method](...$args);
        if ($result instanceof Generator) {
            foreach ($result as $chunk) {
                writeJossFrame(['id' => $requestId, 'event' => 'chunk', 'content' => $chunk]);
            }
            $result = $result->getReturn();
        }

        writeJossFrame(['id' => $requestId, 'result' => $result]);
    } catch (Throwable $error) {
        writeJossFrame([
            'id' => $requestId,
            'error' => [
                'code' => (new ReflectionClass($error))->getShortName(),
                'message' => $error->getMessage(),
            ],
        ]);
    }
}

/** @param array<string, mixed> $frame */
function writeJossFrame(array $frame): void
{
    fwrite(STDOUT, json_encode($frame, JSON_THROW_ON_ERROR | JSON_UNESCAPED_SLASHES) . PHP_EOL);
    fflush(STDOUT);
}
