import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:get_it/get_it.dart";
import "package:go_router/go_router.dart";
import "package:restick_web/core/error/error_cubit.dart";
import "package:restick_web/core/error/loading_cubit.dart";
import "package:restick_web/core/theme/app_theme.dart";
import "package:restick_web/core/theme/theme_cubit.dart";
import "package:restick_web/core/widgets/app_loading_bar.dart";
import "package:restick_web/core/widgets/app_toast_host.dart";
import "package:restick_web/features/auth/presentation/cubit/auth_cubit.dart";
import "package:restick_web/features/preview/presentation/cubit/preview_cubit.dart";
import "package:restick_web/l10n/app_localizations.dart";
import "package:responsive_framework/responsive_framework.dart";

class RestickApp extends StatelessWidget {
  const RestickApp({required this.router, super.key});

  final GoRouter router;

  @override
  Widget build(BuildContext context) {
    return MultiBlocProvider(
      providers: [
        BlocProvider<AuthCubit>.value(value: GetIt.instance<AuthCubit>()),
        BlocProvider<ThemeCubit>.value(value: GetIt.instance<ThemeCubit>()),
        BlocProvider<LocaleCubit>.value(value: GetIt.instance<LocaleCubit>()),
        BlocProvider<ErrorCubit>.value(value: GetIt.instance<ErrorCubit>()),
        BlocProvider<LoadingCubit>.value(value: GetIt.instance<LoadingCubit>()),
        BlocProvider<PreviewCubit>.value(value: GetIt.instance<PreviewCubit>()),
      ],
      child: BlocBuilder<ThemeCubit, ThemeMode>(
        builder: (BuildContext context, ThemeMode themeMode) {
          return BlocBuilder<LocaleCubit, LocaleState>(
            builder: (BuildContext context, LocaleState localeState) {
              return MaterialApp.router(
                title: "Restick",
                debugShowCheckedModeBanner: false,
                theme: AppTheme.light(),
                darkTheme: AppTheme.dark(),
                themeMode: themeMode,
                locale: localeState.locale,
                supportedLocales: AppLocalizations.supportedLocales,
                localizationsDelegates: const <LocalizationsDelegate<dynamic>>[
                  AppLocalizations.delegate,
                  ...GlobalMaterialLocalizations.delegates,
                ],
                routerConfig: router,
                builder: (BuildContext context, Widget? child) {
                  // Bridge legacy SDK Material widgets until packages migrate.
                  // ignore: deprecated_member_use
                  return MaterialUiCompatibilityBridge(
                    child: ResponsiveBreakpoints.builder(
                      child: Stack(
                        children: [
                          Column(
                            children: [
                              const AppLoadingBar(),
                              Expanded(child: child ?? const SizedBox.shrink()),
                            ],
                          ),
                          const AppToastHost(),
                        ],
                      ),
                      breakpoints: const [
                        Breakpoint(start: 0, end: 450, name: MOBILE),
                        Breakpoint(start: 451, end: 800, name: TABLET),
                        Breakpoint(start: 801, end: 1920, name: DESKTOP),
                        Breakpoint(start: 1921, end: double.infinity, name: "4K"),
                      ],
                    ),
                  );
                },
              );
            },
          );
        },
      ),
    );
  }
}
