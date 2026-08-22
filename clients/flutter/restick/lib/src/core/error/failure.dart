import "package:equatable/equatable.dart";

sealed class Failure extends Equatable implements Exception {
  const Failure(this.message);

  final String message;

  @override
  String toString() => message;

  @override
  List<Object?> get props => <Object?>[message];
}

final class NetworkFailure extends Failure {
  const NetworkFailure([super.message = "Network request failed"]);
}

final class ServerFailure extends Failure {
  const ServerFailure([super.message = "Server error"]);
}

final class AuthFailure extends Failure {
  const AuthFailure([super.message = "Authentication failed"]);
}

final class CacheFailure extends Failure {
  const CacheFailure([super.message = "Cache error"]);
}

final class RevokedFailure extends Failure {
  const RevokedFailure(this.packId, [super.message = "Pack was revoked"]);

  final String packId;

  @override
  List<Object?> get props => <Object?>[message, packId];
}

final class UnexpectedFailure extends Failure {
  const UnexpectedFailure([super.message = "Unexpected error"]);
}
