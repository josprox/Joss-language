#ifndef JOSS_PLUGIN_H
#define JOSS_PLUGIN_H

/*
 * Joss JP v2 native sidecar SDK (joss-rpc-v1)
 *
 * A native plugin is an autonomous executable. Joss writes one UTF-8 JSON
 * request to stdin, closes stdin, and expects one UTF-8 JSON response on
 * stdout. Diagnostic output belongs on stderr.
 *
 * The dispatch callback must return a heap-allocated, NUL-terminated JSON
 * response. The header frees it after writing. This keeps the ABI independent
 * from the compiler, OS dynamic loader and Joss internals.
 */

#include <stdio.h>
#include <stdlib.h>

#ifdef __cplusplus
extern "C" {
#endif

#define JOSS_PLUGIN_PROTOCOL "joss-rpc-v1"
#define JOSS_PLUGIN_ABI_VERSION 1

typedef char *(*joss_plugin_dispatch_fn)(const char *request_json);

static char *joss_plugin_read_request(void) {
    size_t capacity = 4096;
    size_t length = 0;
    char *buffer = (char *)malloc(capacity);
    if (buffer == NULL) return NULL;

    for (;;) {
        if (length + 2048 + 1 > capacity) {
            size_t next_capacity = capacity * 2;
            char *next = (char *)realloc(buffer, next_capacity);
            if (next == NULL) {
                free(buffer);
                return NULL;
            }
            buffer = next;
            capacity = next_capacity;
        }
        size_t count = fread(buffer + length, 1, 2048, stdin);
        length += count;
        if (count < 2048) {
            if (ferror(stdin)) {
                free(buffer);
                return NULL;
            }
            break;
        }
    }
    buffer[length] = '\0';
    return buffer;
}

static int joss_plugin_run(joss_plugin_dispatch_fn dispatch) {
    if (dispatch == NULL) return 64;
    char *request = joss_plugin_read_request();
    if (request == NULL) return 70;
    char *response = dispatch(request);
    free(request);
    if (response == NULL) return 70;
    if (fputs(response, stdout) < 0 || fputc('\n', stdout) == EOF) {
        free(response);
        return 74;
    }
    free(response);
    return fflush(stdout) == 0 ? 0 : 74;
}

#define JOSS_PLUGIN_MAIN(dispatch_function) \
    int main(void) { return joss_plugin_run((dispatch_function)); }

#ifdef __cplusplus
}
#endif

#endif /* JOSS_PLUGIN_H */
