import 'package:freezed_annotation/freezed_annotation.dart';

part 'user_dto.freezed.dart';
part 'user_dto.g.dart';

/// Wire shape of `UserProfile` in `contract/user/user.proto` (protojson,
/// camelCase). The Go server emits unpopulated fields (`EmitUnpopulated`), but
/// the defaults keep parsing tolerant of other protojson producers.
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
