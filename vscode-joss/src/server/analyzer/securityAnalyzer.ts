import { Range, DiagnosticSeverity } from 'vscode-languageserver/node';

export interface SecurityIssue {
    severity: 'error' | 'warning';
    range: Range;
    message: string;
    code: string;
}

export class SecurityAnalyzer {
    private rules = [
        {
            id: 'no-eval',
            pattern: /eval\s*\(/g,
            severity: 'error' as const,
            message: 'Uso de eval() detectado - riesgo de inyección de código'
        },
        {
            id: 'sql-injection',
            pattern: /DB::query\s*\(\s*["'].*\$.*["']\s*\)/g,
            severity: 'warning' as const,
            message: 'Posible inyección SQL - usar prepared statements'
        },
        {
            id: 'weak-bcrypt',
            pattern: /bcrypt\s*\([^,]+,\s*(?:[0-9]|1[01])\s*\)/g,
            severity: 'error' as const,
            message: 'Bcrypt con menos de 12 rondas - inseguro'
        }
    ];

    async analyze(text: string): Promise<SecurityIssue[]> {
        const issues: SecurityIssue[] = [];
        const lines = text.split('\n');

        for (let lineNum = 0; lineNum < lines.length; lineNum++) {
            const line = lines[lineNum];

            for (const rule of this.rules) {
                const matches = line.matchAll(rule.pattern);
                for (const match of matches) {
                    const start = match.index || 0;
                    issues.push({
                        severity: rule.severity,
                        range: {
                            start: { line: lineNum, character: start },
                            end: { line: lineNum, character: start + match[0].length }
                        },
                        message: rule.message,
                        code: rule.id
                    });
                }
            }
        }

        return issues;
    }
}
