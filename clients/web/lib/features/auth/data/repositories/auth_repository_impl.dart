import "dart:convert";

import "package:restick_web/core/error/dio_exception_mapper.dart";
import "package:restick_web/core/storage/secure_storage.dart";
import "package:restick_web/features/auth/data/datasources/auth_api.dart";
import "package:restick_web/features/auth/data/models/user_dto.dart";
import "package:restick_web/features/auth/domain/entities/user.dart";
import "package:restick_web/features/auth/domain/repositories/auth_repository.dart";

class AuthRepositoryImpl implements AuthRepository {
  AuthRepositoryImpl({
    required AuthApi authApi,
    required SecureStorage secureStorage,
  })  : _authApi = authApi,
        _secureStorage = secureStorage;

  final AuthApi _authApi;
  final SecureStorage _secureStorage;

  @override
  Future<({String token, User user})> register({
    required String email,
    required String password,
    required String displayName,
  }) async {
    try {
      final AuthResponseDto response = await _authApi.register(
        <String, dynamic>{
          "email": email,
          "password": password,
          "display_name": displayName,
        },
      );
      final User user = response.user.toEntity();
      await persistSession(token: response.accessToken, user: user);
      return (token: response.accessToken, user: user);
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<({String token, User user})> login({
    required String email,
    required String password,
  }) async {
    try {
      final AuthResponseDto response = await _authApi.login(
        <String, dynamic>{
          "email": email,
          "password": password,
        },
      );
      final User user = response.user.toEntity();
      await persistSession(token: response.accessToken, user: user);
      return (token: response.accessToken, user: user);
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<User> me() async {
    try {
      final UserDto dto = await _authApi.me();
      return dto.toEntity();
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<void> persistSession({
    required String token,
    required User user,
  }) async {
    await _secureStorage.writeToken(token);
    await _secureStorage.writeUserJson(
      jsonEncode(
        <String, Object?>{
          "id": user.id,
          "email": user.email,
          "display_name": user.displayName,
          "role": user.role,
        },
      ),
    );
  }

  @override
  Future<({String? token, User? user})> restoreSession() async {
    final String? token = await _secureStorage.readToken();
    if (token == null || token.isEmpty) {
      return (token: null, user: null);
    }
    try {
      final User user = await me();
      await persistSession(token: token, user: user);
      return (token: token, user: user);
    } on Object {
      await clearSession();
      return (token: null, user: null);
    }
  }

  @override
  Future<void> clearSession() => _secureStorage.clearSession();

  @override
  Future<String?> readToken() => _secureStorage.readToken();
}
