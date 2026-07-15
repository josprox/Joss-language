package joss.sdk;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.function.Function;

/** Raw JSON runner for joss-rpc-v1. Compatible with GraalVM native-image. */
public final class JossPlugin {
    public static final String PROTOCOL = "joss-rpc-v1";

    private JossPlugin() {}

    public static void run(Function<String, String> dispatch) throws IOException {
        String request = new String(System.in.readAllBytes(), StandardCharsets.UTF_8);
        String response = dispatch.apply(request);
        if (response == null || response.isBlank()) {
            throw new IllegalStateException("dispatch returned an empty JSON response");
        }
        System.out.write(response.getBytes(StandardCharsets.UTF_8));
        System.out.write('\n');
        System.out.flush();
    }
}
