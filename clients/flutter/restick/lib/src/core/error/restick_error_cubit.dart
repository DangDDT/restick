import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick/src/core/error/failure.dart";

class RestickErrorState extends Equatable {
  const RestickErrorState({this.failure});

  final Failure? failure;

  @override
  List<Object?> get props => <Object?>[failure];
}

class RestickErrorCubit extends Cubit<RestickErrorState> {
  RestickErrorCubit({this.onFailure}) : super(const RestickErrorState());

  final void Function(Failure failure)? onFailure;

  void report(Failure failure) {
    emit(RestickErrorState(failure: failure));
    onFailure?.call(failure);
  }

  void clear() {
    emit(const RestickErrorState());
  }
}
