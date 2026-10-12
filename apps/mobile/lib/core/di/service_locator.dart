import 'package:get_it/get_it.dart';
import 'package:injectable/injectable.dart';

import 'service_locator.config.dart';

final GetIt getIt = GetIt.instance;

/// Registrations are generated from `@injectable` / `@LazySingleton` /
/// `@module` annotations — never register by hand.
@InjectableInit()
void configureDependencies() => getIt.init();
