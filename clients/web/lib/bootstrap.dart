import "package:flutter/widgets.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:get_it/get_it.dart";
import "package:restick_web/app.dart";
import "package:restick_web/core/config/env_config.dart";
import "package:restick_web/core/di/injection.dart";
import "package:restick_web/core/error/app_bloc_observer.dart";
import "package:restick_web/core/router/app_router.dart";
import "package:restick_web/core/theme/theme_cubit.dart";
import "package:restick_web/features/auth/presentation/cubit/auth_cubit.dart";

Future<void> bootstrap() async {
  WidgetsFlutterBinding.ensureInitialized();
  Bloc.observer = AppBlocObserver();
  final EnvConfig config = EnvConfig.load();
  await configureDependencies(config);
  await GetIt.instance<ThemeCubit>().restore();
  await GetIt.instance<LocaleCubit>().restore();
  await GetIt.instance<AuthCubit>().restore();
  final router = createAppRouter(GetIt.instance<AuthCubit>());
  runApp(RestickApp(router: router));
}
