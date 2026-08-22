import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick/src/core/error/failure.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";

class RestickBlocObserver extends BlocObserver {
  RestickBlocObserver({required this.errorCubit});

  final RestickErrorCubit errorCubit;

  @override
  void onError(BlocBase<dynamic> bloc, Object error, StackTrace stackTrace) {
    if (error is Failure) {
      errorCubit.report(error);
    } else {
      errorCubit.report(UnexpectedFailure(error.toString()));
    }
    super.onError(bloc, error, stackTrace);
  }
}
