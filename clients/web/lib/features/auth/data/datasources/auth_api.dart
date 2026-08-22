import "package:dio/dio.dart";
import "package:restick_web/features/auth/data/models/user_dto.dart";
import "package:retrofit/retrofit.dart";

part "auth_api.g.dart";

@RestApi()
abstract class AuthApi {
  factory AuthApi(Dio dio, {String? baseUrl}) = _AuthApi;

  @POST("/api/v1/auth/register")
  Future<AuthResponseDto> register(@Body() Map<String, dynamic> body);

  @POST("/api/v1/auth/login")
  Future<AuthResponseDto> login(@Body() Map<String, dynamic> body);

  @GET("/api/v1/me")
  Future<UserDto> me();
}
