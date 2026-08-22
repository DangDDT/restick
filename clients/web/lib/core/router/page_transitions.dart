import "package:material_ui/material_ui.dart";
import "package:go_router/go_router.dart";

CustomTransitionPage<void> buildAppPage({
  required LocalKey key,
  required Widget child,
}) {
  return CustomTransitionPage<void>(
    key: key,
    child: child,
    transitionDuration: const Duration(milliseconds: 240),
    reverseTransitionDuration: const Duration(milliseconds: 200),
    transitionsBuilder: (
      BuildContext context,
      Animation<double> animation,
      Animation<double> secondaryAnimation,
      Widget child,
    ) {
      final CurvedAnimation primary = CurvedAnimation(
        parent: animation,
        curve: Curves.easeOutCubic,
        reverseCurve: Curves.easeInCubic,
      );
      final Animation<double> outgoingOpacity = Tween<double>(
        begin: 1,
        end: 0.88,
      ).animate(
        CurvedAnimation(
          parent: secondaryAnimation,
          curve: Curves.easeOut,
        ),
      );
      return FadeTransition(
        opacity: primary,
        child: FadeTransition(
          opacity: outgoingOpacity,
          child: SlideTransition(
            position: Tween<Offset>(
              begin: const Offset(0.012, 0),
              end: Offset.zero,
            ).animate(primary),
            child: child,
          ),
        ),
      );
    },
  );
}
