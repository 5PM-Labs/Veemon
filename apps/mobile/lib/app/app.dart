import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:go_router/go_router.dart';

import '../core/di/service_locator.dart';
import '../core/error/failure_message.dart';
import '../core/language/gen/app_localizations.dart';
import '../core/snackbar/global_snack_bar_cubit.dart';
import '../core/snackbar/global_snack_bar_state.dart';
import '../core/theme/app_theme.dart';
import '../features/auth/presentation/cubits/session_cubit.dart';
import '../features/auth/presentation/cubits/session_state.dart';
import 'router.dart';

class App extends StatelessWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiBlocProvider(
      providers: [
        BlocProvider(create: (_) => getIt<SessionCubit>()..restore()),
        BlocProvider(create: (_) => getIt<GlobalSnackBarCubit>()),
      ],
      child: const _AppView(),
    );
  }
}

class _AppView extends StatefulWidget {
  const _AppView();

  @override
  State<_AppView> createState() => _AppViewState();
}

class _AppViewState extends State<_AppView> {
  final _messengerKey = GlobalKey<ScaffoldMessengerState>();
  late final GoRouter _router = createRouter(context.read<SessionCubit>());

  @override
  void dispose() {
    _router.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      onGenerateTitle: (context) => AppLocalizations.of(context).appTitle,
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      routerConfig: _router,
      scaffoldMessengerKey: _messengerKey,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: AppLocalizations.supportedLocales,
      builder: (context, child) => MultiBlocListener(
        listeners: [
          // Session ended with a reason → route it to the global snackbar.
          BlocListener<SessionCubit, SessionState>(
            listener: (context, state) {
              if (state case SessionUnauthenticated(:final failure?)) {
                context.read<GlobalSnackBarCubit>().showFailure(failure);
              }
            },
          ),
          BlocListener<GlobalSnackBarCubit, GlobalSnackBarState>(
            listener: (context, state) {
              if (state case GlobalSnackBarFailure(:final failure)) {
                _messengerKey.currentState
                  ?..hideCurrentSnackBar()
                  ..showSnackBar(
                    SnackBar(content: Text(failure.message(context))),
                  );
              }
            },
          ),
        ],
        child: child ?? const SizedBox.shrink(),
      ),
    );
  }
}
