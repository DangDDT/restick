import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";
import "package:restick/src/core/error/restick_loading_cubit.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";
import "package:restick/src/presentation/widgets/restick_loading_bar.dart";

class RestickScope extends StatelessWidget {
  const RestickScope({
    required this.client,
    required this.child,
    this.showLoadingBar = true,
    super.key,
  });

  final RestickClient client;
  final Widget child;
  final bool showLoadingBar;

  static RestickClient of(BuildContext context) {
    return context.read<RestickClient>();
  }

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final RestickTheme restickTheme =
        theme.extension<RestickTheme>() ??
        RestickTheme.fromColorScheme(theme.colorScheme);
    return RepositoryProvider<RestickClient>.value(
      value: client,
      child: MultiBlocProvider(
        providers: <BlocProvider<dynamic>>[
          BlocProvider<RestickLoadingCubit>.value(value: client.loadingCubit),
          BlocProvider<RestickErrorCubit>.value(value: client.errorCubit),
        ],
        child: Theme(
          data: theme.copyWith(
            extensions: <ThemeExtension<dynamic>>[
              ...theme.extensions.values.where(
                (ThemeExtension<dynamic> extension) =>
                    extension is! RestickTheme,
              ),
              restickTheme,
            ],
          ),
          child: Stack(
            children: <Widget>[
              child,
              if (showLoadingBar)
                const Positioned(
                  top: 0,
                  left: 0,
                  right: 0,
                  child: RestickLoadingBar(),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
