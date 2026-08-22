import "package:equatable/equatable.dart";
import "package:flutter/foundation.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/auth/domain/entities/user.dart";
import "package:restick_web/features/auth/domain/repositories/auth_repository.dart";

class AuthState extends Equatable {
  const AuthState({
    this.isReady = false,
    this.token,
    this.user,
    this.isBusy = false,
    this.errorMessage,
  });

  final bool isReady;
  final String? token;
  final User? user;
  final bool isBusy;
  final String? errorMessage;

  bool get isSignedIn => token != null && user != null;
  bool get isAdmin => user?.isAdmin ?? false;

  AuthState copyWith({
    bool? isReady,
    String? token,
    User? user,
    bool? isBusy,
    String? errorMessage,
    bool clearSession = false,
    bool clearError = false,
  }) {
    return AuthState(
      isReady: isReady ?? this.isReady,
      token: clearSession ? null : (token ?? this.token),
      user: clearSession ? null : (user ?? this.user),
      isBusy: isBusy ?? this.isBusy,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
    );
  }

  @override
  List<Object?> get props =>
      <Object?>[isReady, token, user, isBusy, errorMessage];
}

class AuthCubit extends Cubit<AuthState> {
  AuthCubit({required AuthRepository authRepository})
      : _authRepository = authRepository,
        super(const AuthState()) {
    _listenable = _AuthListenable(this);
  }

  final AuthRepository _authRepository;
  late final _AuthListenable _listenable;

  Listenable get listenable => _listenable;

  Future<void> restore() async {
    final ({String? token, User? user}) session =
        await _authRepository.restoreSession();
    emit(
      state.copyWith(
        isReady: true,
        token: session.token,
        user: session.user,
        clearSession: session.token == null,
        clearError: true,
      ),
    );
  }

  Future<void> login({
    required String email,
    required String password,
  }) async {
    emit(state.copyWith(isBusy: true, clearError: true));
    try {
      final ({String token, User user}) result = await _authRepository.login(
        email: email,
        password: password,
      );
      emit(
        state.copyWith(
          isBusy: false,
          token: result.token,
          user: result.user,
          clearError: true,
        ),
      );
    } on Failure catch (failure) {
      emit(state.copyWith(isBusy: false, errorMessage: failure.message));
      rethrow;
    }
  }

  Future<void> register({
    required String email,
    required String password,
    required String displayName,
  }) async {
    emit(state.copyWith(isBusy: true, clearError: true));
    try {
      final ({String token, User user}) result = await _authRepository.register(
        email: email,
        password: password,
        displayName: displayName,
      );
      emit(
        state.copyWith(
          isBusy: false,
          token: result.token,
          user: result.user,
          clearError: true,
        ),
      );
    } on Failure catch (failure) {
      emit(state.copyWith(isBusy: false, errorMessage: failure.message));
      rethrow;
    }
  }

  Future<void> logout() async {
    await _authRepository.clearSession();
    emit(state.copyWith(clearSession: true, clearError: true, isReady: true));
  }
}

class _AuthListenable extends ChangeNotifier {
  _AuthListenable(this._cubit) {
    _cubit.stream.listen((_) => notifyListeners());
  }

  final AuthCubit _cubit;
}
