import 'package:freezed_annotation/freezed_annotation.dart';

part 'user_entity.freezed.dart';

/// The signed-in user. Pure Dart — no JSON, no Flutter, no Dio.
@freezed
abstract class UserEntity with _$UserEntity {
  const factory UserEntity({
    required String id,
    required String email,
    required String name,
    required String phone,
    required String status,
    DateTime? createdAt,
  }) = _UserEntity;
}
