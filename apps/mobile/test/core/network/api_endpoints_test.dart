// Keeps the hand-written Dart endpoint constants in sync with the generated Go
// route table (`apps/api/handler/grpc/user/user_fiber.pb.go`), which is itself
// generated from `veemon.route` in contract/user/user.proto.
//
// If this fails, update `lib/core/network/api_endpoints.dart` — do not relax the
// test: a stale path means the app calls a route the server does not serve.
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:veemon_mobile/core/network/api_endpoints.dart';

const _goRouteTable = 'apps/api/handler/grpc/user/user_fiber.pb.go';

void main() {
  test('ApiEndpoints matches the generated Fiber route table', () {
    final routes = _fiberRoutes();

    expect(
      routes,
      containsAll(<String>[
        'POST ${ApiEndpoints.login}',
        'POST ${ApiEndpoints.register}',
        'POST ${ApiEndpoints.refresh}',
        'GET ${ApiEndpoints.me}',
        'POST ${ApiEndpoints.logout}',
      ]),
      reason: 'Dart endpoints must exist in the Go route table ($_goRouteTable)',
    );
  });
}

/// Parses `router.Verb("/path", ...)` calls from the generated Go file.
Set<String> _fiberRoutes() {
  const routePattern = r'router\.(\w+)\("([^"]+)"';

  return {
    for (final match in RegExp(routePattern).allMatches(_readGoRouteTable()))
      '${match.group(1)!.toUpperCase()} ${match.group(2)}',
  };
}

/// Locates the generated table without assuming the working directory: walk up
/// from cwd (package root under `flutter test`) until the repo root is found.
String _readGoRouteTable() {
  var dir = Directory.current;
  for (var i = 0; i < 6; i++) {
    final candidate = File('${dir.path}/$_goRouteTable');
    if (candidate.existsSync()) return candidate.readAsStringSync();
    if (dir.parent.path == dir.path) break;
    dir = dir.parent;
  }
  fail('Generated route table not found under ${Directory.current.path}; '
      'run `make -C apps/api proto`.');
}
