import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:go_router/go_router.dart';

import '../features/auth/presentation/cubits/session_cubit.dart';
import '../features/auth/presentation/cubits/session_state.dart';
import '../features/auth/presentation/screens/login_screen.dart';
import '../features/auth/presentation/screens/profile_screen.dart';
import '../features/auth/presentation/screens/splash_screen.dart';

abstract final class AppRoutes {
  static const String splash = '/splash';
  static const String login = '/login';
  static const String home = '/';
}

/// Routing is a pure function of [SessionCubit] state: screens never push
/// login/home themselves.
GoRouter createRouter(SessionCubit session) => GoRouter(
  initialLocation: AppRoutes.splash,
  refreshListenable: _StreamListenable(session.stream),
  redirect: (_, routerState) {
    final location = routerState.matchedLocation;
    final target = switch (session.state) {
      SessionUnknown() => AppRoutes.splash,
      SessionUnauthenticated() => AppRoutes.login,
      SessionAuthenticated() =>
        location == AppRoutes.splash || location == AppRoutes.login
            ? AppRoutes.home
            : null,
    };
    return target == location ? null : target;
  },
  routes: [
    GoRoute(path: AppRoutes.splash, builder: (_, _) => const SplashScreen()),
    GoRoute(path: AppRoutes.login, builder: (_, _) => const LoginScreen()),
    GoRoute(path: AppRoutes.home, builder: (_, _) => const ProfileScreen()),
  ],
);

class _StreamListenable extends ChangeNotifier {
  _StreamListenable(Stream<Object?> stream) {
    _subscription = stream.listen((_) => notifyListeners());
  }

  late final StreamSubscription<Object?> _subscription;

  @override
  void dispose() {
    unawaited(_subscription.cancel());
    super.dispose();
  }
}
