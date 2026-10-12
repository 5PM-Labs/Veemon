import 'package:fpdart/fpdart.dart';
import 'package:mockito/mockito.dart';
import 'package:veemon_mobile/core/error/failures.dart';
import 'package:veemon_mobile/features/auth/domain/entities/user_entity.dart';

/// Mockito needs a dummy for non-nullable generic return types it can't
/// synthesize. Call from `setUpAll`.
void registerDummies() {
  provideDummy<Either<Failure, UserEntity>>(const Left(Failure.unexpected()));
  provideDummy<Either<Failure, Unit>>(const Right(unit));
}
