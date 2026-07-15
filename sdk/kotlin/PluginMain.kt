import joss.sdk.JossPlugin

private fun jsonString(value: String): String = buildString {
    append('"')
    value.forEach { character ->
        when (character) {
            '\\' -> append("\\\\")
            '"' -> append("\\\"")
            '\n' -> append("\\n")
            '\r' -> append("\\r")
            '\t' -> append("\\t")
            else -> append(character)
        }
    }
    append('"')
}

fun main() = JossPlugin.run { request ->
    val id = Regex("\\\"id\\\"\\s*:\\s*\\\"([^\\\"]*)\\\"")
        .find(request)?.groupValues?.get(1) ?: ""
    "{\"id\":${jsonString(id)},\"result\":\"kotlin-ok\"}"
}
