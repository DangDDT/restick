import "dart:async";
import "dart:math" as math;
import "dart:ui" show PathMetric, lerpDouble;

import "package:flutter/services.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";
import "package:restick/src/presentation/theme/restick_strings.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";

enum _Visual { idle, waiting, downloading, owned, failed }

/// Morphing pill that downloads a pack and shows progress, success and
/// failure. Success/failure are inferred from `pack.downloadStatus`
/// transitions between builds; animations never block taps.
class PackDownloadButton extends StatefulWidget {
  const PackDownloadButton({
    required this.pack,
    required this.onDownload,
    required this.onRemove,
    this.strings = RestickStrings.vi,
    this.compact = false,
    super.key,
  });

  final StickerPack pack;
  final VoidCallback onDownload;
  final VoidCallback onRemove;
  final RestickStrings strings;

  /// Card mode: fixed width, owned is a non-tappable chip. Otherwise the
  /// button fills its parent's width and owned offers [onRemove].
  final bool compact;

  @override
  State<PackDownloadButton> createState() => _PackDownloadButtonState();
}

class _PackDownloadButtonState extends State<PackDownloadButton>
    with TickerProviderStateMixin {
  static const Color _lavender = Color(0xFF7A5AD8);
  static const Color _failedBg = Color(0xFFFFE3EC);
  static const Color _failedFg = Color(0xFFC2185B);
  static const Color _fillColor = Color(0xFFFFB8DA);
  static const Color _hairline = Color(0xFFF3D6E6);
  static const List<double> _shakeKeys = <double>[0, -6, 5, -3, 2, 0];

  // shortcut: last-resort release of the tap lock if no update for this pack
  // ever arrives (e.g. onDownload throws before the repository emits).
  // Fast failures whose downloading+failed coalesce into one rebuild are
  // handled in didUpdateWidget via the new pack instance. Upgrade if the
  // repository gains an explicit "download rejected" signal.
  static const Duration _pendingTimeout = Duration(seconds: 30);

  bool _pending = false;
  bool _pressed = false;
  Timer? _pendingTimer;

  // All controllers are lazy: an idle button in a long list owns none.
  AnimationController? _sheen;
  AnimationController? _success;
  AnimationController? _shake;

  bool get _reduce => MediaQuery.disableAnimationsOf(context);

  _Visual get _visual => switch (widget.pack.downloadStatus) {
    PackDownloadStatus.downloading => _Visual.downloading,
    PackDownloadStatus.downloaded => _Visual.owned,
    _ when _pending => _Visual.waiting,
    PackDownloadStatus.failed => _Visual.failed,
    PackDownloadStatus.remote || PackDownloadStatus.revoked => _Visual.idle,
  };

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_reduce) {
      _success?.dispose();
      _success = null;
      _shake?.dispose();
      _shake = null;
    }
    _syncSheen();
  }

  @override
  void didUpdateWidget(PackDownloadButton oldWidget) {
    super.didUpdateWidget(oldWidget);
    final StickerPack old = oldWidget.pack;
    final PackDownloadStatus status = widget.pack.downloadStatus;
    if (old.id != widget.pack.id) {
      _releasePending();
      _success?.dispose();
      _success = null;
      _shake?.dispose();
      _shake = null;
    } else if (old.downloadStatus != status ||
        // MarketplaceCubit keeps unchanged packs identical, so a new instance
        // while pending means this pack got an update, even if a fast
        // downloading -> failed coalesced into failed -> failed.
        (_pending && !identical(old, widget.pack))) {
      final bool wasBusy =
          _pending || old.downloadStatus == PackDownloadStatus.downloading;
      _releasePending();
      _pressed = false;
      if (wasBusy && status == PackDownloadStatus.downloaded) {
        _celebrate();
      } else if (wasBusy && status == PackDownloadStatus.failed) {
        _fail();
      }
    }
    _syncSheen();
  }

  @override
  void dispose() {
    _pendingTimer?.cancel();
    _sheen?.dispose();
    _success?.dispose();
    _shake?.dispose();
    super.dispose();
  }

  void _releasePending() {
    _pendingTimer?.cancel();
    _pendingTimer = null;
    _pending = false;
  }

  void _download() {
    if (_pending) {
      return;
    }
    unawaited(HapticFeedback.selectionClick());
    setState(() {
      _pending = true;
      _syncSheen();
    });
    _pendingTimer = Timer(_pendingTimeout, () {
      if (mounted) {
        setState(() {
          _releasePending();
          _syncSheen();
        });
      }
    });
    widget.onDownload();
  }

  void _syncSheen() {
    final _Visual visual = _visual;
    final bool want =
        !_reduce &&
        (visual == _Visual.waiting || visual == _Visual.downloading);
    if (want && _sheen == null) {
      _sheen = AnimationController(
        vsync: this,
        duration: const Duration(milliseconds: 1100),
      )..repeat();
    } else if (!want && _sheen != null) {
      _sheen!.dispose();
      _sheen = null;
    }
  }

  /// One-shot controller that clears its field and disposes itself when done.
  AnimationController _oneShot(int millis, VoidCallback clear) {
    final AnimationController controller = AnimationController(
      vsync: this,
      duration: Duration(milliseconds: millis),
    );
    controller.addStatusListener((AnimationStatus status) {
      if (status != AnimationStatus.completed) {
        return;
      }
      // Disposing inside the ticker's own callback is unsafe; defer a frame.
      WidgetsBinding.instance.addPostFrameCallback((_) => controller.dispose());
      if (mounted) {
        setState(clear);
      }
    });
    unawaited(controller.forward());
    return controller;
  }

  void _celebrate() {
    unawaited(HapticFeedback.lightImpact());
    if (_reduce) {
      return;
    }
    _success?.dispose();
    _success = _oneShot(900, () => _success = null);
  }

  void _fail() {
    unawaited(HapticFeedback.mediumImpact());
    if (_reduce) {
      return;
    }
    _shake?.dispose();
    _shake = _oneShot(450, () => _shake = null);
  }

  static double _shakeOffset(double t) {
    final double x = t * (_shakeKeys.length - 1);
    final int i = x.floor().clamp(0, _shakeKeys.length - 2);
    return lerpDouble(_shakeKeys[i], _shakeKeys[i + 1], x - i)!;
  }

  @override
  Widget build(BuildContext context) {
    final RestickStrings s = widget.strings;
    final RestickTheme theme = RestickTheme.of(context);
    final StickerPack pack = widget.pack;
    final _Visual visual = _visual;
    final bool reduce = _reduce;
    final Duration duration = reduce ? Duration.zero : RestickMotion.routine;
    final int percent = (pack.downloadProgress.clamp(0, 1) * 100).round();
    final bool busy =
        visual == _Visual.waiting || visual == _Visual.downloading;
    final bool removable = visual == _Visual.owned && !widget.compact;
    final String idleLabel =
        widget.compact ? s.download : s.downloadAllLabel(pack.stickerCount);

    final (Color background, Color foreground) = switch (visual) {
      _Visual.idle => (RestickBrand.deepPink, Colors.white),
      _Visual.waiting ||
      _Visual.downloading => (Colors.white, theme.ink),
      _Visual.owned => (Colors.white, theme.brand == RestickBrand.purple ? _lavender : theme.brand),
      _Visual.failed => (_failedBg, _failedFg),
    };
    final VoidCallback? onTap = switch (visual) {
      _Visual.idle || _Visual.failed => _download,
      _Visual.owned => removable ? widget.onRemove : null,
      _Visual.waiting || _Visual.downloading => null,
    };
    final String semanticLabel = switch (visual) {
      _Visual.idle => "$idleLabel, ${pack.name}",
      _Visual.waiting => "${s.waiting}, ${pack.name}",
      _Visual.downloading => s.a11yDownloading(pack.name, percent),
      _Visual.owned =>
        removable
            ? "${s.a11yDone(pack.name)}, ${s.remove}"
            : s.a11yDone(pack.name),
      _Visual.failed => s.a11yFailed(pack.name),
    };
    final Widget label = switch (visual) {
      _Visual.idle => Text(idleLabel),
      _Visual.waiting => Text(s.waiting),
      _Visual.downloading => Text(
        widget.compact ? "$percent%" : "${s.downloading} · $percent%",
      ),
      _Visual.failed => Text(s.retry),
      _Visual.owned => Row(
        mainAxisSize: MainAxisSize.min,
        spacing: 6,
        children: <Widget>[
          _tick(foreground),
          Text(removable ? "${s.owned} · ${s.remove}" : s.owned),
        ],
      ),
    };

    final double height = widget.compact ? 36 : 48;
    final Widget pill = AnimatedContainer(
      key: const ValueKey<String>("restick.download.pill"),
      duration: duration,
      curve: RestickMotion.ease,
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(height / 2),
        border: Border.all(
          color: background == Colors.white ? _hairline : background,
        ),
      ),
      child: Stack(
        fit: StackFit.expand,
        children: <Widget>[
          if (busy)
            TweenAnimationBuilder<double>(
              tween: Tween<double>(
                end:
                    visual == _Visual.waiting
                        ? 0
                        : pack.downloadProgress.clamp(0, 1),
              ),
              duration: duration,
              curve: RestickMotion.ease,
              builder:
                  (BuildContext context, double value, Widget? child) =>
                      Transform.scale(
                        key: const ValueKey<String>("restick.download.fill"),
                        scaleX: value,
                        alignment: Alignment.centerLeft,
                        child: child,
                      ),
              child: ColoredBox(color: theme.brandSoft == RestickBrand.purpleSoft ? _fillColor : theme.brandSoft),
            ),
          if (busy && _sheen != null) _sheenBand(_sheen!),
          Center(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: FittedBox(
                fit: BoxFit.scaleDown,
                child: AnimatedDefaultTextStyle(
                  duration: duration,
                  style: RestickBrand.text(
                    size: widget.compact ? 13 : 15,
                    weight: FontWeight.w700,
                    color: foreground,
                  ),
                  child: AnimatedSwitcher(
                    duration: duration,
                    switchInCurve: RestickMotion.ease,
                    child: KeyedSubtree(
                      key: ValueKey<_Visual>(visual),
                      child: label,
                    ),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );

    // The keyed hit target is a sibling overlay: its key changes with the
    // action, and keying the pill's ancestor would remount it mid-morph.
    final Widget hitTarget = Positioned.fill(
      child: GestureDetector(
        key:
            removable
                ? ValueKey<String>("marketplace.pack.remove.${pack.id}")
                : visual == _Visual.owned
                ? null
                : ValueKey<String>("marketplace.pack.download.${pack.id}"),
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        onTapDown:
            onTap == null ? null : (_) => setState(() => _pressed = true),
        onTapUp: onTap == null ? null : (_) => setState(() => _pressed = false),
        onTapCancel:
            onTap == null ? null : () => setState(() => _pressed = false),
      ),
    );
    final Widget button = AnimatedScale(
      key: const ValueKey<String>("restick.download.press"),
      scale: _pressed && onTap != null ? 0.96 : 1,
      duration: reduce ? Duration.zero : RestickMotion.fast,
      curve: RestickMotion.ease,
      // Size lives outside the AnimatedContainer: it never animates, and an
      // infinite width cannot be tweened.
      child: SizedBox(
        width: widget.compact ? 112 : double.infinity,
        height: height,
        child: pill,
      ),
    );

    return Semantics(
      container: true,
      liveRegion: true,
      button: onTap != null,
      label: semanticLabel,
      onTap: onTap,
      child: ExcludeSemantics(
        child: Stack(
          clipBehavior: Clip.none,
          children: <Widget>[
            RepaintBoundary(
              child: AnimatedBuilder(
                animation: _shake ?? kAlwaysCompleteAnimation,
                builder:
                    (BuildContext context, Widget? child) =>
                        Transform.translate(
                          key: const ValueKey<String>("restick.download.shake"),
                          offset: Offset(
                            _shake == null ? 0 : _shakeOffset(_shake!.value),
                            0,
                          ),
                          child: child,
                        ),
                child: button,
              ),
            ),
            hitTarget,
            if (_success != null)
              Positioned.fill(
                child: IgnorePointer(
                  child: RepaintBoundary(
                    child: CustomPaint(
                      key: const ValueKey<String>("restick.download.burst"),
                      painter: _BurstPainter(_success!, theme.brand, theme.brandSoft),
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _tick(Color color) {
    final AnimationController? success = _success;
    Widget paint(double progress) => CustomPaint(
      key: const ValueKey<String>("restick.download.tick"),
      size: const Size.square(16),
      painter: TickPainter(progress: progress, color: color),
    );
    if (success == null) {
      return paint(1);
    }
    // 100ms delay, 400ms draw within the 900ms success run.
    const Curve draw = Interval(0.11, 0.55, curve: RestickMotion.ease);
    return AnimatedBuilder(
      animation: success,
      builder:
          (BuildContext context, Widget? _) =>
              paint(draw.transform(success.value)),
    );
  }

  Widget _sheenBand(Animation<double> sheen) {
    return IgnorePointer(
      child: AnimatedBuilder(
        animation: sheen,
        builder:
            (BuildContext context, Widget? child) => FractionalTranslation(
              translation: Offset(sheen.value * 2 - 1, 0),
              child: child,
            ),
        child: const DecoratedBox(
          decoration: BoxDecoration(
            gradient: LinearGradient(
              colors: <Color>[
                Color(0x00F58BC0),
                Color(0x55F58BC0),
                Color(0x00F58BC0),
              ],
              stops: <double>[0.3, 0.5, 0.7],
            ),
          ),
        ),
      ),
    );
  }
}

/// Check mark drawn along its path up to [progress] (0..1).
class TickPainter extends CustomPainter {
  const TickPainter({required this.progress, required this.color});

  final double progress;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    if (progress <= 0) {
      return;
    }
    final double u = size.shortestSide / 24;
    final Path tick =
        Path()
          ..moveTo(5 * u, 12.5 * u)
          ..lineTo(9.5 * u, 17 * u)
          ..lineTo(19 * u, 7.5 * u);
    final Path drawn = Path();
    for (final PathMetric metric in tick.computeMetrics()) {
      drawn.addPath(
        metric.extractPath(0, metric.length * progress),
        Offset.zero,
      );
    }
    canvas.drawPath(
      drawn,
      Paint()
        ..color = color
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2.6 * u
        ..strokeCap = StrokeCap.round
        ..strokeJoin = StrokeJoin.round,
    );
  }

  @override
  bool shouldRepaint(TickPainter oldDelegate) =>
      oldDelegate.progress != progress || oldDelegate.color != color;
}

/// Ring pulse around the pill plus a one-shot burst of hearts, stars and
/// sparkles, all driven by the single success controller.
class _BurstPainter extends CustomPainter {
  _BurstPainter(this.animation, this.brand, this.brandSoft) : super(repaint: animation);

  final Animation<double> animation;
  final Color brand;
  final Color brandSoft;

  Color get _ring => brand;
  static const List<Color> _colors = <Color>[
    Color(0xFFE6197F),
    Color(0xFFFFC83D),
    Color(0xFFFF8FC7),
  ];

  // Shapes in a 64x64 box, from the motion prototype.
  static final Path _heart =
      Path()
        ..moveTo(32, 57)
        ..cubicTo(7, 39, 5, 22, 17, 14)
        ..cubicTo(24, 10, 29, 14, 32, 19)
        ..cubicTo(35, 14, 40, 10, 47, 14)
        ..cubicTo(59, 22, 57, 39, 32, 57)
        ..close();
  static final Path _star =
      Path()..addPolygon(const <Offset>[
        Offset(32, 4),
        Offset(40.5, 21.5),
        Offset(59.5, 24.1),
        Offset(45.7, 37.5),
        Offset(49, 56.5),
        Offset(32, 47.5),
        Offset(15, 56.5),
        Offset(18.3, 37.5),
        Offset(4.5, 24.1),
        Offset(23.5, 21.5),
      ], true);
  static final Path _spark =
      Path()
        ..moveTo(32, 4)
        ..cubicTo(34, 18, 40, 24, 56, 32)
        ..cubicTo(40, 40, 34, 46, 32, 60)
        ..cubicTo(30, 46, 24, 40, 8, 32)
        ..cubicTo(24, 24, 30, 18, 32, 4)
        ..close();
  static final List<Path> _shapes = <Path>[_heart, _star, _spark];
  static const List<int> _order = <int>[0, 1, 2, 0, 2, 1, 0, 2];

  @override
  void paint(Canvas canvas, Size size) {
    final double v = animation.value;

    // Ring: 0-700ms of the 900ms run.
    final double ringT = RestickMotion.ease.transform((v / 0.78).clamp(0, 1));
    if (ringT < 1) {
      final double spread = 14 * ringT;
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          (Offset.zero & size).inflate(spread / 2),
          Radius.circular(size.height / 2 + spread / 2),
        ),
        Paint()
          ..style = PaintingStyle.stroke
          ..strokeWidth = math.max(0.5, spread)
          ..color = _ring.withValues(alpha: 0.67 * (1 - ringT)),
      );
    }

    // Burst: 0-800ms, eight particles fanning upward.
    final double t = RestickMotion.ease.transform((v / 0.89).clamp(0, 1));
    if (t >= 1) {
      return;
    }
    final Offset origin = size.center(Offset.zero);
    for (int i = 0; i < _order.length; i++) {
      final double angle = -math.pi * 0.95 + math.pi * 0.9 * i / 7;
      final double distance = 46 + (i * 37 % 30);
      final double spin = ((i * 53 % 120) - 60) * math.pi / 180;
      final double scale = (0.4 + 0.6 * t) * 22 / 64;
      canvas
        ..save()
        ..translate(
          origin.dx + math.cos(angle) * distance * t,
          origin.dy + math.sin(angle) * distance * t,
        )
        ..rotate(spin * t)
        ..scale(scale)
        ..translate(-32, -32)
        ..drawPath(
          _shapes[_order[i]],
          Paint()..color = (_order[i] == 2 && brandSoft != RestickBrand.purpleSoft ? brandSoft : _colors[_order[i]]).withValues(alpha: 1 - t),
        )
        ..restore();
    }
  }

  @override
  bool shouldRepaint(_BurstPainter oldDelegate) =>
      oldDelegate.animation != animation || oldDelegate.brand != brand || oldDelegate.brandSoft != brandSoft;
}
