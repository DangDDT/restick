import "package:flutter/material.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick/src/core/error/restick_loading_cubit.dart";

class RestickLoadingBar extends StatelessWidget {
  const RestickLoadingBar({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<RestickLoadingCubit, RestickLoadingState>(
      builder: (BuildContext context, RestickLoadingState state) {
        if (!state.isLoading) {
          return const SizedBox.shrink();
        }
        return const LinearProgressIndicator(minHeight: 2);
      },
    );
  }
}
