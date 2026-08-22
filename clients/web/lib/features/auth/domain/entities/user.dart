import "package:equatable/equatable.dart";

class User extends Equatable {
  const User({
    required this.id,
    required this.email,
    required this.displayName,
    required this.role,
  });

  final String id;
  final String email;
  final String displayName;
  final String role;

  bool get isAdmin => role == "admin";

  @override
  List<Object?> get props => <Object?>[id, email, displayName, role];
}
