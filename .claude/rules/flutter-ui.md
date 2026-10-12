---
paths:
  - "apps/mobile/**"
---

# Flutter UI — widgets, theme, l10n, accessibility, performance (`apps/mobile`)

## 1. Private widget classes, not `_buildX()` methods

```dart
// ✓ Good: own const constructor, own rebuild boundary, testable
class _ProfileDetails extends StatelessWidget {
  const _ProfileDetails(this.user);
  final UserEntity user;
  @override
  Widget build(BuildContext context) => ListView(...);
}

// ✗ Bad: rebuilt with the parent every time, can't be const
Widget _buildDetails(UserEntity user) => ListView(...);
```

- A widget used by more than one screen of the same feature goes in
  `features/<f>/presentation/widgets/`. One used across features goes in
  `core/widgets/`.
- Screens (`*_screen.dart`) wire providers/listeners and the `Scaffold`.
  Forms and content are separate widgets.
- Widgets receive **entities** or primitives, never DTOs.

## 2. `const` everywhere it compiles

`const` constructors, `EdgeInsets`, `SizedBox`, `Duration`, and literal
lists. `prefer_const_constructors` is enforced. Hoist unchanging expensive
subtrees into `const` widgets instead of rebuilding them.

## 3. Theme only, no hardcoded colors or styles

```dart
// ✓ Good
style: TextStyle(color: Theme.of(context).colorScheme.error)
Text(user.name, style: Theme.of(context).textTheme.headlineSmall)

// ✗ Bad
style: const TextStyle(color: Colors.red, fontSize: 22)
color: Color(0xFF3D5AFE)
```

- Colors, typography, shapes, and component themes are defined only in
  `core/theme/app_theme.dart` (seeded `ColorScheme`, light + dark).
- Use **semantic roles** (`primary`, `error`, `surfaceContainer`,
  `onSurfaceVariant`), not "the blue one".
- Need a value the scheme lacks? Add a `ThemeExtension` in `core/theme/`.
  Don't inline it.

## 4. All text through `context.l10n`

```dart
// ✓ Good
Text(context.l10n.loginTitle)

// ✗ Bad
Text('Sign in')
```

- ARB files live in `lib/core/language/l10n/`. `app_en.arb` is the template,
  and **`app_id.arb` must get the same key in the same PR**.
- Key naming: `<screenOrArea><Thing>` (`loginSubmit`, `profileEmailLabel`,
  `errorNetwork`).
- Use ICU placeholders/plurals instead of string concatenation:
  `"itemsCount": "{count, plural, =0{No items} =1{1 item} other{{count} items}}"`.
- Dates and numbers are formatted with `intl` (`DateFormat.yMMMd(locale)`),
  never with `toString()`.
- Run `make gen` after editing ARB files.

## 5. `context.mounted` after every `await`

```dart
// ✓ Good
onPressed: () async {
  final confirmed = await showDialog<bool>(...);
  if (!context.mounted || confirmed != true) return;
  context.read<SessionCubit>().logout();
}
```

Also inside `catch` blocks and after `Future.delayed`. In a `State`, check
`mounted`.

## 6. Responsive layout

- Layout decisions come from `LayoutBuilder` constraints
  (`constraints.maxWidth >= 600` → two columns). `MediaQuery` is only for
  device facts (`padding`, `viewInsets`, `textScaler`). Prefer
  `MediaQuery.sizeOf(context)` over `MediaQuery.of(context).size` to avoid
  extra rebuilds.
- Wrap screen bodies in `SafeArea`. Make forms scrollable
  (`SingleChildScrollView`) so the keyboard never hides the submit button.
- No fixed heights on containers holding text, because text scales.

## 7. Accessibility (required, not polish)

- Tap targets ≥ 48×48 dp. Material widgets do this by default, so don't
  shrink them with `visualDensity`/`materialTapTargetSize` hacks.
- Icon-only buttons have a `tooltip` (it doubles as the screen-reader label),
  e.g. the logout `IconButton` in `ProfileScreen`.
- Custom tappables/images: `Semantics(label: ..., button: true)` or
  `excludeSemantics` for decorative ones.
- Support text scaling up to 200%. Test with large fonts and don't clamp
  `textScaler`.
- Contrast comes from the `ColorScheme` roles (`onX` on `X`). Don't put
  arbitrary colors on arbitrary surfaces.
- Don't convey state by color alone. Pair the error color with text or an
  icon.

## 8. Forms and inputs

- `TextEditingController`s are owned by a `StatefulWidget` and disposed in
  `dispose()`.
- Use correct `keyboardType`, `textInputAction`, and `autofillHints`, wrapped
  in an `AutofillGroup`, so password managers work.
- Secret fields: `obscureText: true, enableSuggestions: false,
  autocorrect: false`.
- Client-side validation is UX. The **use case** returns
  `Failure.validation(...)`, and the server is the authority.
- Disable the submit button while submitting (`LoginSubmitting`/`LoginSuccess`)
  instead of only showing a spinner.

## 9. Performance

- Long or unbounded lists: `ListView.builder` / `SliverList` with
  `itemExtent` or `prototypeItem` when rows are uniform. Never use `Column` +
  `map` for server data.
- Give list items stable `ValueKey(entity.id)` when items can reorder.
- Images: set `cacheWidth`/`cacheHeight` to the display size. Network images
  get a placeholder and an error widget.
- No heavy work in `build()` (parsing, sorting, regex). Do it in the use case
  or cubit and keep `build` pure.
- Prefer `Opacity` → `FadeTransition`/`AnimatedOpacity`. Avoid `saveLayer`
  triggers (`ShaderMask`, `ClipRRect` with antialias) in scrolling lists.
- Profile in **profile mode** on a real device (`flutter run --profile`),
  never in debug.
