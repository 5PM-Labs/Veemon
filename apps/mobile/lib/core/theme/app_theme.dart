import 'package:flutter/material.dart';

/// The only place colors are defined. Widgets read
/// `Theme.of(context).colorScheme` — never `Colors.x` or hex literals.
abstract final class AppTheme {
  static const Color _seed = Color(0xFF3D5AFE);

  static ThemeData get light => _build(Brightness.light);

  static ThemeData get dark => _build(Brightness.dark);

  static ThemeData _build(Brightness brightness) {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: _seed,
      brightness: brightness,
    );
    return ThemeData(
      colorScheme: colorScheme,
      useMaterial3: true,
      inputDecorationTheme: const InputDecorationTheme(
        border: OutlineInputBorder(),
      ),
    );
  }
}
