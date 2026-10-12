import 'package:flutter/widgets.dart';

import 'gen/app_localizations.dart';

/// All user-facing text goes through `context.l10n` — no string literals in
/// widgets.
extension L10nX on BuildContext {
  AppLocalizations get l10n => AppLocalizations.of(this);
}
