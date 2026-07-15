#ifndef JOSS_DRIVER_H
#define JOSS_DRIVER_H

/*
 * Joss JP v2 in-process ABI C v1.
 *
 * Export joss_driver_call from a DLL, SO or dylib. args_json is always a
 * UTF-8 JSON array and the returned buffer must contain UTF-8 JSON terminated
 * by NUL. Export joss_driver_free when each call allocates its response.
 *
 * The ABI deliberately uses only C strings so libraries built with C, C++,
 * Rust or another C-compatible toolchain can be consumed by the same Joss JP.
 */

#include <stddef.h>

#if defined(_WIN32)
#  if defined(JOSS_DRIVER_BUILD)
#    define JOSS_DRIVER_API __declspec(dllexport)
#  else
#    define JOSS_DRIVER_API __declspec(dllimport)
#  endif
#  define JOSS_DRIVER_CALL __cdecl
#elif defined(__GNUC__) || defined(__clang__)
#  define JOSS_DRIVER_API __attribute__((visibility("default")))
#  define JOSS_DRIVER_CALL
#else
#  define JOSS_DRIVER_API
#  define JOSS_DRIVER_CALL
#endif

#ifdef __cplusplus
extern "C" {
#endif

#define JOSS_DRIVER_ABI_VERSION 1

typedef const char *(JOSS_DRIVER_CALL *joss_driver_call_fn)(
    const char *method,
    const char *args_json
);

typedef void (JOSS_DRIVER_CALL *joss_driver_free_fn)(const char *result);

JOSS_DRIVER_API const char *JOSS_DRIVER_CALL joss_driver_call(
    const char *method,
    const char *args_json
);

/* Optional. Omit this export when joss_driver_call returns static storage. */
JOSS_DRIVER_API void JOSS_DRIVER_CALL joss_driver_free(const char *result);

#ifdef __cplusplus
}
#endif

#endif /* JOSS_DRIVER_H */
