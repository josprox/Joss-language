export interface JossSettings {
    enableJosSecurity: boolean;
}

export function getDefaultSettings(): JossSettings {
    return {
        enableJosSecurity: true
    };
}
