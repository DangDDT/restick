import "package:material_ui/material_ui.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";

/// Pastel placeholder block with a soft shimmer; idle when hidden/reduced.
class PackSkeleton extends StatefulWidget {
  const PackSkeleton({
    this.width,
    this.height,
    this.color,
    this.radius = 20,
    super.key,
  });

  final double? width;
  final double? height;
  final Color? color;
  final double radius;

  @override
  State<PackSkeleton> createState() => _PackSkeletonState();
}

class _PackSkeletonState extends State<PackSkeleton>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 1400),
  );

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (MediaQuery.disableAnimationsOf(context)) {
      _controller.stop();
    } else if (!_controller.isAnimating) {
      _controller.repeat();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final RestickTheme theme = RestickTheme.of(context);
    final Color base = theme.brandSoft == RestickBrand.purpleSoft
        ? widget.color ?? RestickBrand.packPastels.first : theme.brandSoft;
    final Color glow = Color.lerp(base, const Color(0xFFFFFFFF), 0.6)!;
    return RepaintBoundary(
      child: AnimatedBuilder(
        animation: _controller,
        builder: (BuildContext context, Widget? _) {
          final double t = _controller.value * 3 - 1;
          return Container(
            width: widget.width,
            height: widget.height,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(widget.radius),
              gradient: LinearGradient(
                begin: Alignment(t - 1, 0),
                end: Alignment(t + 1, 0),
                colors: <Color>[base, glow, base],
              ),
            ),
          );
        },
      ),
    );
  }
}
