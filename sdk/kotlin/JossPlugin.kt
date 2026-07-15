package joss.sdk

/** Raw JSON runner for joss-rpc-v1. Compile with Kotlin/Native for a standalone payload. */
object JossPlugin {
    const val PROTOCOL = "joss-rpc-v1"

    fun run(dispatch: (String) -> String) {
        val request = generateSequence { readlnOrNull() }.joinToString("\n")
        val response = dispatch(request)
        require(response.isNotBlank()) { "dispatch returned an empty JSON response" }
        println(response)
    }
}
