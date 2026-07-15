<?php

declare(strict_types=1);

require __DIR__ . '/joss_plugin.php';

runJossPlugin([
    'ping' => static fn (): string => 'php-ok',
    'sum' => static fn (int|float $left, int|float $right): int|float => $left + $right,
    'tokens' => static function (string $text): Generator {
        foreach (str_split($text) as $character) {
            yield $character;
        }
        return $text;
    },
]);
