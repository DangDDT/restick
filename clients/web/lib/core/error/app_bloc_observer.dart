import "package:flutter/foundation.dart";
import "package:flutter_bloc/flutter_bloc.dart";

class AppBlocObserver extends BlocObserver {
  @override
  void onChange(BlocBase<dynamic> bloc, Change<dynamic> change) {
    super.onChange(bloc, change);
    debugPrint("${bloc.runtimeType} $change");
  }

  @override
  void onError(BlocBase<dynamic> bloc, Object error, StackTrace stackTrace) {
    debugPrint("${bloc.runtimeType} $error");
    super.onError(bloc, error, stackTrace);
  }
}
