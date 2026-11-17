import "dart:math" as math;
import "dart:async";

import "package:material_ui/material_ui.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";

/// A completion transition only: owned mounts and metadata refreshes stay still.
class PackSuccessStamp extends StatefulWidget {
  const PackSuccessStamp({required this.pack, required this.child, super.key});

  final StickerPack pack;
  final Widget child;

  @override
  State<PackSuccessStamp> createState() => _PackSuccessStampState();
}

class _PackSuccessStampState extends State<PackSuccessStamp> with TickerProviderStateMixin {
  AnimationController? _stamp;

  void _stop() {
    _stamp?.dispose();
    _stamp = null;
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (MediaQuery.disableAnimationsOf(context)) _stop();
  }

  @override
  void didUpdateWidget(PackSuccessStamp oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.pack.id != widget.pack.id || !widget.pack.isDownloaded) {
      _stop();
    } else if (oldWidget.pack.downloadStatus == PackDownloadStatus.downloading &&
        !MediaQuery.disableAnimationsOf(context)) {
      _stop();
      final AnimationController stamp = AnimationController(vsync: this, duration: RestickMotion.enter);
      _stamp = stamp;
      stamp.addStatusListener((AnimationStatus status) {
        if (status == AnimationStatus.completed) {
          setState(() => _stamp = null);
          WidgetsBinding.instance.addPostFrameCallback((_) => stamp.dispose());
        }
      });
      unawaited(stamp.forward());
    }
  }

  @override
  void dispose() {
    _stop();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AnimatedBuilder(
    animation: _stamp ?? kAlwaysCompleteAnimation,
    builder: (BuildContext context, Widget? child) => Transform.scale(
      key: ValueKey<String>("restick.card.stamp.${widget.pack.id}"),
      scale: 1 + math.sin((_stamp?.value ?? 1) * math.pi) * 0.035,
      child: child,
    ),
    child: widget.child,
  );
}
