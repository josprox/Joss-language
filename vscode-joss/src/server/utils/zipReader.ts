import * as fs from 'fs';
import { inflateRawSync } from 'zlib';

const endOfCentralDirectory = 0x06054b50;
const centralFileHeader = 0x02014b50;
const localFileHeader = 0x04034b50;
const maxArchiveSize = 256 * 1024 * 1024;
const maxEntrySize = 4 * 1024 * 1024;

export function readZipEntry(archivePath: string, wantedName: string): Buffer | undefined {
    const stat = fs.statSync(archivePath);
    if (stat.size > maxArchiveSize) throw new Error(`JP excede ${maxArchiveSize} bytes`);
    const fd = fs.openSync(archivePath, 'r');
    try {
        const tailSize = Math.min(stat.size, 65_557);
        const tail = readExactly(fd, stat.size - tailSize, tailSize);
        const eocdOffset = findSignatureBackwards(tail, endOfCentralDirectory);
        if (eocdOffset < 0) throw new Error('JP sin directorio ZIP central');
        const entryCount = tail.readUInt16LE(eocdOffset + 10);
        const centralSize = tail.readUInt32LE(eocdOffset + 12);
        const centralOffset = tail.readUInt32LE(eocdOffset + 16);
        if (centralSize > maxArchiveSize || centralOffset + centralSize > stat.size) throw new Error('Directorio ZIP inválido');
        const central = readExactly(fd, centralOffset, centralSize);
        let cursor = 0;
        for (let entry = 0; entry < entryCount && cursor + 46 <= central.length; entry++) {
            if (central.readUInt32LE(cursor) !== centralFileHeader) throw new Error('Entrada ZIP central inválida');
            const compression = central.readUInt16LE(cursor + 10);
            const compressedSize = central.readUInt32LE(cursor + 20);
            const uncompressedSize = central.readUInt32LE(cursor + 24);
            const nameLength = central.readUInt16LE(cursor + 28);
            const extraLength = central.readUInt16LE(cursor + 30);
            const commentLength = central.readUInt16LE(cursor + 32);
            const localOffset = central.readUInt32LE(cursor + 42);
            const name = central.subarray(cursor + 46, cursor + 46 + nameLength).toString('utf8').replace(/\\/g, '/');
            if (name === wantedName) {
                if (uncompressedSize > maxEntrySize || compressedSize > maxEntrySize) throw new Error('Índice de símbolos demasiado grande');
                return readLocalEntry(fd, localOffset, compressedSize, uncompressedSize, compression);
            }
            cursor += 46 + nameLength + extraLength + commentLength;
        }
        return undefined;
    } finally {
        fs.closeSync(fd);
    }
}

function readLocalEntry(fd: number, offset: number, compressedSize: number, expectedSize: number, compression: number): Buffer {
    const header = readExactly(fd, offset, 30);
    if (header.readUInt32LE(0) !== localFileHeader) throw new Error('Entrada ZIP local inválida');
    const nameLength = header.readUInt16LE(26);
    const extraLength = header.readUInt16LE(28);
    const data = readExactly(fd, offset + 30 + nameLength + extraLength, compressedSize);
    const result = compression === 0 ? data : compression === 8 ? inflateRawSync(data) : undefined;
    if (!result) throw new Error(`Compresión ZIP ${compression} no soportada`);
    if (result.length !== expectedSize || result.length > maxEntrySize) throw new Error('Tamaño de entrada ZIP inválido');
    return result;
}

function readExactly(fd: number, position: number, length: number): Buffer {
    const buffer = Buffer.alloc(length);
    let read = 0;
    while (read < length) {
        const count = fs.readSync(fd, buffer, read, length - read, position + read);
        if (count === 0) throw new Error('JP truncado');
        read += count;
    }
    return buffer;
}

function findSignatureBackwards(buffer: Buffer, signature: number): number {
    for (let index = buffer.length - 22; index >= 0; index--) {
        if (buffer.readUInt32LE(index) === signature) return index;
    }
    return -1;
}
