import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";

class LoadingState extends Equatable {
  const LoadingState({this.activeCount = 0});

  final int activeCount;

  bool get isLoading => activeCount > 0;

  LoadingState copyWith({int? activeCount}) {
    return LoadingState(activeCount: activeCount ?? this.activeCount);
  }

  @override
  List<Object?> get props => <Object?>[activeCount];
}

class LoadingCubit extends Cubit<LoadingState> {
  LoadingCubit() : super(const LoadingState());

  void start() {
    emit(state.copyWith(activeCount: state.activeCount + 1));
  }

  void stop() {
    final int next = state.activeCount > 0 ? state.activeCount - 1 : 0;
    emit(state.copyWith(activeCount: next));
  }
}
