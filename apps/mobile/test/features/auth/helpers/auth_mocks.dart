import 'package:mockito/annotations.dart';
import 'package:veemon_mobile/core/storage/token_storage.dart';
import 'package:veemon_mobile/features/auth/data/data_sources/auth_remote_data_source.dart';
import 'package:veemon_mobile/features/auth/domain/repositories/auth_repository.dart';
import 'package:veemon_mobile/features/auth/domain/use_cases/get_current_user.dart';
import 'package:veemon_mobile/features/auth/domain/use_cases/login.dart';
import 'package:veemon_mobile/features/auth/domain/use_cases/logout.dart';

@GenerateNiceMocks([
  MockSpec<AuthRepository>(),
  MockSpec<AuthRemoteDataSource>(),
  MockSpec<TokenStorage>(),
  MockSpec<Login>(),
  MockSpec<GetCurrentUser>(),
  MockSpec<Logout>(),
])
void main() {}
