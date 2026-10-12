import 'package:freezed_annotation/freezed_annotation.dart';

part 'user_dto.freezed.dart';
part 'user_dto.g.dart';

/// Wire shape of `UserProfile` in `contract/user/user.proto` (protojson,
/// camelCase). Proto3 omits empty strings, hence the defaults.
@freezed
abstract class UserDto with _$UserDto {
  const factory UserDto({
    required String id,
    required String email,
    @Default('') String name,
    @Default('') String phone,
    @Default('') String status,
    String? createdAt,
  }) = _UserDto;

  factory UserDto.fromJson(Map<String, dynamic> json) =>
      _$UserDtoFromJson(json);
}
