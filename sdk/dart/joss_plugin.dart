import 'dart:async';
import 'dart:convert';
import 'dart:io';

typedef JossMethod = FutureOr<dynamic> Function(List<dynamic> args);

/// joss-rpc-v1 runner. Build with `dart compile exe` for a standalone payload.
Future<void> runJossPlugin(Map<String, JossMethod> methods) async {
  var requestId = '';
  Map<String, dynamic> response;
  try {
    final text = await stdin.transform(utf8.decoder).join();
    final request = jsonDecode(text) as Map<String, dynamic>;
    requestId = request['id']?.toString() ?? '';
    if (request['protocol'] != 'joss-rpc-v1') {
      throw StateError('unsupported protocol');
    }
    final method = request['method']?.toString() ?? '';
    final handler = methods[method];
    if (handler == null) throw ArgumentError('unknown method: $method');
    final args = (request['args'] as List<dynamic>?) ?? const [];
    response = {'id': requestId, 'result': await handler(args)};
  } catch (error) {
    response = {
      'id': requestId,
      'error': {'code': error.runtimeType.toString(), 'message': error.toString()}
    };
  }
  stdout.writeln(jsonEncode(response));
}
