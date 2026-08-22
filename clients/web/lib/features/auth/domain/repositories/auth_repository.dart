import "package:restick_web/features/auth/domain/entities/user.dart";

abstract class AuthRepository {
  Future<({String token, User user})> register({
    required String email,
    required String password,
    required String displayName,
  });

  Future<({String token, User user})> login({
    required String email,
    required String password,
  });

  Future<User> me();

  Future<void> persistSession({required String token, required User user});

  Future<({String? token, User? user})> restoreSession();

  Future<void> clearSession();

  Future<String?> readToken();
}
