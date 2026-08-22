import "package:equatable/equatable.dart";
import "package:json_annotation/json_annotation.dart";
import "package:restick_web/features/auth/domain/entities/user.dart";

part "user_dto.g.dart";

@JsonSerializable()
class UserDto extends Equatable {
  const UserDto({
    required this.id,
    required this.email,
    required this.displayName,
    required this.role,
  });

  factory UserDto.fromJson(Map<String, dynamic> json) =>
      _$UserDtoFromJson(json);

  final String id;
  final String email;
  @JsonKey(name: "display_name")
  final String displayName;
  final String role;

  Map<String, dynamic> toJson() => _$UserDtoToJson(this);

  User toEntity() {
    return User(
      id: id,
      email: email,
      displayName: displayName,
      role: role,
    );
  }

  @override
  List<Object?> get props => <Object?>[id, email, displayName, role];
}

@JsonSerializable()
class AuthResponseDto extends Equatable {
  const AuthResponseDto({
    required this.accessToken,
    required this.user,
  });

  factory AuthResponseDto.fromJson(Map<String, dynamic> json) =>
      _$AuthResponseDtoFromJson(json);

  @JsonKey(name: "access_token")
  final String accessToken;
  final UserDto user;

  Map<String, dynamic> toJson() => _$AuthResponseDtoToJson(this);

  @override
  List<Object?> get props => <Object?>[accessToken, user];
}
