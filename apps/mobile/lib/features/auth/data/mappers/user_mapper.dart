import '../../domain/entities/user_entity.dart';
import '../dtos/user_dto.dart';

extension UserDtoMapper on UserDto {
  UserEntity toEntity() => UserEntity(
    id: id,
    email: email,
    name: name,
    phone: phone,
    status: status,
    createdAt: createdAt == null ? null : DateTime.tryParse(createdAt!),
  );
}
