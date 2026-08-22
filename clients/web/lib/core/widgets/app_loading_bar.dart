import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/loading_cubit.dart";

class AppLoadingBar extends StatelessWidget {
  const AppLoadingBar({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<LoadingCubit, LoadingState>(
      builder: (BuildContext context, LoadingState state) {
        if (!state.isLoading) {
          return const SizedBox.shrink();
        }
        return const LinearProgressIndicator(minHeight: 2);
      },
    );
  }
}
