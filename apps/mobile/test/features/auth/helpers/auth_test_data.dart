import 'package:veemon_mobile/features/auth/data/dtos/login_response_dto.dart';
import 'package:veemon_mobile/features/auth/data/dtos/user_dto.dart';
import 'package:veemon_mobile/features/auth/domain/entities/user_entity.dart';

// Synthetic data only — never real personal data in tests (compliance-pdpa.md).
const tEmail = 'jane@example.test';
const tPassword = 'Secr3t-pass';
const tToken = 'v4.local.test-token';

const tUserDto = UserDto(
  id: '7f6c1b9e-0000-4000-8000-000000000001',
  email: tEmail,
  name: 'Jane Doe',
  phone: '',
  status: 'active',
  createdAt: '2026-01-02T03:04:05Z',
);

final tUser = UserEntity(
  id: '7f6c1b9e-0000-4000-8000-000000000001',
  email: tEmail,
  name: 'Jane Doe',
  phone: '',
  status: 'active',
  createdAt: DateTime.utc(2026, 1, 2, 3, 4, 5),
);

const tLoginResponse = LoginResponseDto(token: tToken, user: tUserDto);
