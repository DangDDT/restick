import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";

class RestickLoadingState extends Equatable {
  const RestickLoadingState({this.activeCount = 0});

  final int activeCount;

  bool get isLoading => activeCount > 0;

  RestickLoadingState copyWith({int? activeCount}) {
    return RestickLoadingState(activeCount: activeCount ?? this.activeCount);
  }

  @override
  List<Object?> get props => <Object?>[activeCount];
}

class RestickLoadingCubit extends Cubit<RestickLoadingState> {
  RestickLoadingCubit() : super(const RestickLoadingState());

  void start() {
    emit(state.copyWith(activeCount: state.activeCount + 1));
  }

  void stop() {
    final int next = state.activeCount > 0 ? state.activeCount - 1 : 0;
    emit(state.copyWith(activeCount: next));
  }
}
